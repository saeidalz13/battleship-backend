package api

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	md "github.com/saeidalz13/battleship-backend/models"
)

var GlobalSessions = make(map[string]*Session)

const (
	PingInterval time.Duration = time.Second * 15
	GracePeriod  time.Duration = time.Minute * 3
)

type Session struct {
	ID         string
	Conn       *websocket.Conn
	Game       *md.Game
	Player     *md.Player
	GraceTimer <-chan time.Time
	PingTicker *time.Ticker
	mu         sync.Mutex
}

func NewSession(conn *websocket.Conn) *Session {
	return &Session{
		Conn: conn,
	}
}

func (s *Session) manageSession() {
	conn := s.Conn
	defer func() {
		s.Conn.Close()
		log.Println("connection closed:", conn.RemoteAddr().String())
	}()

sessionLoop:
	for {
		// A WebSocket frame can be one of 6 types: text=1, binary=2, ping=9, pong=10, close=8 and continuation=0
		// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
		retries := 0
		_, payload, err := s.Conn.ReadMessage()
		if err != nil {
			switch IdentifyWsErrorAction(err) {
			case ConnLoopCodeRetry:
				if retries < maxWriteWsRetries {
					retries++
					log.Printf("failed to read from ws conn [%s]; retrying... (retry no. %d)\n", conn.RemoteAddr().String(), retries)
					time.Sleep(time.Duration(retries*backOffFactor) * time.Second)
					continue sessionLoop

				} else {
					break sessionLoop
				}

			case ConnLoopCodeBreak:
				log.Printf("break ws conn loop [%s] due to: %s\n", conn.RemoteAddr().String(), err)
				break sessionLoop

			case ConnLoopCodeContinue:
				continue sessionLoop
			}
		}

		// the incoming message must be of type json containing the field "code"
		// which would allow us to determine what action is required
		// In case of absence of "code" field, the message is invalid
		var signal md.Signal
		if err := json.Unmarshal(payload, &signal); err != nil {
			log.Println("incoming msg does not contain 'code':", err)
			resp := md.NewMessage[md.NoPayload](md.CodeSignalAbsent)
			resp.AddError("incoming req payload must contain 'code' field", "")

			switch WriteJsonWithRetry(conn, resp) {
			case ConnLoopCodeBreak:
				break sessionLoop
			default:
				continue sessionLoop
			}
		}

		// This is where we choose the action based on the code in incoming json
		switch signal.Code {

		case md.CodeCreateGame:
			req := NewRequest(conn)
			resp := req.HandleCreateGame()

			switch WriteJsonWithRetry(conn, resp) {
			case ConnLoopCodeBreak:
				break sessionLoop
			default:
				continue sessionLoop
			}

		case md.CodeAttack:
			req := NewRequest(nil, payload)
			// response will have the IsTurn field of attacker
			resp, defender := req.HandleAttack()

			if resp.Error.ErrorDetails != "" {
				switch WriteJsonWithRetry(conn, resp) {
				case ConnLoopCodeBreak:
					break sessionLoop
				default:
					continue sessionLoop
				}
			}

			done := make(chan bool, 2)
			go func() {
				// attacker turn is set to false
				resp.Payload.IsTurn = false
				switch WriteJsonWithRetry(conn, resp) {
				case ConnLoopCodeBreak:
					done <- false
				default:
					done <- true
				}
			}()
			go func() {
				// defender turn is set to true
				resp.Payload.IsTurn = true
				switch WriteJsonWithRetry(defender.WsConn, resp) {
				case ConnLoopCodeBreak:
					done <- false
				default:
					done <- true
				}
			}()

			// Wait for the results and break if any is false
			for i := 0; i < 2; i++ {
				if !<-done {
					break
				}
			}

			// If this attack caused the game to end.
			// Both attacker and defender will get a end game
			// message indicating if they lost or won
			if defender.MatchStatus == md.PlayerMatchStatusLost {
				currentGame := defender.CurrentGame

				done := make(chan bool, 2)
				go func() {
					// Sending victory code to the attacker
					respAttacker := md.NewMessage[md.RespEndGame](md.CodeEndGame)
					respAttacker.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusWon})
					switch WriteJsonWithRetry(conn, respAttacker) {
					case ConnLoopCodeBreak:
						done <- false
					default:
						done <- true
					}
				}()

				go func() {
					// Sending failure code to the defender
					respDefender := md.NewMessage[md.RespEndGame](md.CodeEndGame)
					respDefender.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusLost})
					_ = WriteJsonWithRetry(defender.WsConn, respDefender)
				}()

				GlobalGameManager.endGameSignal <- md.NewEndGameSignal(md.ManageGameCodeSuccess, currentGame.Uuid, "")
			}

		case md.CodeReady:
			req := NewRequest(nil, payload)
			resp, game := req.HandleReadyPlayer()

			if resp.Error.ErrorDetails != "" {
				switch WriteJsonWithRetry(conn, resp) {
				case ConnLoopCodeBreak:
					break sessionLoop
				default:
					continue sessionLoop
				}
			}

			switch WriteJsonWithRetry(conn, resp) {
			case ConnLoopCodeBreak:
				break sessionLoop
			case ConnLoopCodeContinue:
				continue sessionLoop
			case ConnLoopCodePassThrough:
			}

			if game.HostPlayer.IsReady && game.JoinPlayer.IsReady {
				respStartGame := md.NewMessage[md.NoPayload](md.CodeStartGame)
				switch SendMsgToBothPlayers(game, &respStartGame, &respStartGame) {
				case ConnLoopCodeBreak:
					break sessionLoop
				case ConnLoopCodePassThrough:
				}
			}

		case md.CodeJoinGame:
			req := NewRequest(conn, payload)
			resp, game := req.HandleJoinPlayer()

			switch WriteJsonWithRetry(conn, resp) {
			case ConnLoopCodeBreak:
				// delete(s.Players, game.JoinPlayer.Uuid)
				game.JoinPlayer = nil
				break sessionLoop
			case ConnLoopCodeContinue:
				// delete(s.Players, game.JoinPlayer.Uuid)
				game.JoinPlayer = nil
				continue sessionLoop
			case ConnLoopCodePassThrough:
			}

			// If the second playerd joined successfully, then `CodeSelectGrid`
			// is sent to both players as an indication of grid selection
			if resp.Error.ErrorDetails == "" {
				readyResp := md.NewMessage[md.NoPayload](md.CodeSelectGrid)
				switch SendMsgToBothPlayers(game, &readyResp, &readyResp) {
				case ConnLoopCodeBreak:
					break sessionLoop
				case ConnLoopCodePassThrough:
				}
			}

		default:
			respInvalidSignal := md.NewMessage[md.NoPayload](md.CodeInvalidSignal)
			respInvalidSignal.AddError("", "invalid code in the incoming payload")
			switch WriteJsonWithRetry(conn, respInvalidSignal) {
			case ConnLoopCodeBreak:
				break sessionLoop
			default:
				continue sessionLoop
			}
		}
	}
}

func (s *Session) WaitAndClose() {
	s.mu.Lock()
	s.PingTicker = time.NewTicker(PingInterval)
	defer s.PingTicker.Stop()

	s.GraceTimer = time.After(GracePeriod)
	s.mu.Unlock()

	for {
		select {
		case <-s.GraceTimer:
			s.mu.Lock()

			delete(GlobalSessions, s.ID)
			s.Conn.Close()

			s.mu.Unlock()
			return

		case <-s.PingTicker.C:
			if err := s.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("ping error for reconnection of session %s, player %s, error:%s\n", s.ID, s.Player.Uuid, err)
			}
		}
	}
}
