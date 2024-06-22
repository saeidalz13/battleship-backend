package api

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	md "github.com/saeidalz13/battleship-backend/models"
)

const (
	gracePeriod time.Duration = time.Minute * 3
)

type SessionMessage struct {
	SenderSession *Session
	ReceiverID    string
	GameUuid      string
	Payload       interface{}
}

func NewSessionMessage(senderSession *Session, receiverId string, gameUuid string, p interface{}) SessionMessage {
	return SessionMessage{
		ReceiverID: receiverId,
		GameUuid:   gameUuid,
		Payload:    p,
	}
}

type Session struct {
	ID             string
	Conn           *websocket.Conn
	GameUuid       string
	Player         *md.Player
	GraceTimer     *time.Timer
	StopRetry      chan struct{}
	mu             sync.Mutex
	GameManager    *GameManager
	SessionManager *SessionManager
	CreatedAt      time.Time
}

func NewSession(conn *websocket.Conn, sessionID string, gameManager *GameManager, sessionManager *SessionManager) *Session {
	return &Session{
		ID:             sessionID,
		Conn:           conn,
		StopRetry:      make(chan struct{}),
		GameManager:    gameManager,
		SessionManager: sessionManager,
		CreatedAt:      time.Now(),
	}
}

func (s *Session) run() {
	defer s.terminate()

sessionLoop:
	for {
		// conn := s.Conn
		// A WebSocket frame can be one of 6 types: text=1, binary=2, ping=9, pong=10, close=8 and continuation=0
		// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
		retries := 0
		_, payload, err := s.Conn.ReadMessage()
		if err != nil {
			switch IdentifyWsConnErrAction(err) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
				case ConnLoopCodeBreak:
					break sessionLoop
				case ConnLoopCodeContinue:
					continue sessionLoop
				}

			case ConnLoopCodeRetry:
				if retries < maxWriteWsRetries {
					retries++
					log.Printf("failed to read from ws conn [%s]; retrying... (retry no. %d)\n", s.Conn.RemoteAddr().String(), retries)
					time.Sleep(time.Duration(retries*backOffFactor) * time.Second)
					continue sessionLoop

				} else {
					break sessionLoop
				}

			case ConnLoopCodeBreak:
				log.Printf("break ws conn loop [%s] due to: %s\n", s.Conn.RemoteAddr().String(), err)
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

			switch WriteJSONWithRetry(s.Conn, resp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
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
			req := NewRequest(s)
			resp := req.HandleCreateGame()

			switch WriteJSONWithRetry(s.Conn, resp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
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
			req := NewRequest(s, payload)
			// response will have the IsTurn field of attacker
			resp, defender := req.HandleAttack()

			if resp.Error.ErrorDetails != "" {
				switch WriteJSONWithRetry(s.Conn, resp) {
				case ConnLoopCodeBreak:
					break sessionLoop
				default:
					continue sessionLoop
				}
			}

			// attacker turn is set to false
			resp.Payload.IsTurn = false
			switch WriteJSONWithRetry(s.Conn, resp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
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
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, defender.SessionID, s.GameUuid, resp)

			// If this attack caused the game to end.
			// Both attacker and defender will get a end game
			// message indicating if they lost or won
			if defender.MatchStatus == md.PlayerMatchStatusLost {
				// Sending victory code to the attacker
				respAttacker := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respAttacker.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusWon})
				switch WriteJSONWithRetry(s.Conn, respAttacker) {
				case ConnLoopAbnormalClosureRetry:
					switch s.handleAbnormalClosure() {
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
				s.SessionManager.CommunicationChan <- NewSessionMessage(s, defender.SessionID, s.GameUuid, respDefender)
			}

		case md.CodeReady:
			req := NewRequest(s, payload)
			resp, game := req.HandleReadyPlayer()

			if resp.Error.ErrorDetails != "" {
				switch WriteJSONWithRetry(s.Conn, resp) {
				case ConnLoopCodeBreak:
					break sessionLoop
				default:
					continue sessionLoop
				}
			}

			switch WriteJSONWithRetry(s.Conn, resp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
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
				switch WriteJSONWithRetry(s.Conn, respStartGame) {
				case ConnLoopAbnormalClosureRetry:
					switch s.handleAbnormalClosure() {
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
				s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayerSessionId, s.GameUuid, respStartGame)
			}

		case md.CodeJoinGame:
			req := NewRequest(s, payload)
			resp, game := req.HandleJoinPlayer()

			switch WriteJSONWithRetry(s.Conn, resp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodeContinue:
				}

			case ConnLoopCodeBreak:
				break sessionLoop

			case ConnLoopCodePassThrough:
			}

			// If the second playerd joined successfully, then `CodeSelectGrid`
			// is sent to both players as an indication of grid selection
			if resp.Error.ErrorDetails == "" {
				readyResp := md.NewMessage[md.NoPayload](md.CodeSelectGrid)

				switch WriteJSONWithRetry(s.Conn, readyResp) {
				case ConnLoopAbnormalClosureRetry:
					switch s.handleAbnormalClosure() {
					case ConnLoopCodeBreak:
						break sessionLoop

					case ConnLoopCodeContinue:
					}

				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodePassThrough:
				}

				s.SessionManager.CommunicationChan <- NewSessionMessage(s, game.HostPlayer.SessionID, s.GameUuid, readyResp)
			}

		case md.CodeRequestRematchFromServer:
			// 1. See if the game still exists
			game, err := s.GameManager.FindGame(s.GameUuid)
			if err != nil {
				break sessionLoop
			}

			// 2. Check if the other player still exists
			var otherPlayer *md.Player
			for _, player := range game.Players {
				if player.Uuid != s.Player.Uuid {
					otherPlayer = player
				}
			}
			// The other player had already quit and didn't
			// want a rematch
			if otherPlayer == nil {
				break sessionLoop
			}

			msg := md.NewMessage[md.NoPayload](md.CodeRequestRematchFromOtherPlayer)
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, msg)

		case md.CodeRematch:
			s.restartGame()
			readyResp := md.NewMessage[md.NoPayload](md.CodeSelectGrid)

			switch WriteJSONWithRetry(s.Conn, readyResp) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
				case ConnLoopCodeBreak:
					break sessionLoop

				case ConnLoopCodeContinue:
				}

			case ConnLoopCodeBreak:
				break sessionLoop

			case ConnLoopCodePassThrough:
			}

		default:
			respInvalidSignal := md.NewMessage[md.NoPayload](md.CodeInvalidSignal)
			respInvalidSignal.AddError("", "invalid code in the incoming payload")
			switch WriteJSONWithRetry(s.Conn, respInvalidSignal) {
			case ConnLoopAbnormalClosureRetry:
				switch s.handleAbnormalClosure() {
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

func (s *Session) handleAbnormalClosure() int {
	log.Printf("starting grace period for %s\n", s.ID)

	s.mu.Lock()
	s.GraceTimer = time.AfterFunc(gracePeriod, func() {
		s.SessionManager.mu.Lock()
		s.Conn.Close()
		delete(s.SessionManager.Sessions, s.ID)
		s.SessionManager.mu.Unlock()
	})
	s.mu.Unlock()

	// This means there is no game and abnormal closure is happening
	game, err := s.GameManager.FindGame(s.GameUuid)
	if err != nil {
		return ConnLoopCodeBreak
	}

	otherPlayer := game.HostPlayer
	if s.Player.IsHost {
		otherPlayer = game.JoinPlayer
	}

	if otherPlayer != nil {
		if err := otherPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerGracePeriod)); err != nil {
			// If other player connection is disrupted as well, then end the session
			return ConnLoopCodeBreak
		}
	}

	select {
	case <-s.GraceTimer.C:
		if otherPlayer != nil {
			_ = otherPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerDisconnected))
		}
		log.Printf("session terminated: %s\n", s.ID)
		return ConnLoopCodeBreak

		// If reconnection happens, loop stops
	case <-s.StopRetry:
		if otherPlayer != nil {
			_ = otherPlayer.WsConn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeOtherPlayerReconnected))
		}
		log.Printf("player reconnected, session: %s\n", s.ID)
		return ConnLoopCodeContinue
	}
}

// This will delete player from the game players map
// and delete the player session
func (s *Session) terminate() {
	s.GameManager.DeletePlayerChan <- NewDeletePlayerSignal(s.GameUuid, s.Player.Uuid)
	s.SessionManager.DeleteSessionChan <- s.ID
}

// Restarting the game means we play the game with
// the same gameUuid but the player info gets reset
func (s *Session) restartGame() {
	s.Player.AttackGrid = md.NewGrid()
	s.Player.DefenceGrid = md.NewGrid()
	if s.Player.IsHost {
		s.Player.IsTurn = true
	} else {
		s.Player.IsTurn = false
	}
	s.Player.Ships = md.NewShipsMap()
	s.Player.SunkenShips = 0
	s.Player.MatchStatus = md.PlayerMatchStatusUndefined
}
