package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/models"
)

const (
	StageProd = "prod"
	StageDev  = "dev"
)

var defaultPort int16 = 8000

var allowedOrigins = map[string]bool{
	"https://www.allowed_url.com": true,
}

type Server struct {
	Port     *int16
	Upgrader websocket.Upgrader
	Db       *sql.DB
	Games    map[string]*models.Game
	Players  map[string]*models.Player
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
		server.Games = make(map[string]*models.Game)
	}
	if server.Players == nil {
		server.Players = make(map[string]*models.Player)
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

func WithPort(port int16) Option {
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
		messageType, payload, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
			}
			// whatever else is not really an error. would be normal closure
			break
		}

		// log the incoming messages
		log.Println("message type:", messageType)
		log.Printf("payload: %s, len: %d", string(payload), len(payload))

		// the incoming message must be of type json containing the field "code"
		// which would allow us to determine what action is required
		// any other format of incoming message is invalid and will be ignored
		var signal models.Signal
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
		case models.CodeReqCreateGame:
			if err := CreateGame(s, ws); err != nil {
				log.Println(err)
				continue
			}

		case models.CodeEndGame:
			log.Println("end game")

		case models.CodeReqAttack:
			log.Println("attack!")

		case models.CodeReqReady:
			if err := ManageReadyPlayer(s, ws, payload); err != nil {
				log.Println(err)
				if err := ws.WriteJSON(models.NewRespFail(models.CodeRespFailReady, err.Error(), "failed to make the player ready")); err != nil {
					// TODO: to be decided what to do if writing to connection failed
					continue
				}
				continue
			}

		case models.CodeReqJoinGame:
			if err := JoinPlayerToGame(s, ws, payload); err != nil {
				log.Println(err)
				if err := ws.WriteJSON(models.NewRespFail(models.CodeRespFailJoinGame, err.Error(), "failed to join the player")); err != nil {
					// TODO: to be decided what to do if writing to connection failed
					continue
				}
				continue
			}
		default:
			continue
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
