package api

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	md "github.com/saeidalz13/battleship-backend/models"
)

var GlobalSessionManager = NewSessionManager()

type OtherSessionMsg struct {
	ID       string
	GameUuid string
	Payload  interface{}
}

func NewOtherSessionMsg(id string, gameUuid string, p interface{}) OtherSessionMsg {
	return OtherSessionMsg{
		ID:       id,
		GameUuid: gameUuid,
		Payload:  p,
	}
}

type SessionManager struct {
	Sessions        map[string]*Session
	otherSessionMsg chan OtherSessionMsg
	mu              sync.Mutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		Sessions:        make(map[string]*Session),
		otherSessionMsg: make(chan OtherSessionMsg),
	}
}

func (sm *SessionManager) ManageOtherSessionMsg() {
	for {
		msg := <-sm.otherSessionMsg

		sm.mu.Lock()
		otherSession := GlobalSessionManager.Sessions[msg.ID]

		if otherSession.Game.Uuid != msg.GameUuid {
			panic("other session msg game is not the same as game uuid; this error should never happen")
		}

		switch WriteJsonWithRetry(otherSession.Conn, msg.Payload) {
		case ConnLoopAbnormalClosureRetry:
			switch otherSession.waitAndClose() {
			case ConnLoopCodeBreak:
				otherSession.terminateSession()

			case ConnLoopCodeContinue:
			}

		case ConnLoopCodeBreak:
			otherSession.terminateSession()

		case ConnLoopCodePassThrough:
		}

		sm.mu.Unlock()
	}
}

const (
	PingInterval time.Duration = time.Second * 15
	GracePeriod  time.Duration = time.Minute * 3
)

type Session struct {
	// Player-related info
	ID         string
	Conn       *websocket.Conn
	Game       *md.Game
	Player     *md.Player
	GraceTimer *time.Timer

	// To send signal for player reconnection
	StopRetry chan struct{}
	mu        sync.Mutex
}

func NewSession(conn *websocket.Conn, sessionID string) *Session {
	return &Session{
		ID:        sessionID,
		Conn:      conn,
		StopRetry: make(chan struct{}),
	}
}

