package test

import (
	"log"

	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"

	"testing"
)

func TestGameMine(t *testing.T) {
	/*
	   Create Mine Game
	*/
	reqCreateGame := mc.Message[mc.ReqCreateGame]{Code: mc.CodeCreateGame, Payload: mc.ReqCreateGame{
		GameDifficulty: mb.GameDifficultyEasy,
		GameMode:       mb.GameModeMine,
	}}
	respCreateGame := mc.NewMessage[mc.RespCreateGame](mc.CodeCreateGame)

	if err := hostClientConn.WriteJSON(reqCreateGame); err != nil {
		t.Fatal(err)
	}
	if err := hostClientConn.ReadJSON(&respCreateGame); err != nil {
		t.Fatal(err)
	}
	gameUuid := respCreateGame.Payload.GameUuid
	game, err := testGameManager.FetchGame(gameUuid)
	if err != nil {
		t.Fatal(err)
	}
	hostPlayer := game.FetchPlayer(true)

	/*
	   Join Mine Game
	*/
	reqJoinGame := mc.NewMessage[mc.ReqJoinGame](mc.CodeJoinGame)
	reqJoinGame.AddPayload(mc.ReqJoinGame{GameUuid: gameUuid})

	if err := joinClientConn.WriteJSON(reqJoinGame); err != nil {
		t.Fatal(err)
	}

	respJoinGame := mc.NewMessage[mc.RespJoinGame](mc.CodeJoinGame)
	if err := joinClientConn.ReadJSON(&respJoinGame); err != nil {
		t.Fatal(err)
	}

	joinPlayer := game.FetchPlayer(false)
	log.Printf("join player resp %+v\n\n", respJoinGame)

	var respSelectGridJoin mc.Message[mc.NoPayload]
	if err := joinClientConn.ReadJSON(&respSelectGridJoin); err != nil {
		t.Fatal(err)
	}
	if respSelectGridJoin.Error != nil {
		t.Fatalf("failed to receive select ready message for join: %s", respSelectGridJoin.Error.ErrorDetails)
	}

	var respSelectGridHost mc.Message[mc.NoPayload]
	if err := hostClientConn.ReadJSON(&respSelectGridHost); err != nil {
		t.Fatal(err)
	}
	if respSelectGridHost.Error != nil {
		t.Fatalf("failed to receive select ready message for host: %s", respSelectGridHost.Error.ErrorDetails)
	}

	/*
	   Ready Mine Game
	*/
	reqReadyHost := mc.NewMessage[mc.ReqReadyPlayer](mc.CodeReady)
	reqReadyHost.AddPayload(mc.ReqReadyPlayer{
		GameUuid:    gameUuid,
		PlayerUuid:  hostPlayer.Uuid(),
		DefenceGrid: defenceGridHost,
	})

    if err := hostClientConn.WriteJSON(reqReadyHost); err != nil {
        t.Fatal(err)
    }

	var respReadyHost mc.Message[mc.RespReady]
	if err := hostClientConn.ReadJSON(&respReadyHost); err != nil {
		t.Fatal(err)
	}

    reqReadyJoin := mc.NewMessage[mc.ReqReadyPlayer](mc.CodeReady)
	reqReadyJoin.AddPayload(mc.ReqReadyPlayer{
		GameUuid:    gameUuid,
		PlayerUuid:  joinPlayer.Uuid(),
		DefenceGrid: defenceGridJoin,
	})

    if err := joinClientConn.WriteJSON(reqReadyJoin); err != nil {
        t.Fatal(err)
    }

	var respReadyJoin mc.Message[mc.RespReady]
	if err := joinClientConn.ReadJSON(&respReadyJoin); err != nil {
		t.Fatal(err)
	}

    // Free up both host and join client connections from StartGame response
    var respStartGameHost mc.Message[mc.NoPayload]
    if err := hostClientConn.ReadJSON(&respStartGameHost); err != nil {
        t.Fatal(err)
    }

    var respStartGameJoin mc.Message[mc.NoPayload]
    if err := joinClientConn.ReadJSON(&respStartGameJoin); err != nil {
        t.Fatal(err)
    }
}
