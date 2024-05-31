package test

import (
	"log"
	"testing"

	md "github.com/saeidalz13/battleship-backend/models"
)

type Test struct {
	Number     int8
	Desc       string
	ReqPayload interface{}
}

func (te *Test) logError() {
	log.Printf("failed: test number %d; desc: %s\n", te.Number, te.Desc)
}

func (te *Test) logSuccess(v interface{}) {
	log.Printf("success: test number %d; desc: %s; resp: %+v\n", te.Number, te.Desc, v)
}

func (te *Test) logStart() {
	log.Printf("\n\nstarting test %d", te.Number)
}

func TestCreateGame(t *testing.T) {
	test := Test{
		Number:     0,
		Desc:       "should fail with invalid code",
		ReqPayload: md.NewMessage[md.NoPayload](-1),
	}
	test.logStart()
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}

	var respErr md.Message[md.NoPayload]
	if err := HostConn.ReadJSON(&respErr); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respErr)

	/*
		Test 1
	*/
	test = Test{
		Number:     1,
		Desc:       "should create game with valid code",
		ReqPayload: md.NewMessage[md.NoPayload](md.CodeCreateGame),
	}
	test.logStart()
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respCreateGame md.Message[md.RespCreateGame]
	if err := HostConn.ReadJSON(&respCreateGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respCreateGame)
	gameUuid := respCreateGame.Payload.GameUuid
	hostUuid := respCreateGame.Payload.HostUuid

	/*
		Test 2
	*/
	joinGameReqPayload := md.NewMessage[md.ReqJoinGame](md.CodeJoinGame)
	joinGameReqPayload.AddPayload(md.ReqJoinGame{GameUuid: gameUuid})
	test = Test{
		Number:     2,
		Desc:       "should join the game with valid game uuid",
		ReqPayload: joinGameReqPayload,
	}

	test.logStart()
	if err := JoinConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respJoinGame md.Message[md.RespJoinGame]
	if err := JoinConn.ReadJSON(&respJoinGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Error.ErrorDetails != "" {
		t.Fatalf("failed to join the game, msg:\t %+v", respJoinGame)
	}
	joinUuid := respJoinGame.Payload.PlayerUuid
	test.logSuccess(respJoinGame)

	// Read extra message of success to host
	// we have to read it so it frees up the queue for the next steps of host read
	// when join player is added, a select grid code is sent to both players
	var respSelectGrid md.Message[md.NoPayload]
	if err := HostConn.ReadJSON(&respSelectGrid); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Error.ErrorDetails != "" {
		t.Fatalf("failed to join the game for the host player\t%+v\t%s", hostUuid, respCreateGame.Error.ErrorDetails)
	}
	if err := JoinConn.ReadJSON(&respSelectGrid); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Error.ErrorDetails != "" {
		t.Fatalf("failed to join the game for the join player\t%+v\t%s", joinUuid, respCreateGame.Error.ErrorDetails)
	}

	/*
		Test 3
	*/
	invalidReqJoinPayload := md.NewMessage[md.ReqJoinGame](md.CodeJoinGame)
	invalidReqJoinPayload.AddPayload(md.ReqJoinGame{GameUuid: "invalid"})
	test = Test{
		Number:     3,
		Desc:       "should fail with invalid game uuid",
		ReqPayload: invalidReqJoinPayload,
	}
	if err := JoinConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respFailJoin md.Message[md.RespJoinGame]
	if err := JoinConn.ReadJSON(&respFailJoin); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respFailJoin.Error.ErrorDetails == "" {
		test.logError()
		t.Fatal("must have failed")
	}
	test.logSuccess(respFailJoin)

	/*
		test 4
	*/
	defenceGridHost := md.GridInt{
		{0, md.PositionStateDefenceDestroyer, md.PositionStateDefenceDestroyer, 0, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{0, 0, 0, md.PositionStateDefenceBattleship, 0},
	}
	readyReqPayload := md.NewMessage[md.ReqReadyPlayer](md.CodeReady)
	readyReqPayload.AddPayload(md.ReqReadyPlayer{
		DefenceGrid: defenceGridHost,
		GameUuid:    gameUuid,
		PlayerUuid:  hostUuid,
	})
	test = Test{
		Number:     4,
		Desc:       "should set the defence grid and set ready for host",
		ReqPayload: readyReqPayload,
	}
	test.logStart()
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respSuccessReady md.Message[md.NoPayload]
	if err := HostConn.ReadJSON(&respSuccessReady); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respSuccessReady)

	/*
		Test 5
	*/
	defenceGridJoin := md.GridInt{
		{0, md.PositionStateDefenceDestroyer, md.PositionStateDefenceDestroyer, 0, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{0, 0, 0, md.PositionStateDefenceBattleship, 0},
	}
	readyJoin := md.NewMessage[md.ReqReadyPlayer](md.CodeReady)
	readyJoin.AddPayload(md.ReqReadyPlayer{
		DefenceGrid: defenceGridJoin,
		GameUuid:    gameUuid,
		PlayerUuid:  joinUuid,
	})
	test = Test{
		Number:     5,
		Desc:       "should set the defence grid for join and send back start game code",
		ReqPayload: readyJoin,
	}
	test.logStart()
	if err := JoinConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respSuccessReadyJoin md.Message[md.NoPayload]
	if err := JoinConn.ReadJSON(&respSuccessReadyJoin); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respSuccessReadyJoin)

	// Reading game ready codes
	// Host
	var respStartGameHost md.Message[md.NoPayload]
	if err := HostConn.ReadJSON(&respStartGameHost); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respStartGameHost)

	// Join
	var respStartGameJoin md.Message[md.NoPayload]
	if err := JoinConn.ReadJSON(&respStartGameJoin); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respStartGameJoin)
}