func (s *Session) run() {
	defer s.terminateSession()

sessionLoop:
	for {
		conn := s.Conn
		// A WebSocket frame can be one of 6 types: text=1, binary=2, ping=9, pong=10, close=8 and continuation=0
		// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
		retries := 0
		_, payload, err := s.Conn.ReadMessage()
		if err != nil {
			switch IdentifyWsErrorAction(err) {
			case ConnLoopAbnormalClosureRetry:
				switch s.waitAndClose() {
				case ConnLoopCodeBreak:
					break sessionLoop
				case ConnLoopCodeContinue:
					continue sessionLoop
				}

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
			case ConnLoopAbnormalClosureRetry:
				switch s.waitAndClose() {
				case ConnLoopCodeBreak:
					break sessionLoop
				case ConnLoopCodeContinue:
					continue sessionLoop
				}

			case ConnLoopCodeBreak:
				break sessionLoop
			default:
				continue sessionLoop
			}
		}

		// This is where we choose the action based on the code in incoming json
		switch signal.Code {

		case md.CodeCreateGame:
			req := NewRequest(conn, s)
			resp := req.HandleCreateGame()

			switch WriteJsonWithRetry(conn, resp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.waitAndClose() {
				case ConnLoopCodeBreak:
					break sessionLoop
				case ConnLoopCodeContinue:
					continue sessionLoop
				}
			case ConnLoopCodeBreak:
				break sessionLoop
			default:
				continue sessionLoop
			}

		case md.CodeAttack:
			req := NewRequest(nil, s, payload)
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

			// attacker turn is set to false
			resp.Payload.IsTurn = false
			switch WriteJsonWithRetry(conn, resp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.waitAndClose() {
				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodeContinue:
				}

			case ConnLoopCodeBreak:
				break sessionLoop

			case ConnLoopCodePassThrough:
			}

			// defender turn is set to true
			resp.Payload.IsTurn = true
			GlobalSessionManager.otherSessionMsg <- NewOtherSessionMsg(defender.SessionID, s.Game.Uuid, resp)

			// If this attack caused the game to end.
			// Both attacker and defender will get a end game
			// message indicating if they lost or won
			if defender.MatchStatus == md.PlayerMatchStatusLost {
				// Sending victory code to the attacker
				respAttacker := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respAttacker.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusWon})
				switch WriteJsonWithRetry(conn, respAttacker) {
				case ConnLoopAbnormalClosureRetry:
					switch s.waitAndClose() {
					case ConnLoopCodeBreak:
						break sessionLoop

					case ConnLoopCodeContinue:
					}

				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodePassThrough:
				}

				// Sending failure code to the defender
				respDefender := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respDefender.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusLost})
				GlobalSessionManager.otherSessionMsg <- NewOtherSessionMsg(defender.SessionID, s.Game.Uuid, respDefender)

				// Tell the game manager to get rid of this game in map
				GlobalGameManager.EndGameSignal <- s.Game.Uuid
			}

		case md.CodeReady:
			req := NewRequest(nil, s, payload)
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
			case ConnLoopAbnormalClosureRetry:
				switch s.waitAndClose() {
				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodeContinue:
				}

			case ConnLoopCodeBreak:
				break sessionLoop

			case ConnLoopCodePassThrough:
			}

			if game.HostPlayer.IsReady && game.JoinPlayer.IsReady {
				respStartGame := md.NewMessage[md.NoPayload](md.CodeStartGame)
				switch WriteJsonWithRetry(conn, respStartGame) {
				case ConnLoopAbnormalClosureRetry:
					switch s.waitAndClose() {
					case ConnLoopCodeBreak:
						break sessionLoop

					case ConnLoopCodeContinue:
					}

				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodePassThrough:
				}

				otherPlayerSessionId := game.HostPlayer.SessionID
				if s.Player.IsHost {
					otherPlayerSessionId = game.JoinPlayer.SessionID
				}
				GlobalSessionManager.otherSessionMsg <- NewOtherSessionMsg(otherPlayerSessionId, s.Game.Uuid, respStartGame)
			}

		case md.CodeJoinGame:
			req := NewRequest(conn, s, payload)
			resp, game := req.HandleJoinPlayer()

			switch WriteJsonWithRetry(conn, resp) {
			case ConnLoopCodeBreak:
				game.JoinPlayer = nil
				break sessionLoop
			case ConnLoopCodeContinue:
				game.JoinPlayer = nil
				continue sessionLoop
			case ConnLoopCodePassThrough:
			}

			// If the second playerd joined successfully, then `CodeSelectGrid`
			// is sent to both players as an indication of grid selection
			if resp.Error.ErrorDetails == "" {
				readyResp := md.NewMessage[md.NoPayload](md.CodeSelectGrid)

				switch WriteJsonWithRetry(conn, readyResp) {
				case ConnLoopAbnormalClosureRetry:
					switch s.waitAndClose() {
					case ConnLoopCodeBreak:
						break sessionLoop

					case ConnLoopCodeContinue:
					}

				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodePassThrough:
				}

				GlobalSessionManager.otherSessionMsg <- NewOtherSessionMsg(game.HostPlayer.SessionID, s.Game.Uuid, readyResp)
			}

		default:
			respInvalidSignal := md.NewMessage[md.NoPayload](md.CodeInvalidSignal)
			respInvalidSignal.AddError("", "invalid code in the incoming payload")
			switch WriteJsonWithRetry(conn, respInvalidSignal) {
			case ConnLoopAbnormalClosureRetry:
				switch s.waitAndClose() {
				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodeContinue:
					continue sessionLoop
				}
			case ConnLoopCodeBreak:
				break sessionLoop
			default:
				continue sessionLoop
			}
		}
	}
}

func (s *Session) waitAndClose() int {
	log.Printf("starting grace period for %s\n", s.ID)

	s.mu.Lock()
	s.GraceTimer = time.AfterFunc(GracePeriod, func() {
		GlobalSessionManager.mu.Lock()
		delete(GlobalSessionManager.Sessions, s.ID)
		s.Conn.Close()
		GlobalSessionManager.mu.Unlock()
	})
	s.mu.Unlock()

	// This means there is no game and abnormal closure is happening
	if s.Game == nil {
		return ConnLoopCodeBreak
	}

	otherPlayer := s.Game.HostPlayer
	if s.Player.IsHost {
		otherPlayer = s.Game.JoinPlayer
	}
	if err := otherPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerGracePeriod)); err != nil {
		// If other player connection is disrupted as well, then end the session
		return ConnLoopCodeBreak
	}

	select {
	case <-s.GraceTimer.C:
		if otherPlayer != nil {
			_ = otherPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerDisconnected))
		}
		return ConnLoopCodeBreak

		// If reconnection happens, loop stops
	case <-s.StopRetry:
		if otherPlayer != nil {
			_ = otherPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerReconnected))
		}
		return ConnLoopCodeContinue
	}
}

func (s *Session) terminateSession() {
	GlobalGameManager.mu.Lock()
	delete(GlobalGameManager.Games, s.Game.Uuid)
	GlobalGameManager.mu.Unlock()

	GlobalSessionManager.mu.Lock()
	delete(GlobalSessionManager.Sessions, s.ID)
	GlobalSessionManager.mu.Unlock()

	log.Println("session closed:", s.ID)
}
