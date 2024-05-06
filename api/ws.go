package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	md "github.com/saeidalz13/battleship-backend/models"
)

const (
	StageProd = "prod"
	StageDev  = "dev"
)

var defaultPort int = 8000

var allowedOrigins = map[string]bool{
	"https://www.allowed_url.com": true,
}

type Server struct {
	Port     *int
	Upgrader websocket.Upgrader
	Db       *sql.DB
	Games    map[string]*md.Game
	Players  map[string]*md.Player
}

func (s *Server) FindGame(gameUuid string) *md.Game {
	game, prs := s.Games[gameUuid]
	if !prs {
		return nil 
	}
	log.Printf("game found: %s", gameUuid)
	return game 
}

func (s *Server) FindPlayer(playerUuid string) *md.Player {
	player, prs := s.Players[playerUuid]
	if !prs {
		return nil 
	}
	log.Printf("player found: %s", playerUuid)
	return player
}

type Option func(*Server) error


func NewServer(optFuncs ...Option) *Server {
	var server Server
	for _, opt := range optFuncs {
		if err := opt(&server); err != nil {
			panic(err)
		}
	}
	if server.Port == nil {
		server.Port = &defaultPort
	}
	if server.Games == nil {
		server.Games = make(map[string]*md.Game)
	}
	if server.Players == nil {
		server.Players = make(map[string]*md.Player)
	}

	upgrader := websocket.Upgrader{
		// good average time since this is not a high-latency operation such as video streaming
		HandshakeTimeout: time.Second * 5,

		// probably more that enough but this is a good average size
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
	}

	server.Upgrader = upgrader
	if server.Upgrader.CheckOrigin == nil {
		server.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	}
	return &server
}

func WithPort(port int) Option {
	return func(s *Server) error {
		if port > 10000 {
			panic("choose a port less than 10000")
		}
		s.Port = &port
		return nil
	}
}

func WithDb(db *sql.DB) Option {
	return func(s *Server) error {
		s.Db = db
		return nil
	}
}

func WithStage(stage string) Option {
	return func(s *Server) error {
		if stage == StageProd {
			s.Upgrader.CheckOrigin = func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return allowedOrigins[origin]
			}
			return nil
		}

		if stage == StageDev {
			s.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
			return nil
		}

		return fmt.Errorf("stage must be prod or dev")
	}
}

func (s *Server) manageWsConn(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		log.Println("connection closed:", ws.RemoteAddr().String())
	}()

	for {
		// A WebSocket frame can be one of 6 types: text=1, binary=2, ping=9, pong=10, close=8 and continuation=0
		// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
		_, payload, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
			}
			// whatever else is not really an error. would be normal closure
			break
		}

		// log the incoming messages
		// log.Println("message type:", messageType)
		// log.Printf("payload: %s, len: %d", string(payload), len(payload))

		// the incoming message must be of type json containing the field "code"
		// which would allow us to determine what action is required
		// any other format of incoming message is invalid and will be ignored
		var signal md.Signal
		if err := json.Unmarshal(payload, &signal); err != nil {
			log.Println(err)
			if err := ws.WriteMessage(websocket.TextMessage, []byte("incoming message is invalid; must be of type json")); err != nil {
				log.Println(err)
				break
			}
			continue
		}

		// This is where we choose the action based on the code in incoming json
		switch signal.Code {
		case md.CodeReqCreateGame:
			// Finalized
			req := NewWsRequest(s, ws)
			resp, _ := req.CreateGame()
			if err := ws.WriteJSON(resp); err != nil {
				log.Printf("failed to create new game: %v\n", err)
				continue
			}

		case md.CodeRespEndGame:
			if err := EndGame(s, ws, payload); err != nil {
				log.Printf("failed to end game: %v\n", err)
			}
			log.Println("end game")

		case md.CodeReqAttack:
			if err := Attack(s, ws, payload); err != nil {
				log.Printf("failed to attack: %v\n", err)
				respFail := md.NewMessage(md.CodeRespFailAttack, md.WithPayload(md.NewRespFail(err.Error(), "failed to handle attack request")))
				if err := ws.WriteJSON(respFail); err != nil {
					log.Println(err)
					continue
				}
			} else {
				// TODO: should you give me x and y? or the entire grid? seems redundant...
				if err := ws.WriteJSON(md.NewMessage(0)); err != nil {
					log.Println(err)
					continue
				}
				// TODO: Notify the other player about this event and tell them it's their turn
			}

		case md.CodeReqReady:
			req := NewWsRequest(s, ws, payload)
			resp, game, err := req.ManageReadyPlayer()
			if err != nil {
				log.Printf("failed to make the player ready: %v\n", err)
				respFail := md.NewMessage(md.CodeRespFailReady, md.WithPayload(md.NewRespFail(err.Error(), "failed to make the player ready")))
				if err := ws.WriteJSON(respFail); err != nil {
					log.Println(err)
				}
				continue
			} else {
				if err := ws.WriteJSON(resp); err != nil {
					log.Println(err)
					continue
				}

				if game.HostPlayer.IsReady && game.JoinPlayer.IsReady {
					if err := SendJSONBothPlayers(game, md.NewSignal(md.CodeRespStartGame)); err != nil {
						log.Println(err)
					}
					continue
				}
			}

		case md.CodeReqJoinGame:
			// Finalized
			req := NewWsRequest(s, ws, payload)
			resp, game, err := req.JoinPlayerToGame()
			if err != nil {
				log.Printf("failed to join player: %v\n", err)
				respFail := md.NewMessage(md.CodeRespFailJoinGame, md.WithPayload(md.NewRespFail(err.Error(), "failed to join the player")))
				if err := ws.WriteJSON(respFail); err != nil {
					continue
				}
			} else {
				if err := ws.WriteJSON(resp); err != nil {
					log.Println(err)
					continue
				}

				if err := SendJSONBothPlayers(game, md.Signal{Code: md.CodeRespSuccessJoinGame}); err != nil {
					log.Println(err)
					continue
				}
			}

		default:
			respInvalidSignal := md.NewMessage(md.CodeRespInvalidSignal)
			if err := ws.WriteJSON(respInvalidSignal); err != nil {
				log.Println(err)
				continue
			}
		}
	}
}

func (s *Server) HandleWs(w http.ResponseWriter, r *http.Request) {
	// use Upgrade method to make a websocket connection
	ws, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		return
	}

	log.Println("a new connection established!", ws.RemoteAddr().String())

	// managing connection on another goroutine
	go s.manageWsConn(ws)
}
