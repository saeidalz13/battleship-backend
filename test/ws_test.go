package test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
	"github.com/sqlc-dev/pqtype"
)

type Test[T, K any] struct {
	name string

	expectedCode uint8
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
			name:         "random invalid code host",
			expectedCode: mc.CodeInvalidSignal,
			reqPayload:   mc.NewMessage[mc.NoPayload](255),
			respPayload:  mc.NewMessage[mc.NoPayload](mc.CodeInvalidSignal),
			conn:         HostConn,
		},
		{
			name:         "random invalid code join",
			expectedCode: mc.CodeInvalidSignal,
			reqPayload:   mc.NewMessage[mc.NoPayload](200),
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
				gameUuid := test.respPayload.Payload.GameUuid
				game, err := testGameManager.GetGame(gameUuid)
				if err != nil {
					t.Fatal(err)
				}
				testGame = game
				testGameUuid = gameUuid

				hostPlayer := testGameManager.GetPlayer(game, true)
				testHostPlayer = hostPlayer

				testMock.ExpectQuery(`SELECT games_created FROM game_server_analytics WHERE server_ip = \$1`).
					WithArgs(pqtype.Inet{IPNet: testServer.GetIpNet(), Valid: true}).
					WillReturnRows(sqlmock.NewRows([]string{"games_created"}).AddRow(1))

				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				gamesCreated, err := testDbManager.Analytics.GetGamesCreatedCount(ctx, pqtype.Inet{IPNet: testServer.GetIpNet(), Valid: true})
				if err != nil {
					t.Fatalf("failed to fetch created games: %v", err)
				}

				if gamesCreated != 1 {
					t.Fatalf("expected number of created games: %d\tgot: %d", 1, gamesCreated)
				}

				if err = testMock.ExpectationsWereMet(); err != nil {
					t.Fatalf("expectations were not met: %v", err)
				}
			}
		})
	}
}

