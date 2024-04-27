package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/saeidalz13/battleship-backend/db"
)

var DB *sql.DB

var upgrader = websocket.Upgrader{
	HandshakeTimeout: time.Second * 4, // arbitrary duration
	// arbitrary buffer size
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
}

func manageWsConn(ws *websocket.Conn) {
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

		if err := ws.WriteMessage(messageType, []byte("server response: message received client!")); err != nil {
			log.Println(err)
			continue
		}
	}
}

func HandleWs(w http.ResponseWriter, r *http.Request) {
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
	go manageWsConn(ws)
}

func main() {
	if os.Getenv("STAGE") != "prod" {
		if err := godotenv.Load(".env"); err != nil {
			panic(err)
		}
	}
	psqlUrl := os.Getenv("PSQL_URL")
	DB = db.MustConnectToDb(psqlUrl)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /battleship", HandleWs)

	log.Println("listening to port 9191...")
	log.Fatalln(http.ListenAndServe(":9191", mux))
}
