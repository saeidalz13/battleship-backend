package api

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

const (
	gracePeriod time.Duration = time.Minute * 2
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
	Player         *mb.Player
	StopRetry      chan struct{}
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
		var signal mc.Signal
		if err := json.Unmarshal(payload, &signal); err != nil {
			log.Println("incoming msg does not contain 'code':", err)
			resp := mc.NewMessage[mc.NoPayload](mc.CodeSignalAbsent)
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

		case mc.CodeCreateGame:
			req := NewRequest(s, payload)
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

		case mc.CodeAttack:
			req := NewRequest(s, payload)
			// response will have the IsTurn as false field of attacker
			resp, defender := req.HandleAttack()

			if resp.Error != nil {
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
			if defender.MatchStatus == mb.PlayerMatchStatusLost {
				// Sending victory code to the attacker
				respAttacker := mc.NewMessage[mc.RespEndGame](mc.CodeEndGame)
				respAttacker.AddPayload(mc.RespEndGame{PlayerMatchStatus: mb.PlayerMatchStatusWon})
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
				respDefender := mc.NewMessage[mc.RespEndGame](mc.CodeEndGame)
				respDefender.AddPayload(mc.RespEndGame{PlayerMatchStatus: mb.PlayerMatchStatusLost})
				s.SessionManager.CommunicationChan <- NewSessionMessage(s, defender.SessionID, s.GameUuid, respDefender)
			}

		case mc.CodeReady:
			req := NewRequest(s, payload)
			resp, game := req.HandleReadyPlayer()

			if resp.Error != nil {
				switch WriteJSONWithRetry(s.Conn, resp) {
				case ConnLoopAbnormalClosureRetry:
					switch s.handleAbnormalClosure() {
					case ConnLoopCodeBreak:
						break sessionLoop

					case ConnLoopCodeContinue:
					}
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
				respStartGame := mc.NewMessage[mc.NoPayload](mc.CodeStartGame)
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

		case mc.CodeJoinGame:
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
			if resp.Error == nil {
				readyResp := mc.NewMessage[mc.NoPayload](mc.CodeSelectGrid)
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
			} else {
				break sessionLoop
			}

		case mc.CodeRematchCall:
			// 1. See if the game still exists
			game, err := s.GameManager.FindGame(s.GameUuid)
			if err != nil {
				break sessionLoop
			}

			if game.IsRematchAlreadyCalled() {
				continue sessionLoop
			}

			game.CallRematch()

			// 2. Find the other player
			otherPlayer := game.HostPlayer
			if s.Player.IsHost {
				otherPlayer = game.JoinPlayer
			}

			// If the other player had already left
			if otherPlayer == nil {
				break sessionLoop
			}

			// Notify the other player if they want a rematch
			msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCall)
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, msg)

		case mc.CodeRematchCallAccepted:
			// Send the rematch call acceptance to other player
			game, err := s.GameManager.FindGame(s.GameUuid)
			if err != nil {
				break sessionLoop
			}

			// Notify the other player that let's play again!
			msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCallAccepted)
			otherPlayer := game.HostPlayer
			if s.Player.IsHost {
				otherPlayer = game.JoinPlayer
			}
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, msg)

			go game.Reset()

		case mc.CodeRematchCallRejected:
			game, err := s.GameManager.FindGame(s.GameUuid)
			if err != nil {
				break sessionLoop
			}

			// Notify the other player that no rematch is wanted now
			msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCallRejected)
			otherPlayer := game.HostPlayer
			if s.Player.IsHost {
				otherPlayer = game.JoinPlayer
			}
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, msg)

			break sessionLoop

		default:
			respInvalidSignal := mc.NewMessage[mc.NoPayload](mc.CodeInvalidSignal)
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

// This will delete player from the game players map
// and delete the player session
func (s *Session) terminate() {
	if s.Player != nil {
		s.GameManager.DeletePlayerFromGame(s.GameUuid, s.Player.Uuid)
	}
	s.SessionManager.DeleteSession(s.ID)
}

func (s *Session) handleAbnormalClosure() int {
	log.Printf("starting grace period for %s\n", s.ID)

	timer := time.NewTimer(gracePeriod)

	// This means there is no game and abnormal closure is happening
	// which means this session is invalid and should end
	game, err := s.GameManager.FindGame(s.GameUuid)
	if err != nil {
		return ConnLoopCodeBreak
	}

	otherPlayer := game.HostPlayer
	if s.Player.IsHost {
		otherPlayer = game.JoinPlayer
	}

	var otherSession *Session
	if otherPlayer != nil {
		var err error
		// Absence of otherPlayer session means this game is invalid
		otherSession, err = s.SessionManager.FindSession(otherPlayer.SessionID)
		if err != nil {
			return ConnLoopCodeBreak
		}
	}

	if otherSession != nil {
		if err := otherSession.Conn.WriteJSON(mc.NewMessage[mc.NoPayload](mc.CodeOtherPlayerGracePeriod)); err != nil {
			// If other player connection is disrupted as well, then end the session
			return ConnLoopCodeBreak
		}
	}

	select {
	case <-timer.C:
		if otherSession != nil {
			_ = otherSession.Conn.WriteJSON(mc.NewMessage[mc.NoPayload](mc.CodeOtherPlayerDisconnected))
		}
		log.Printf("session terminated: %s\n", s.ID)
		return ConnLoopCodeBreak

	case <-s.StopRetry:
		if otherSession != nil {
			_ = otherSession.Conn.WriteJSON(mc.NewMessage[mc.NoPayload](mc.CodeOtherPlayerReconnected))
		}
		log.Printf("player reconnected, session: %s\n", s.ID)
		return ConnLoopCodeContinue
	}
}
