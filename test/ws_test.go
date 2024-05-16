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

func TestCreateGame(t *testing.T) {
	test := Test{
		Number:     0,
		Desc:       "should fail with invalid code",
		ReqPayload: md.NewMessage[any](-1),
	}
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}

	var respErr md.Message[any]
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
		ReqPayload: md.NewMessage[any](md.CodeCreateGame),
	}
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

	if err := JoinConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respJoinGame md.Message[md.RespJoinGame]
	if err := JoinConn.ReadJSON(&respJoinGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Code != md.CodeJoinGame {
		t.Fatalf("failed to join the game, msg:\t %+v", respJoinGame)
	}
	test.logSuccess(respJoinGame)

	// Read extra message of success to host
	// we have to read it so it frees up the queue for the next steps of host read
	if err := HostConn.ReadJSON(&respJoinGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Error.ErrorDetails != "" {
		t.Fatalf("failed to join the game for the join player\t%+v\t%s", hostUuid, respCreateGame.Error.ErrorDetails)
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
	readyReqPayload := md.NewMessage[md.ReqReadyPlayer](md.CodeReady)
	readyReqPayload.AddPayload(md.ReqReadyPlayer{
		DefenceGrid: md.NewGrid(),
		GameUuid:    gameUuid,
		PlayerUuid:  hostUuid,
	})
	test = Test{
		Number:     4,
		Desc:       "should set the defence grid and set ready",
		ReqPayload: readyReqPayload,
	}
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respSuccessReady md.Message[md.RespReadyPlayer]
	if err := HostConn.ReadJSON(&respSuccessReady); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respSuccessReady)

	/*
		test 5
	*/

	// test = Test{
	// 	Number: 5,
	// 	Desc:   "should adjust the grid and turn for both host and join player",
	// 	ReqPayload: md.NewMessage(md.CodeReqAttack,
	// 		md.WithPayload(md.ReqAttack{
	// 			GameUuid:      gameUuid,
	// 			PlayerUuid:    hostUuid,
	// 			X:             3,
	// 			Y:             2,
	// 			PositionState: md.PositionStateHit,
	// 		}),
	// 	),
	// }

	// if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
	// 	test.logError()
	// 	t.Fatal(err)
	// }

	// var respAttack md.Message
	// if err := HostConn.ReadJSON(&respAttack); err != nil {
	// 	test.logError()
	// 	t.Fatal(err)
	// }
	// test.logSuccess(respAttack)
}
