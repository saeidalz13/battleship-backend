package api

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/db/sqlc"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
	"github.com/sqlc-dev/pqtype"
)

const (
	URLQuerySessionIDKeyword string = "sessionID"
)

var (
	// allowedOrigins     = map[string]bool{
	// 	"https://www.allowed_url.com": true,
	// }
	upgrader = websocket.Upgrader{

		// good average time since this is not a high-latency operation such as video streaming
		HandshakeTimeout: time.Second * 5,

		// probably more that enough but this is a good average size
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

type RequestProcessor struct {
	sessionManager mc.SessionManager
	gameManager    mb.GameManager
	dbManager      sqlc.DbManager
	ipnet          net.IPNet
}

func NewRequestProcessor(
	sessionManager mc.SessionManager,
	gameManager mb.GameManager,
	dbManager sqlc.DbManager,
) RequestProcessor {
	rp := RequestProcessor{
		sessionManager: sessionManager,
		gameManager:    gameManager,
		dbManager:      dbManager,
	}

	rp = rp.mustGetServerIpNet()
	return rp
}

func (rp RequestProcessor) mustGetServerIpNet() RequestProcessor {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, iface := range ifaces {
		// If the flag is down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			panic(err)
		}

		for _, addr := range addrs {
			var ipnet *net.IPNet
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ipnet = v
				ip = v.IP

			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				rp.ipnet = *ipnet
				return rp
			}
		}
	}

	panic("ipnet could not be found!")
}

// Expose this method to use it in testing
func (rp RequestProcessor) GetIpNet() net.IPNet {
	return rp.ipnet
}


func (rp RequestProcessor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// use Upgrade method to make a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		return
	}

	sessionIdQuery := r.URL.Query().Get(URLQuerySessionIDKeyword)
	switch sessionIdQuery {
	case "":
		log.Println("a new connection established\tRemote Addr: ", conn.RemoteAddr().String())
		rp.processSessionRequests(rp.sessionManager.GenerateNewSession(conn))

	default:
		session, err := rp.sessionManager.FindSession(sessionIdQuery)
		if err != nil {
			// This either means an expired session or invalid session ID
			conn.WriteJSON(mc.NewMessage[mc.NoPayload](mc.CodeReceivedInvalidSessionID))
			conn.Close()
			return
		}

		rp.sessionManager.ReconnectSession(session, conn)
		/*
			we discussed that if app total closure or crash happens
			it is not the server's fault. Hence, the server doesn not need
			to provide the session information upon reconnection
			Send the session data to update client information
		*/
	}
}

