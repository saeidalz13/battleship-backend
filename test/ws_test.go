package test

import (
	"testing"

	"github.com/saeidalz13/battleship-backend/models"
)

type Test struct {
	Desc        string
	ReqPayload  interface{}
	RespPayload interface{}
}

func TestCreateGame(t *testing.T) {
	test := Test{
		Desc:        "should pass and receive the response in the defined structs",
		ReqPayload:  models.Signal{Code: models.CodeReqCreateGame},
		RespPayload: &models.RespCreateGame{},
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		t.Fatal(err)
	}

	if err := ClientConn.ReadJSON(test.RespPayload); err != nil {
		t.Fatal(err)
	}

	test = Test{
		Desc:        "should fail with invalid code in the payload",
		ReqPayload:  models.Signal{Code: -1},
		RespPayload: &models.RespFail{},
	}
	if err := ClientConn.WriteJSON(test.ReqPayload); err != nil {
		t.Fatal(err)
	}

	if err := ClientConn.ReadJSON(test.RespPayload); err != nil {
		t.Fatal(err)
	}
}
