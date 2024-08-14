package test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/api"

	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"

	"github.com/DATA-DOG/go-sqlmock"
)

const (
	testPort                = "127.0.0.1"
	testWsUrl               = "ws://127.0.0.1:7171/battleship"
	outOfGridBoundNum uint8 = 255
)

var (
	HostConn       *websocket.Conn
	JoinConn       *websocket.Conn
	testGame       *mb.Game
	testGameUuid   string
	testHostPlayer *mb.BattleshipPlayer
	testJoinPlayer *mb.BattleshipPlayer
	HostSessionID  string
	JoinSessionID  string
	testRp         api.RequestProcessor
	dialer         = websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	testMock           sqlmock.Sqlmock
	testGameManager    *mb.BattleshipGameManager
	testSessionManager *mc.BattleshipSessionManager
	// testQuerier        sqlc.Querier
)

func TestMain(m *testing.M) {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	testMock = mock

	go func() {
		// // test db manager
		// querier := sqlc.New(db)
		// testQuerier = querier

		// test session manager
		bsm := mc.NewBattleshipSessionManager()
		testSessionManager = bsm
		go bsm.CleanupPeriodically()

		// test game manager
		bgm := mb.NewBattleshipGameManager()
		testGameManager = bgm

		rp := api.NewRequestProcessor(bsm, bgm, nil)
		testRp = rp

		mux := http.NewServeMux()
		mux.Handle("GET /battleship", rp)

		log.Println("Listening to port 7171...")
		if err := http.ListenAndServe(":7171", mux); err != nil {
			log.Println(err)
			os.Exit(0)
		}

	}()

	// Give the server time to start
	time.Sleep(time.Second * 2)

	log.Println("dialing...")
	c, _, err := dialer.Dial(testWsUrl, nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	HostConn = c

	// Read host session ID
	var respSessionId mc.Message[mc.RespSessionId]
	_ = HostConn.ReadJSON(&respSessionId)
	HostSessionID = respSessionId.Payload.SessionID

	c2, _, err := dialer.Dial(testWsUrl, nil)
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