func (sp *RequestProcessor) processSessionRequests(session *mc.Session) {
	defer sp.sessionManager.TerminateSession(session)

	var (
		otherSessionPlayer mb.Player
		sessionPlayer      mb.Player
		sessionGame        *mb.Game

		receiverSessionId string
		sessionId         = sp.sessionManager.GetSessionId(session)
	)

	resp := mc.NewMessage[mc.RespSessionId](mc.CodeSessionID)
	resp.AddPayload(mc.RespSessionId{SessionID: sp.sessionManager.GetSessionId(session)})
	if err := sp.sessionManager.WriteToSessionConn(session, resp, mc.MessageTypeJSON, receiverSessionId); err != nil {
		return
	}

	serverPqtypeInet := pqtype.Inet{IPNet: sp.ipnet, Valid: true}

sessionLoop:
	for {
		// A WebSocket frame can be one of 6 types: text=1, binary=2, ping=9, pong=10, close=8 and continuation=0
		// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
		_, payload, err := sp.sessionManager.ReadFromSessionConn(session, receiverSessionId)
		if err != nil {
			// This error happens after retriesp. If it's not nil,
			// then something was wrong with the session connection
			// and couldn't be resolved
			break sessionLoop
		}

		code, err := sp.sessionManager.FetchCodeFromMsg(session, payload)
		if err != nil {
			msg := mc.NewMessage[mc.NoPayload](mc.CodeSignalAbsent)
			msg.AddError("incoming req payload must contain 'code' field", "")
			if err = sp.sessionManager.WriteToSessionConn(session, msg, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}
			continue sessionLoop
		}

		switch code {

		// In this branch we initialize the game and hence create a host player
		case mc.CodeCreateGame:
			ctx, cancel := context.WithTimeout(context.Background(), sqlc.QuerierCtxTimeout)
			defer cancel()
			if err := sp.dbManager.Analytics.IncrementGamesCreatedCount(ctx, serverPqtypeInet); err != nil {
				// for now not killing the game for it
				log.Println(err)
			}

			game, hostPlayer, respMsg := NewRequest(payload).HandleCreateGame(sp.gameManager, sp.sessionManager.GetSessionId(session))
			sessionPlayer = hostPlayer
			sessionGame = game

			if err := sp.sessionManager.WriteToSessionConn(session, respMsg, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}

		// This branch handles joining a new player to an existing
		// game.
		case mc.CodeJoinGame:
			req := NewRequest(payload)
			game, joinPlayer, respMsg := req.HandleJoinPlayer(sp.gameManager, sp.sessionManager.GetSessionId(session))

			if err := sp.sessionManager.WriteToSessionConn(session, respMsg, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}
			if respMsg.Error != nil {
				break sessionLoop
			}

			// Cache this information for later use in the logic
			sessionPlayer = joinPlayer
			sessionGame = game

			if otherSessionPlayer == nil {
				otherSessionPlayer = sp.gameManager.GetOtherPlayerForGame(sessionGame, sessionPlayer)
				receiverSessionId = otherSessionPlayer.GetSessionId()
			}

			readyRespMsg := mc.NewMessage[mc.NoPayload](mc.CodeSelectGrid)
			if err := sp.sessionManager.WriteToSessionConn(session, readyRespMsg, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}

			if err := sp.sessionManager.Communicate(sessionId, receiverSessionId, readyRespMsg, mc.MessageTypeJSON); err != nil {
				break sessionLoop
			}

		// This code means the player has selected their grid and
		// ready to start the game
		case mc.CodeReady:
			req := NewRequest(payload)
			respMsg := req.HandleReadyPlayer(sp.gameManager, sessionGame, sessionPlayer)

			if err := sp.sessionManager.WriteToSessionConn(session, respMsg, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}

			if respMsg.Error != nil {
				continue sessionLoop
			}

			if otherSessionPlayer == nil {
				otherSessionPlayer = sp.gameManager.GetOtherPlayerForGame(sessionGame, sessionPlayer)
				receiverSessionId = otherSessionPlayer.GetSessionId()
			}

			if sp.gameManager.IsGameReadyToStart(sessionGame) {
				respStartGame := mc.NewMessage[mc.NoPayload](mc.CodeStartGame)
				if err := sp.sessionManager.WriteToSessionConn(session, respStartGame, mc.MessageTypeJSON, receiverSessionId); err != nil {
					break sessionLoop
				}

				if err := sp.sessionManager.Communicate(sessionId, receiverSessionId, respStartGame, mc.MessageTypeJSON); err != nil {
					break sessionLoop
				}
			}

		// This branch takse care of the attack logic. After every attack
		// `SessionPlayer` checks if the attacker has won the game. if so,
		// the game ends and a signal is sent to both players
		case mc.CodeAttack:
			req := NewRequest(payload)
			respMsg := req.HandleAttack(sessionGame, sessionPlayer, otherSessionPlayer, sp.gameManager)

			if err := sp.sessionManager.WriteToSessionConn(session, respMsg, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}

			// This means attack operation did not complete
			if respMsg.Error != nil {
				continue sessionLoop
			}

			// defender turn is set to true
			respMsg.Payload.IsTurn = true
			if err := sp.sessionManager.Communicate(sessionId, receiverSessionId, respMsg, mc.MessageTypeJSON); err != nil {
				break sessionLoop
			}
			log.Println("attack resp sent to other")

			if sessionPlayer.IsWinner() {
				respAttacker := mc.NewMessage[mc.RespEndGame](mc.CodeEndGame)
				respAttacker.AddPayload(mc.RespEndGame{PlayerMatchStatus: mb.PlayerMatchStatusWon})
				if err := sp.sessionManager.WriteToSessionConn(session, respAttacker, mc.MessageTypeJSON, receiverSessionId); err != nil {
					break sessionLoop
				}

				respDefender := mc.NewMessage[mc.RespEndGame](mc.CodeEndGame)
				respDefender.AddPayload(mc.RespEndGame{PlayerMatchStatus: mb.PlayerMatchStatusLost})
				if err := sp.sessionManager.Communicate(sessionId, receiverSessionId, respDefender, mc.MessageTypeJSON); err != nil {
					break sessionLoop
				}
			}

		case mc.CodeRematchCall:
			ctx, cancel := context.WithTimeout(context.Background(), sqlc.QuerierCtxTimeout)
			defer cancel()
			if err := sp.dbManager.Analytics.IncrementRematchCalledCount(ctx, serverPqtypeInet); err != nil {
				// for now not killing the game for it
				log.Println(err)
			}

			respMsg, err := NewRequest().HandleCallRematch(sp.gameManager, sessionGame)
			if err != nil {
				continue sessionLoop
			}

			if err := sp.sessionManager.Communicate(sessionId, receiverSessionId, respMsg, mc.MessageTypeJSON); err != nil {
				break sessionLoop
			}

		case mc.CodeRematchCallAccepted:
			msgPlayer, msgOtherPlayer, err := NewRequest().HandleAcceptRematchCall(sp.gameManager, sessionGame, sessionPlayer, otherSessionPlayer)
			if err != nil {
				log.Println(err)
				break sessionLoop
			}

			if err := sp.sessionManager.Communicate(sessionId, receiverSessionId, msgOtherPlayer, mc.MessageTypeJSON); err != nil {
				break sessionLoop
			}
			if err := sp.sessionManager.WriteToSessionConn(session, msgPlayer, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}

		// Notify the other player that no rematch is wanted now
		case mc.CodeRematchCallRejected:
			msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCallRejected)
			sp.sessionManager.Communicate(sessionId, receiverSessionId, msg, mc.MessageTypeJSON)
			break sessionLoop

		default:
			respInvalidSignal := mc.NewMessage[mc.NoPayload](mc.CodeInvalidSignal)
			respInvalidSignal.AddError("", "invalid code in the incoming payload")
			if err := sp.sessionManager.WriteToSessionConn(session, respInvalidSignal, mc.MessageTypeJSON, receiverSessionId); err != nil {
				break sessionLoop
			}
		}
	}
}