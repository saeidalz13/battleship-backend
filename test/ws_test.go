package test

import (
	"log"
	"reflect"
	"testing"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

type Test[T, K any] struct {
	name string

	expectedCode int
	expectedErr  string

	reqPayload          T
	respPayload         K // Used to unmarshal the response
	expectedRespPayload K // To compare to data unmarshaled in respPayload

	conn      *websocket.Conn
	otherConn *websocket.Conn // Used for attack when we need to know defender
}

func TestInvalidCode(t *testing.T) {
	tests := []Test[mc.Message[mc.NoPayload], mc.Message[mc.NoPayload]]{
		{
			name:         "random invalid code 1",
			expectedCode: mc.CodeInvalidSignal,
			reqPayload:   mc.NewMessage[mc.NoPayload](-1),
			respPayload:  mc.NewMessage[mc.NoPayload](mc.CodeInvalidSignal),
			conn:         HostConn,
		},
		{
			name:         "random invalid code 2",
			expectedCode: mc.CodeInvalidSignal,
			reqPayload:   mc.NewMessage[mc.NoPayload](999),
			respPayload:  mc.NewMessage[mc.NoPayload](mc.CodeInvalidSignal),
			conn:         JoinConn,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.conn.WriteJSON(test.reqPayload); err != nil {
				t.Fatal(err)
			}

			if err := test.conn.ReadJSON(&test.respPayload); err != nil {
				t.Fatal(err)
			}

			if test.respPayload.Code != test.expectedCode {
				t.Fatalf("expected status: %d\t got: %d", test.expectedCode, test.respPayload.Code)
			}
		})
	}
}

func TestCreateGame(t *testing.T) {
	tests := []Test[mc.Message[mc.ReqCreateGame], mc.Message[mc.RespCreateGame]]{
		{
			name:         "create game valid code",
			expectedCode: mc.CodeCreateGame,
			reqPayload: mc.Message[mc.ReqCreateGame]{Code: mc.CodeCreateGame, Payload: mc.ReqCreateGame{
				GameDifficulty: mb.GameDifficultyEasy,
			}},
			respPayload: mc.NewMessage[mc.RespCreateGame](mc.CodeCreateGame),
			conn:        HostConn,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.conn.WriteJSON(test.reqPayload); err != nil {
				t.Fatal(err)
			}

			if err := test.conn.ReadJSON(&test.respPayload); err != nil {
				t.Fatal(err)
			}

			if test.respPayload.Code != test.expectedCode {
				t.Fatalf("expected status: %d\t got: %d", test.expectedCode, test.respPayload.Code)
			}

			if test.respPayload.Error != nil {
				t.Fatalf("error: %s\t", test.reqPayload.Error.ErrorDetails)
			}

			if test.respPayload.Error == nil {
				GameUuid = test.respPayload.Payload.GameUuid
				game, err := testServer.GameManager.FindGame(GameUuid)
				if err != nil {
					t.Fatal(err)
				}
				testHostPlayer = game.HostPlayer
			}
		})
	}
}

func TestJoinPlayer(t *testing.T) {
	tests := []Test[mc.Message[mc.ReqJoinGame], mc.Message[mc.RespJoinGame]]{
		{
			name:         "valid game uuid",
			expectedCode: mc.CodeJoinGame,
			reqPayload:   mc.Message[mc.ReqJoinGame]{Code: mc.CodeJoinGame, Payload: mc.ReqJoinGame{GameUuid: GameUuid}},
			respPayload:  mc.NewMessage[mc.RespJoinGame](mc.CodeJoinGame),
			conn:         JoinConn,
		},
		// Any invalid join request will close the connection
		// {
		// 	name:         "invalid game uuid",
		// 	expectedCode: mc.CodeJoinGame,
		// 	reqPayload:   mc.Message[mc.ReqJoinGame]{Code: mc.CodeJoinGame, Payload: mc.ReqJoinGame{GameUuid: "-1invalid"}},
		// 	respPayload:  mc.NewMessage[mc.RespJoinGame](mc.CodeJoinGame),
		// 	conn:         JoinConn,
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.conn.WriteJSON(test.reqPayload); err != nil {
				t.Fatal(err)
			}

			if err := test.conn.ReadJSON(&test.respPayload); err != nil {
				t.Fatal(err)
			}

			if test.respPayload.Error == nil {
				if GameUuid != test.respPayload.Payload.GameUuid {
					t.Fatal("incoming game uuid did not match the req uuid after join")
				}
				// if it was successful, join player id is set to the response
				game, err := testServer.GameManager.FindGame(GameUuid)
				if err != nil {
					t.Fatal(err)
				}
				testJoinPlayer = game.JoinPlayer

				// Read extra message of success to host
				// we have to read it so it frees up the queue for the next steps of host read
				// when join player is added, a select grid code is sent to both players
				var respSelectGridHost mc.Message[mc.NoPayload]
				if err := HostConn.ReadJSON(&respSelectGridHost); err != nil {
					t.Fatal(err)
				}
				if respSelectGridHost.Error != nil {
					t.Fatalf("failed to receive select ready message for host: %s", respSelectGridHost.Error.ErrorDetails)
				}

				var respSelectGridJoin mc.Message[mc.NoPayload]
				if err := JoinConn.ReadJSON(&respSelectGridJoin); err != nil {
					t.Fatal(err)
				}
				if respSelectGridJoin.Error != nil {
					t.Fatalf("failed to receive select ready message for join: %s", respSelectGridJoin.Error.ErrorDetails)
				}
			}
		})
	}
}

