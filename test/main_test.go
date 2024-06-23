package test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/api"

	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

var (
	HostConn           *websocket.Conn
	JoinConn           *websocket.Conn
	GameUuid           string
	HostPlayerId       string
	JoinPlayerId       string
	HostSessionID      string
	JoinSessionID      string
)

func TestMain(m *testing.M) {
	go func() {
		stage := "dev"
		server := api.NewServer(api.WithPort(7171), api.WithStage(stage))

		go server.SessionManager.ManageCommunication()
		go server.SessionManager.CleanUpPeriodically()

		mux := http.NewServeMux()
		mux.HandleFunc("GET /battleship", server.HandleWs)

		log.Println("Listening to port 7171...")
		if err := http.ListenAndServe(":7171", mux); err != nil {
			log.Println(err)
			os.Exit(0)
		}

	}()

	// Give the server time to start
	time.Sleep(time.Second * 2)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	log.Println("dialing...")
	c, _, err := dialer.Dial("ws://localhost:7171/battleship", nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	HostConn = c

	// Read host session ID
	var respSessionId mc.Message[mc.RespSessionId]
	_ = HostConn.ReadJSON(&respSessionId)
	HostSessionID = respSessionId.Payload.SessionID

	c2, _, err := dialer.Dial("ws://localhost:7171/battleship", nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	JoinConn = c2

	// Read Join sessoin ID
	_ = JoinConn.ReadJSON(&respSessionId)
	JoinSessionID = respSessionId.Payload.SessionID

	log.Println("Host session ID:", HostSessionID)
	log.Println("Join session ID:", JoinSessionID)

	log.Printf("host: %s\tjoin: %s", HostConn.LocalAddr().String(), JoinConn.LocalAddr().String())
	os.Exit(m.Run())
}
