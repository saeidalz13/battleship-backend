package test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/api"
)

var ClientConn *websocket.Conn

func TestMain(m *testing.M) {
	go func() {
		stage := "dev"
		server := api.NewServer(api.WithPort(7171), api.WithStage(stage))

		mux := http.NewServeMux()
		mux.HandleFunc("GET /battleship", server.HandleWs)

		log.Println("Listening to port 7171...")
		if err := http.ListenAndServe(":7171", mux); err != nil {
			log.Println(err)
			os.Exit(0)
		}

	}()

	// Give the server time to start
	time.Sleep(time.Second*2)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	log.Println("dialing...")
	c, _, err := dialer.Dial("ws://localhost:7171/battleship", nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	ClientConn = c
	os.Exit(m.Run())
}