func TestReadyGame(t *testing.T) {
	defenceGridHost := mb.Grid{
		{0, mb.PositionStateDefenceDestroyer, mb.PositionStateDefenceDestroyer, 0, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{0, 0, 0, mb.PositionStateDefenceBattleship, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	defenceGridJoin := mb.Grid{
		{0, mb.PositionStateDefenceDestroyer, mb.PositionStateDefenceDestroyer, 0, 0, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{mb.PositionStateDefenceCruiser, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{0, 0, 0, 0, mb.PositionStateDefenceBattleship, 0},
		{0, 0, 0, 0, 0, 0},
	}

	tests := []Test[mc.Message[mc.ReqReadyPlayer], mc.Message[mc.NoPayload]]{
		{
			name:         "set defence grid ready host",
			expectedCode: mc.CodeReady,
			reqPayload: mc.Message[mc.ReqReadyPlayer]{
				Code: mc.CodeReady,
				Payload: mc.ReqReadyPlayer{
					DefenceGrid: defenceGridHost,
					GameUuid:    GameUuid,
					PlayerUuid:  testHostPlayer.Uuid,
				},
			},
			respPayload: mc.Message[mc.NoPayload]{},
			conn:        HostConn,
		},
		{
			name:         "set defence grid ready join",
			expectedCode: mc.CodeReady,
			reqPayload: mc.Message[mc.ReqReadyPlayer]{
				Code: mc.CodeReady,
				Payload: mc.ReqReadyPlayer{
					DefenceGrid: defenceGridJoin,
					GameUuid:    GameUuid,
					PlayerUuid:  testJoinPlayer.Uuid,
				},
			},
			respPayload: mc.Message[mc.NoPayload]{},
			conn:        JoinConn,
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.conn.WriteJSON(test.reqPayload); err != nil {
				t.Fatal(err)
			}

			if err := test.conn.ReadJSON(&test.respPayload); err != nil {
				t.Fatal(err)
			}

			if test.respPayload.Code != test.expectedCode {
				t.Fatalf("expected status: %d\t got: %d", test.expectedCode, test.respPayload.Code)
			}

			if test.respPayload.Error != nil {
				if test.respPayload.Error.ErrorDetails != test.expectedErr {
					t.Fatalf("expected error: %s\t got: %s", test.reqPayload.Error.ErrorDetails, test.expectedErr)
				}
			}

			// After the success of second test
			// start game code will be sent to both parties
			if i == 1 {
				// Reading game ready codes

				// Host
				var respStartGameHost mc.Message[mc.NoPayload]
				if err := HostConn.ReadJSON(&respStartGameHost); err != nil {
					t.Fatal(err)
				}

				// Join
				var respStartGameJoin mc.Message[mc.NoPayload]
				if err := JoinConn.ReadJSON(&respStartGameJoin); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestAttack(t *testing.T) {
	tests := []Test[mc.Message[mc.ReqAttack], mc.Message[mc.RespAttack]]{
		{
			name:         "successful hit attack destroyer valid payload host 1",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          0,
				Y:          1,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               0,
				Y:               1,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 0,
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},

		{
			name:         "successful hit attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          0,
				Y:          1,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               0,
				Y:               1,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 0,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		{
			name:         "successful hit attack destroyer valid payload 2 and sunk destroyer",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          0,
				Y:          2,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               0,
				Y:               2,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 1,
				DefenderSunkenShipsCoords: []mb.Coordinates{
					{X: 0, Y: 1},
					{X: 0, Y: 2},
				},
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},

		{
			name:         "successful miss attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          0,
				Y:          0,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               0,
				Y:               0,
				PositionState:   mb.PositionStateAttackGridMiss,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 1,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		{
			name:         "wrong turn of player join",
			expectedCode: mc.CodeAttack,
			expectedErr:  cerr.ErrNotTurnForAttacker(testJoinPlayer.Uuid).Error(),
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          0,
				Y:          0,
			}},
			respPayload:         mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack},
			conn:                JoinConn,
			otherConn:           HostConn,
		},

		{
			name:         "invalid x attack host",
			expectedCode: mc.CodeAttack,
			expectedErr:  cerr.ErrXorYOutOfGridBound(-1, 0).Error(),
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          -1,
				Y:          0,
			}},
			respPayload:         mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack},
			conn:                HostConn,
			otherConn:           JoinConn,
		},

		{
			name:         "invalid y attack host",
			expectedCode: mc.CodeAttack,
			expectedErr:  cerr.ErrXorYOutOfGridBound(0, -1).Error(),
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          0,
				Y:          -1,
			}},
			respPayload:         mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack},
			conn:                HostConn,
			otherConn:           JoinConn,
		},

		{
			name:         "invalid attack already hit host",
			expectedCode: mc.CodeAttack,
			expectedErr:  cerr.ErrAttackPositionAlreadyFilled(0, 1).Error(),
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          0,
				Y:          1,
			}},
			respPayload:         mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack},
			conn:                HostConn,
			otherConn:           JoinConn,
		},

		/*
			Sinking Cruiser
		*/
		{
			name:         "successful hit attack cruiser valid payload host 1",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          1,
				Y:          0,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               1,
				Y:               0,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 1,
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},
		{
			name:         "successful miss attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          0,
				Y:          3,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               0,
				Y:               3,
				PositionState:   mb.PositionStateAttackGridMiss,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 1,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		{
			name:         "successful hit attack cruiser valid payload host 2",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          2,
				Y:          0,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               2,
				Y:               0,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 1,
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},
		{
			name:         "successful miss attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          0,
				Y:          4,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               0,
				Y:               4,
				PositionState:   mb.PositionStateAttackGridMiss,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 1,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		{
			name:         "successful hit attack cruiser valid payload host 3 and sunk cruiser",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          3,
				Y:          0,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               3,
				Y:               0,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
				DefenderSunkenShipsCoords: []mb.Coordinates{
					{X: 1, Y: 0},
					{X: 2, Y: 0},
					{X: 3, Y: 0},
				},
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},
		{
			name:         "successful miss attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          0,
				Y:          5,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               0,
				Y:               5,
				PositionState:   mb.PositionStateAttackGridMiss,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		/*
			Sinking Battleship
		*/
		{
			name:         "successful hit attack battleship valid payload hos 1",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          1,
				Y:          4,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               1,
				Y:               4,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},
		{
			name:         "successful miss attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          5,
				Y:          0,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               5,
				Y:               0,
				PositionState:   mb.PositionStateAttackGridMiss,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		{
			name:         "successful hit attack battleship valid payload host 2",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          2,
				Y:          4,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               2,
				Y:               4,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},
		{
			name:         "successful miss attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          5,
				Y:          1,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               5,
				Y:               1,
				PositionState:   mb.PositionStateAttackGridMiss,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		{
			name:         "successful hit attack battleship valid payload host 3",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          3,
				Y:          4,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               3,
				Y:               4,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},
		{
			name:         "successful miss attack valid payload join",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testJoinPlayer.Uuid,
				X:          5,
				Y:          2,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               5,
				Y:               2,
				PositionState:   mb.PositionStateAttackGridMiss,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 2,
			}},
			conn:      JoinConn,
			otherConn: HostConn,
		},

		// Final attack that sinks the last ship
		{
			name:         "final hit attack valid battleship payload host 4 and sunk battleship",
			expectedCode: mc.CodeAttack,
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   GameUuid,
				PlayerUuid: testHostPlayer.Uuid,
				X:          4,
				Y:          4,
			}},
			respPayload: mc.Message[mc.RespAttack]{},
			expectedRespPayload: mc.Message[mc.RespAttack]{Code: mc.CodeAttack, Payload: mc.RespAttack{
				X:               4,
				Y:               4,
				PositionState:   mb.PositionStateAttackGridHit,
				IsTurn:          false,
				SunkenShipsHost: 0,
				SunkenShipsJoin: 3,
				DefenderSunkenShipsCoords: []mb.Coordinates{
					{X: 1, Y: 4},
					{X: 2, Y: 4},
					{X: 3, Y: 4},
					{X: 4, Y: 4},
				},
			}},
			conn:      HostConn,
			otherConn: JoinConn,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if err := test.conn.WriteJSON(test.reqPayload); err != nil {
				t.Fatal(err)
			}

			if err := test.conn.ReadJSON(&test.respPayload); err != nil {
				t.Fatal(err)
			}

			if test.respPayload.Code != test.expectedCode {
				t.Fatalf("expected status: %d\t got: %d", test.expectedCode, test.respPayload.Code)
			}

			if test.respPayload.Error != nil {
				log.Printf("%+v\n\n\n", test.respPayload.Error)
				if test.respPayload.Error.ErrorDetails != test.expectedErr {
					t.Fatalf("expected error: %s\t got: %s", test.reqPayload.Error.ErrorDetails, test.expectedErr)
				}

			} else {
				if !reflect.DeepEqual(test.respPayload, test.expectedRespPayload) {
					t.Fatalf("expected resp payload: %+v\n got: %+v", test.expectedRespPayload, test.respPayload)
				}

				// If the attack was successful (in terms of operation and no error occurred)
				// a resp is sent to both players. Here we read from the defender connection
				// to empty the queue
				var respJoin mc.RespAttack
				if err := test.otherConn.ReadJSON(&respJoin); err != nil {
					t.Fatal(err)
				}

				if test.name == "final hit attack valid payload host and sunk battleship" {
					// When the game ends, both players receive this message
					var endGameResp mc.Message[mc.RespEndGame]
					if err := HostConn.ReadJSON(&endGameResp); err != nil {
						t.Fatal(err)
					}
					if err := JoinConn.ReadJSON(&endGameResp); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}
