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

type SessionManager struct {
	Sessions          map[string]*Session
	CommunicationChan chan SessionMessage
	mu                sync.Mutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		Sessions:          make(map[string]*Session),
		CommunicationChan: make(chan SessionMessage),
	}
}

func (sm *SessionManager) ManageCommunication() {
	for {
		msg := <-sm.CommunicationChan

		sm.mu.Lock()
		receiverSession, prs := sm.Sessions[msg.ReceiverID]
		if !prs {
			// It should never be the case that the other session
			// is not found. The sender session should terminate
			msg.SenderSession.terminate()
			continue
		}

		if receiverSession.Game.Uuid != msg.GameUuid {
			panic("receiver session msg game is not the same as game uuid; this error should never happen")
		}

		switch WriteJSONWithRetry(receiverSession.Conn, msg.Payload) {
		case ConnLoopAbnormalClosureRetry:
			switch receiverSession.handleAbnormalClosure() {
			case ConnLoopCodeBreak:
				receiverSession.terminate()

			case ConnLoopCodeContinue:
			}

		case ConnLoopCodeBreak:
			receiverSession.terminate()

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
	ID             string
	Conn           *websocket.Conn
	Game           *md.Game
	Player         *md.Player
	GraceTimer     *time.Timer
	StopRetry      chan struct{}
	mu             sync.Mutex
	GameManager    *GameManager
	SessionManager *SessionManager
}

func NewSession(conn *websocket.Conn, sessionID string, gameManager *GameManager, sessionManager *SessionManager) *Session {
	return &Session{
		ID:             sessionID,
		Conn:           conn,
		StopRetry:      make(chan struct{}),
		GameManager:    gameManager,
		SessionManager: sessionManager,
	}
}

func (s *Session) run() {
	defer s.terminate()

sessionLoop:
	for {
		conn := s.Conn
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

			switch WriteJSONWithRetry(conn, resp) {
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
			req := NewRequest(conn, s)
			resp := req.HandleCreateGame()

			switch WriteJSONWithRetry(conn, resp) {
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
			req := NewRequest(nil, s, payload)
			// response will have the IsTurn field of attacker
			resp, defender := req.HandleAttack()

			if resp.Error.ErrorDetails != "" {
				switch WriteJSONWithRetry(conn, resp) {
				case ConnLoopCodeBreak:
					break sessionLoop
				default:
					continue sessionLoop
				}
			}

			// attacker turn is set to false
			resp.Payload.IsTurn = false
			switch WriteJSONWithRetry(conn, resp) {
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
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, defender.SessionID, s.Game.Uuid, resp)

			// If this attack caused the game to end.
			// Both attacker and defender will get a end game
			// message indicating if they lost or won
			if defender.MatchStatus == md.PlayerMatchStatusLost {
				// Sending victory code to the attacker
				respAttacker := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respAttacker.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusWon})
				switch WriteJSONWithRetry(conn, respAttacker) {
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
				s.SessionManager.CommunicationChan <- NewSessionMessage(s, defender.SessionID, s.Game.Uuid, respDefender)

				// Wait for 5 seconds to make sure all the messages have been
				// sent and nothing has a nil pointer after session termination
				time.Sleep(time.Second * 5)
				return
			}

		case md.CodeReady:
			req := NewRequest(nil, s, payload)
			resp, game := req.HandleReadyPlayer()

			if resp.Error.ErrorDetails != "" {
				switch WriteJSONWithRetry(conn, resp) {
				case ConnLoopCodeBreak:
					break sessionLoop
				default:
					continue sessionLoop
				}
			}

			switch WriteJSONWithRetry(conn, resp) {
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
				switch WriteJSONWithRetry(conn, respStartGame) {
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
				s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayerSessionId, s.Game.Uuid, respStartGame)
			}

		case md.CodeJoinGame:
			req := NewRequest(conn, s, payload)
			resp, game := req.HandleJoinPlayer()

			switch WriteJSONWithRetry(conn, resp) {
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

				switch WriteJSONWithRetry(conn, readyResp) {
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

				s.SessionManager.CommunicationChan <- NewSessionMessage(s, game.HostPlayer.SessionID, s.Game.Uuid, readyResp)
			}

		default:
			respInvalidSignal := md.NewMessage[md.NoPayload](md.CodeInvalidSignal)
			respInvalidSignal.AddError("", "invalid code in the incoming payload")
			switch WriteJSONWithRetry(conn, respInvalidSignal) {
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
	s.GraceTimer = time.AfterFunc(GracePeriod, func() {
		s.SessionManager.mu.Lock()
		s.Conn.Close()
		delete(s.SessionManager.Sessions, s.ID)
		s.SessionManager.mu.Unlock()
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

func (s *Session) terminate() {
	if s.Game != nil {
		s.GameManager.EndGameSignal <- s.Game.Uuid
	}

	s.SessionManager.mu.Lock()
	delete(s.SessionManager.Sessions, s.ID)
	s.SessionManager.mu.Unlock()

	log.Println("session closed:", s.ID)
}
