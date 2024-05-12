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
	var respMessage md.Message

	test := Test{
		Number:     0,
		Desc:       "should fail with invalid code",
		ReqPayload: md.NewMessage(-1),
	}
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}

	if err := HostConn.ReadJSON(&respMessage); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respMessage)

	/*
		Test 1
	*/
	test = Test{
		Number:     1,
		Desc:       "should create game with valid code",
		ReqPayload: md.NewMessage(md.CodeReqCreateGame),
	}
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respCreateGame md.Message
	if err := HostConn.ReadJSON(&respCreateGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respCreateGame)

	resp, ok := respCreateGame.Payload.(map[string]interface{})
	if !ok {
		t.Fatal("payload of create game response is nil")
	}
	createdGameUuid, prs := resp["game_uuid"]
	if !prs {
		t.Fatal("payload of create does not contain the key 'game_uuid'")
	}
	gameUuid, ok := createdGameUuid.(string)
	if !ok {
		t.Fatal("game_uuid is not of type string")
	}
	createdPlayerUuid, prs := resp["host_uuid"]
	if !prs {
		t.Fatal("payload of create does not contain the key 'host_uuid'")
	}
	hostUuid, ok := createdPlayerUuid.(string)
	if !ok {
		t.Fatal("host_uuid is not of type string")
	}

	/*
		Test 2
	*/
	test = Test{
		Number:     2,
		Desc:       "should join the game with valid game uuid",
		ReqPayload: md.NewMessage(md.CodeReqJoinGame, md.WithPayload(md.ReqJoinGame{GameUuid: gameUuid})),
	}

	if err := JoinConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respJoinGame md.Message
	if err := JoinConn.ReadJSON(&respJoinGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Code != md.CodeRespSuccessJoinGame {
		t.Fatalf("failed to join the game, msg:\t %+v", respJoinGame)
	}
	test.logSuccess(respJoinGame)

	// Read extra message of success to host
	// we have to read it so it frees up the queue for the next steps of host read
	if err := HostConn.ReadJSON(&respJoinGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Code != md.CodeRespSuccessJoinGame {
		t.Fatalf("failed to join the game for the join player\t%+v", hostUuid)
	}

	/*
		Test 3
	*/
	test = Test{
		Number:     3,
		Desc:       "should fail with invalid game uuid",
		ReqPayload: md.NewMessage(md.CodeReqJoinGame, md.WithPayload(md.ReqJoinGame{GameUuid: "invalid"})),
	}
	if err := JoinConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respFailJoin md.Message
	if err := JoinConn.ReadJSON(&respFailJoin); err != nil {
		test.logError()
		t.Fatal(err)
	}

	if respFailJoin.Code != md.CodeRespFailJoinGame {
		t.Fatalf("should have failed to join and received code %d but got %d", md.CodeRespFailJoinGame, respFailJoin.Code)
	}
	test.logSuccess(respFailJoin)

	/*
		test 4
	*/
	test = Test{
		Number: 4,
		Desc:   "should set the defence grid and set ready",
		ReqPayload: md.Message{
			Code: md.CodeReqReady,
			Payload: md.ReqReadyPlayer{
				DefenceGrid: md.NewGrid(),
				GameUuid:    gameUuid,
				PlayerUuid:  hostUuid,
			},
		},
	}
	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respSuccessReady md.Message
	if err := HostConn.ReadJSON(&respSuccessReady); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respSuccessReady)

	/*
		test 5
	*/
	test = Test{
		Number: 5,
		Desc:   "should adjust the grid and turn for both host and join player",
		ReqPayload: md.NewMessage(md.CodeReqAttack,
			md.WithPayload(md.ReqAttack{
				GameUuid:      gameUuid,
				PlayerUuid:    hostUuid,
				X:             3,
				Y:             2,
				PositionState: md.PositionStateHit,
			}),
		),
	}

	if err := HostConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}

	var respAttack md.Message
	if err := HostConn.ReadJSON(&respAttack); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respAttack)
}