func TestJoinPlayer(t *testing.T) {
	tests := []Test[mc.Message[mc.ReqJoinGame], mc.Message[mc.RespJoinGame]]{
		{
			name:         "valid game uuid",
			expectedCode: mc.CodeJoinGame,
			reqPayload:   mc.Message[mc.ReqJoinGame]{Code: mc.CodeJoinGame, Payload: mc.ReqJoinGame{GameUuid: testGameUuid}},
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
				if testGameUuid != test.respPayload.Payload.GameUuid {
					t.Fatal("incoming game uuid did not match the req uuid after join")
				}
				// if it was successful, join player id is set to the response
				joinPlayer := testGameManager.GetPlayer(testGame, false)
				testJoinPlayer = joinPlayer

				// Read extra message of success to host
				// we have to read it so it frees up the queue for the next steps of host read
				// when join player is added, a select grid code is sent to both players
				var respSelectGridJoin mc.Message[mc.NoPayload]
				if err := JoinConn.ReadJSON(&respSelectGridJoin); err != nil {
					t.Fatal(err)
				}
				if respSelectGridJoin.Error != nil {
					t.Fatalf("failed to receive select ready message for join: %s", respSelectGridJoin.Error.ErrorDetails)
				}

				var respSelectGridHost mc.Message[mc.NoPayload]
				if err := HostConn.ReadJSON(&respSelectGridHost); err != nil {
					t.Fatal(err)
				}
				if respSelectGridHost.Error != nil {
					t.Fatalf("failed to receive select ready message for host: %s", respSelectGridHost.Error.ErrorDetails)
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
					GameUuid:    testGameUuid,
					PlayerUuid:  testHostPlayer.GetUuid(),
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
					GameUuid:    testGameUuid,
					PlayerUuid:  testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
			expectedErr:  cerr.ErrNotTurnForAttacker(testJoinPlayer.GetUuid()).Error(),
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
			expectedErr:  cerr.ErrXorYOutOfGridBound(outOfGridBoundNum, 0).Error(),
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
				X:          outOfGridBoundNum,
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
			expectedErr:  cerr.ErrXorYOutOfGridBound(0, outOfGridBoundNum).Error(),
			reqPayload: mc.Message[mc.ReqAttack]{Code: mc.CodeAttack, Payload: mc.ReqAttack{
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
				X:          0,
				Y:          outOfGridBoundNum,
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testJoinPlayer.GetUuid(),
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
				GameUuid:   testGameUuid,
				PlayerUuid: testHostPlayer.GetUuid(),
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
				if test.respPayload.Error.ErrorDetails != test.expectedErr {
					t.Fatalf("expected error: %s\t got: %s", test.expectedErr, test.respPayload.Error.ErrorDetails)
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

				if test.name == "final hit attack valid battleship payload host 4 and sunk battleship" {
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

func TestRematchAcceptance(t *testing.T) {
	// Host client sends a rematch call
	msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCall)
	if err := HostConn.WriteJSON(msg); err != nil {
		t.Fatal(err)
	}

	// Test database
	testMock.ExpectQuery(`SELECT rematch_called FROM game_server_analytics WHERE server_ip = \$1`).
		WithArgs(pqtype.Inet{IPNet: testServer.GetIpNet(), Valid: true}).
		WillReturnRows(sqlmock.NewRows([]string{"games_created"}).AddRow(1))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	rematchCalls, err := testDbManager.Analytics.GetRematchCalledCount(ctx, pqtype.Inet{IPNet: testServer.GetIpNet(), Valid: true})
	if err != nil {
		t.Fatalf("failed to fetch created games: %v", err)
	}

	if err = testMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}

	if rematchCalls != 1 {
		t.Fatal("rematch calls must be 1 at this point ")
	}

	// Join client receives this rematch call
	var rematchCall mc.Message[mc.NoPayload]
	if err := JoinConn.ReadJSON(&rematchCall); err != nil {
		t.Fatal(err)
	}

	// Join client sends this msg to server
	msg = mc.NewMessage[mc.NoPayload](mc.CodeRematchCallAccepted)
	if err := JoinConn.WriteJSON(msg); err != nil {
		t.Fatal(err)
	}

	// Host client reads this rematch response including their turn
	var rematchHost mc.Message[mc.RespRematch]
	if err := HostConn.ReadJSON(&rematchHost); err != nil {
		t.Fatal(err)
	}
	// Join client reads this rematch response including their turn
	var rematchJoin mc.Message[mc.RespRematch]
	if err := JoinConn.ReadJSON(&rematchJoin); err != nil {
		t.Fatal(err)
	}

	if testHostPlayer.GetMatchStatus() != mb.PlayerMatchStatusUndefined || testJoinPlayer.GetMatchStatus() != mb.PlayerMatchStatusUndefined {
		t.Fatalf("both players match status must be undefined after reset game but\n host: %d join: %d", testHostPlayer.GetMatchStatus(), testJoinPlayer.GetMatchStatus())
	}

	if testHostPlayer.IsReady() || testJoinPlayer.IsReady() {
		t.Fatal("both players is ready must be false but it is still true")
	}

	// if !reflect.DeepEqual(testHostPlayer.Ships, mb.NewShipsMap()) || !reflect.DeepEqual(testJoinPlayer.Ships, mb.NewShipsMap()) {
	// 	t.Fatal("both players ships must have fresh set of ships after game reset")
	// }

	// if testHostPlayer.SunkenShips != 0 || testJoinPlayer.SunkenShips != 0 {
	// 	t.Fatal("both players must have 0 sunken ships after game reset")
	// }

	// if !reflect.DeepEqual(testHostPlayer.AttackGrid, mb.NewGrid(game.GridSize)) || !reflect.DeepEqual(testJoinPlayer.AttackGrid, mb.NewGrid(game.GridSize)) {
	// 	t.Fatal("both players must have fresh attack grids after game reset")
	// }

	// if !reflect.DeepEqual(testHostPlayer.DefenceGrid, mb.NewGrid(game.GridSize)) || !reflect.DeepEqual(testJoinPlayer.DefenceGrid, mb.NewGrid(game.GridSize)) {
	// 	t.Fatal("both players must have fresh defence grids after game reset")
	// }
}

func TestRematchRejection(t *testing.T) {
	// Host client sends a rematch call
	msg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCall)
	if err := HostConn.WriteJSON(msg); err != nil {
		t.Fatal(err)
	}

	// Join client receives this call and responds with no
	var rematchCall mc.Message[mc.NoPayload]
	if err := JoinConn.ReadJSON(&rematchCall); err != nil {
		t.Fatal(err)
	}

	msg = mc.NewMessage[mc.NoPayload](mc.CodeRematchCallRejected)
	if err := JoinConn.WriteJSON(msg); err != nil {
		t.Fatal(err)
	}

	// Host client reads this acceptance
	var rematchCallRejected mc.Message[mc.NoPayload]
	if err := HostConn.ReadJSON(&rematchCallRejected); err != nil {
		t.Fatal(err)
	}

	hostSession, err := testSessionManager.FindSession(HostSessionID)
	if err == nil {
		// This line will be done by IOS client
		testSessionManager.TerminateSession(hostSession)
	}

	_, err = testSessionManager.FindSession(HostSessionID)
	if err.Error() != cerr.ErrSessionNotFound(HostSessionID).Error() {
		t.Fatal("session for host player must not exist in session maps")
	}

	_, err = testSessionManager.FindSession(JoinSessionID)
	if err.Error() != cerr.ErrSessionNotFound(JoinSessionID).Error() {
		t.Fatal("session for join player must not exist in session maps")
	}
}
