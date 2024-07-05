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

			s.writeToConn(resp)
		}

		// This is where we choose the action based on the code in incoming json
		switch signal.Code {

		case mc.CodeCreateGame:
			req := NewRequest(s, payload)
			resp := req.HandleCreateGame()

			s.writeToConn(resp)

		case mc.CodeAttack:
			req := NewRequest(s, payload)
			// response will have the IsTurn as false field of attacker
			resp, defender := req.HandleAttack()

			s.writeToConn(resp)
			if resp.Error != nil {
				continue sessionLoop
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
				s.writeToConn(respAttacker)

				// Sending failure code to the defender
				respDefender := mc.NewMessage[mc.RespEndGame](mc.CodeEndGame)
				respDefender.AddPayload(mc.RespEndGame{PlayerMatchStatus: mb.PlayerMatchStatusLost})
				s.SessionManager.CommunicationChan <- NewSessionMessage(s, defender.SessionID, s.GameUuid, respDefender)
			}

		case mc.CodeReady:
			req := NewRequest(s, payload)
			resp, game := req.HandleReadyPlayer()

			s.writeToConn(resp)
			if resp.Error != nil {
				continue sessionLoop
			}

			if game.HostPlayer.IsReady && game.JoinPlayer.IsReady {
				respStartGame := mc.NewMessage[mc.NoPayload](mc.CodeStartGame)
				s.writeToConn(respStartGame)

				otherPlayer := game.GetOtherPlayer(s.Player)
				s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, respStartGame)
			}

		case mc.CodeJoinGame:
			req := NewRequest(s, payload)
			resp, game := req.HandleJoinPlayer()

			s.writeToConn(resp)
			if resp.Error != nil {
				break sessionLoop
			}

			// If the second playerd joined successfully, then `CodeSelectGrid`
			// is sent to both players as an indication of grid selection
			readyResp := mc.NewMessage[mc.NoPayload](mc.CodeSelectGrid)
			s.writeToConn(readyResp)
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, game.HostPlayer.SessionID, s.GameUuid, readyResp)

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

			otherPlayer := game.GetOtherPlayer(s.Player)
			if otherPlayer == nil {
				break sessionLoop
			}

			s.Player.IsTurn = true
			// Notify the other player if they want a rematch
			msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCall)
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, msg)

		case mc.CodeRematchCallAccepted:
			// Send the rematch call acceptance to other player
			game, err := s.GameManager.FindGame(s.GameUuid)
			if err != nil {
				break sessionLoop
			}

			if err := game.Reset(); err != nil {
				break sessionLoop
			}

			// Notify the other player with their turn
			msgOtherPlayer := mc.NewMessage[mc.RespRematch](mc.CodeRematch)
			otherPlayer := game.GetOtherPlayer(s.Player)
			if otherPlayer == nil {
				break sessionLoop
			}
			msgOtherPlayer.AddPayload(mc.RespRematch{IsTurn: otherPlayer.IsTurn})
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, msgOtherPlayer)

			s.Player.IsTurn = false
			msgPlayer := mc.NewMessage[mc.RespRematch](mc.CodeRematch)
			msgPlayer.AddPayload(mc.RespRematch{IsTurn: s.Player.IsTurn})

			// Notify the acceptor with their turn
			s.writeToConn(msgPlayer)

		case mc.CodeRematchCallRejected:
			game, err := s.GameManager.FindGame(s.GameUuid)
			if err != nil {
				break sessionLoop
			}

			// Notify the other player that no rematch is wanted now
			msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCallRejected)
			otherPlayer := game.GetOtherPlayer(s.Player)
			if otherPlayer == nil {
				break sessionLoop
			}
			s.SessionManager.CommunicationChan <- NewSessionMessage(s, otherPlayer.SessionID, s.GameUuid, msg)

			break sessionLoop

		default:
			respInvalidSignal := mc.NewMessage[mc.NoPayload](mc.CodeInvalidSignal)
			respInvalidSignal.AddError("", "invalid code in the incoming payload")
			s.writeToConn(respInvalidSignal)
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

// Writes to the connection of that session. It also
// handles the abnormal or other types of errors of
// writing to a websocket connection.
func (s *Session) writeToConn(p interface{}) {
	switch WriteJSONWithRetry(s.Conn, p) {
	case ConnLoopAbnormalClosureRetry:
		switch s.handleAbnormalClosure() {
		case ConnLoopCodeBreak:
			s.terminate()

		case ConnLoopCodeContinue:
		}
	case ConnLoopCodeBreak:
		s.terminate()
	default:
	}
}

func (s *Session) handleAbnormalClosure() int {
	// This means there is no game and abnormal closure is happening
	// which means this session is invalid and should end
	game, err := s.GameManager.FindGame(s.GameUuid)
	if err != nil {
		return ConnLoopCodeBreak
	}

	otherPlayer := game.GetOtherPlayer(s.Player)
	if otherPlayer == nil {
		return ConnLoopCodeBreak
	}

	// Absence of otherPlayer session means this game is invalid
	otherSession, err := s.SessionManager.FindSession(otherPlayer.SessionID)
	if err != nil {
		return ConnLoopCodeBreak
	}

	if err := otherSession.Conn.WriteJSON(mc.NewMessage[mc.NoPayload](mc.CodeOtherPlayerGracePeriod)); err != nil {
		// If other player connection is disrupted as well, then end the session
		return ConnLoopCodeBreak
	}

	log.Printf("starting grace period for %s\n", s.ID)
	timer := time.NewTimer(gracePeriod)

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
