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
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}

	if err := ClientConn.ReadJSON(&respMessage); err != nil {
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
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respCreateGame md.Message
	if err := ClientConn.ReadJSON(&respCreateGame); err != nil {
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
	playerUuid, ok := createdPlayerUuid.(string)
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

	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respJoinGame md.Message
	if err := ClientConn.ReadJSON(&respJoinGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	if respJoinGame.Code != md.CodeRespSuccessJoinGame {
		t.Fatalf("failed to join the game\tplayer: %s", playerUuid)
	}
	test.logSuccess(respJoinGame)

	test = Test{
		Number:     3,
		Desc:       "should fail with invalid game uuid",
		ReqPayload: md.NewMessage(md.CodeReqJoinGame, md.WithPayload(md.ReqJoinGame{GameUuid: "invalid"})),
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respFailJoin md.RespFail
	if err := ClientConn.ReadJSON(&respFailJoin); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respFailJoin)

	test = Test{
		Number: 4,
		Desc:   "should set the defence grid and set ready",
		ReqPayload: md.Message{
			Code: md.CodeReqReady,
			Payload: md.ReqReadyPlayer{
				DefenceGrid: md.NewGrid(),
				GameUuid:    gameUuid,
				PlayerUuid:  playerUuid,
			},
		},
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respSuccessReady md.RespReadyPlayer
	if err := ClientConn.ReadJSON(&respSuccessReady); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respSuccessReady)
}
