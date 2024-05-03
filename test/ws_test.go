package test

import (
	"log"
	"testing"

	"github.com/saeidalz13/battleship-backend/models"
)

type Test struct {
	Number     int8
	Desc       string
	ReqPayload interface{}
}

func TestCreateGame(t *testing.T) {
	test := Test{
		Number:     0,
		Desc:       "should fail with invalid code",
		ReqPayload: models.Signal{Code: -1},
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		t.Fatal(err)
	}
	var respFail models.RespFail
	if err := ClientConn.ReadJSON(&respFail); err != nil {
		t.Fatal(err)
	}
	log.Printf("%+v", respFail)

	test = Test{
		Number:     1,
		Desc:       "should pass with valid code",
		ReqPayload: models.Signal{Code: models.CodeReqCreateGame},
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		t.Fatal(err)
	}

	var respCreateGame models.RespCreateGame
	if err := ClientConn.ReadJSON(&respCreateGame); err != nil {
		t.Fatal(err)
	}
	log.Printf("%+v", respCreateGame)
	createdGameUuid := respCreateGame.GameUuid
	// createdPlaerUuid := respCreateGame.HostUuid

	test = Test{
		Number:     2,
		Desc:       "should join the game with valid game uuid",
		ReqPayload: models.ReqJoinGame{Code: models.CodeReqJoinGame, GameUuid: createdGameUuid},
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		t.Fatal(err)
	}
	var respJoinGame models.RespJoinGame
	if err := ClientConn.ReadJSON(&respJoinGame); err != nil {
		t.Fatal(err)
	}
	log.Printf("response success join game: %+v", respJoinGame)

	test = Test{
		Number:     3,
		Desc:       "should fail with invalid game uuid",
		ReqPayload: models.ReqJoinGame{Code: models.CodeReqJoinGame, GameUuid: "invalid"},
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		t.Fatal(err)
	}
	var respFailJoin models.RespFail
	if err := ClientConn.ReadJSON(&respFailJoin); err != nil {
		t.Fatal(err)
	}
	log.Printf("%+v", respFailJoin)
}
