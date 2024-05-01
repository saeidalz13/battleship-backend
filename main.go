package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"

	// "github.com/saeidalz13/battleship-backend/db"
	"github.com/saeidalz13/battleship-backend/models"
	"github.com/saeidalz13/battleship-backend/utils"
)

var DB *sql.DB
var defaultPort int16 = 8000

type Server struct {
	Port    *int16
	Db      *sql.DB
	Games   map[string]*models.Game
	Players map[string]*models.Player
}

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

type Option func(*Server) error

var upgrader = websocket.Upgrader{
	HandshakeTimeout: time.Second * 4, // arbitrary duration
	// arbitrary buffer size
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
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

		log.Println("message type:", messageType)
		log.Print("payload:", string(payload))

		var signal models.SignalStruct
		if err := json.Unmarshal(payload, &signal); err != nil {
			log.Println(err)
			continue
		}

		switch {
		case signal.Code == models.CodeStartGame:
			newId, newGame, newPlayer := utils.StartGame(ws.RemoteAddr().String())
			s.Games[newId] = newGame
			s.Players[ws.RemoteAddr().String()] = newPlayer

			newResp := models.StartGameResp{
				GameUuid: newGame.Uuid,
				HostUuid: newPlayer.Uuid,
				Grid:     newPlayer.Grid,
			}

			jsonResp, err := json.Marshal(newResp)
			if err != nil {
				log.Println(err)
			}
			if err := ws.WriteJSON(jsonResp); err != nil {
				log.Println(err)
				continue
			}

		case signal.Code == models.CodeEndGame:
			log.Println("end game")

		case signal.Code == models.CodeAttack:
			log.Println("attack!")

		default:
			continue
		}

		if err := ws.WriteMessage(messageType, []byte("server response: message received client!")); err != nil {
			log.Println(err)
			continue
		}
	}
}

func (s *Server) HandleWs(w http.ResponseWriter, r *http.Request) {
	/*
		! TODO: this accept connection from any origin
		! TODO: Must change for production
	*/
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// use Upgrade method to make a websocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		return
	}

	// TODO: Put this connection in a database (psql)
	log.Println("a new connection established!", ws.RemoteAddr().String())

	// managing connection on another goroutine
	go s.manageWsConn(ws)
}

func main() {
	if os.Getenv("STAGE") != "prod" {
		if err := godotenv.Load(".env"); err != nil {
			panic(err)
		}
	}
	// psqlUrl := os.Getenv("PSQL_URL")
	// DB = db.MustConnectToDb(psqlUrl)

	server := NewServer(WithPort(9191))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /battleship", server.HandleWs)

	log.Println("listening to port 9191...")
	log.Fatalln(http.ListenAndServe(":9191", mux))
}
