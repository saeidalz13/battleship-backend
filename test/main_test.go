package test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/api"
	"github.com/saeidalz13/battleship-backend/db/sqlc"

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
	hostClientConn *websocket.Conn
	joinClientConn *websocket.Conn

	testHostPlayer *mb.BattleshipPlayer
	testJoinPlayer *mb.BattleshipPlayer

	hostSessionID string
	joinSessionID string

	hostSession *mc.Session
	joinSession *mc.Session

	testRp api.RequestProcessor
	dialer = websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	testMock           sqlmock.Sqlmock
	testGameManager    *mb.BattleshipGameManager
	testSessionManager *mc.BattleshipSessionManager
	testQuerier        sqlc.Querier

	defenceGridHost = mb.Grid{
		{0, mb.PositionStateDefenceDestroyer, mb.PositionStateDefenceDestroyer, 0, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{0, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	defenceGridJoin = mb.Grid{
		{0, mb.PositionStateDefenceDestroyer, mb.PositionStateDefenceDestroyer, 0, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{0, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{0, 0, 0, 0, 0, 0},
	}
)

func TestMain(m *testing.M) {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	testMock = mock

	go func() {
		// test db manager
		querier := sqlc.New(db)
		testQuerier = querier

		// test session manager
		bsm := mc.NewBattleshipSessionManager()
		testSessionManager = bsm
		go bsm.CleanupPeriodically()

		// test game manager
		bgm := mb.NewBattleshipGameManager()
		testGameManager = bgm

		rp := api.NewRequestProcessor(bsm, bgm, querier)
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
	hostClientConn = c

	// Read host session ID
	var respSessionId mc.Message[mc.RespSessionId]
	_ = hostClientConn.ReadJSON(&respSessionId)
	hostSessionID = respSessionId.Payload.SessionID
	hs, err := testSessionManager.FindSession(hostSessionID)
	if err != nil {
		log.Fatalln(err)
	}
	hostSession = hs

	c2, _, err := dialer.Dial(testWsUrl, nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	joinClientConn = c2

	// Read Join sessoin ID
	_ = joinClientConn.ReadJSON(&respSessionId)
	joinSessionID = respSessionId.Payload.SessionID
	js, err := testSessionManager.FindSession(joinSessionID)
	if err != nil {
		log.Fatalln(err)
	}
	joinSession = js

	log.Println("Host session ID:", hostSessionID)
	log.Println("Join session ID:", joinSessionID)
	log.Printf("host: %s\tjoin: %s", hostClientConn.LocalAddr().String(), joinClientConn.LocalAddr().String())
	os.Exit(m.Run())
}
