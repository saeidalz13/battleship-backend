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
		ReqPayload: md.NewMessage(-1),
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respFail md.RespFail
	if err := ClientConn.ReadJSON(&respFail); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respFail)

	test = Test{
		Number:     1,
		Desc:       "should create game with valid code",
		ReqPayload: md.NewMessage(md.CodeReqCreateGame),
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}

	var respCreateGame md.RespCreateGame
	if err := ClientConn.ReadJSON(&respCreateGame); err != nil {
		test.logError()
		t.Fatal(err)
	}
	test.logSuccess(respCreateGame)
	createdGameUuid := respCreateGame.GameUuid
	createdPlayerUuid := respCreateGame.HostUuid

	test = Test{
		Number:     2,
		Desc:       "should join the game with valid game uuid",
		ReqPayload: md.NewMessage(md.CodeReqJoinGame, md.WithPayload(md.ReqJoinGame{GameUuid: createdGameUuid})),
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		test.logError()
		t.Fatal(err)
	}
	var respJoinGame md.RespJoinGame
	if err := ClientConn.ReadJSON(&respJoinGame); err != nil {
		test.logError()
		t.Fatal(err)
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
				GameUuid:    createdGameUuid,
				PlayerUuid:  createdPlayerUuid,
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
