package test

import (
	"testing"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

type Test[T, K any] struct {
	name         string
	expectedCode int
	expectedErr  string
	reqPayload   T
	respPayload  K
	conn         *websocket.Conn
}

func TestInvalidCode(t *testing.T) {
	tests := []Test[md.Message[md.NoPayload], md.Message[md.NoPayload]]{
		{
			name:         "random invalid code 1",
			expectedCode: md.CodeInvalidSignal,
			reqPayload:   md.NewMessage[md.NoPayload](-1),
			respPayload:  md.NewMessage[md.NoPayload](md.CodeInvalidSignal),
			conn:         HostConn,
		},
		{
			name:         "random invalid code 2",
			expectedCode: md.CodeInvalidSignal,
			reqPayload:   md.NewMessage[md.NoPayload](999),
			respPayload:  md.NewMessage[md.NoPayload](md.CodeInvalidSignal),
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
	tests := []Test[md.Message[md.NoPayload], md.Message[md.RespCreateGame]]{
		{
			name:         "create game valid code",
			expectedCode: md.CodeCreateGame,
			expectedErr:  "",
			reqPayload:   md.NewMessage[md.NoPayload](md.CodeCreateGame),
			respPayload:  md.NewMessage[md.RespCreateGame](md.CodeCreateGame),
			conn:         HostConn,
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

			if test.respPayload.Error.ErrorDetails != test.expectedErr {
				t.Fatalf("expected error: %s\t got: %s", test.reqPayload.Error.ErrorDetails, test.expectedErr)
			}

			if test.respPayload.Error.ErrorDetails == "" {
				GameUuid = test.respPayload.Payload.GameUuid
				HostPlayerId = test.respPayload.Payload.HostUuid
			}
		})
	}
}

func TestJoinPlayer(t *testing.T) {
	tests := []Test[md.Message[md.ReqJoinGame], md.Message[md.RespJoinGame]]{
		{
			name:         "valid game uuid",
			expectedCode: md.CodeJoinGame,
			expectedErr:  "",
			reqPayload:   md.Message[md.ReqJoinGame]{Code: md.CodeJoinGame, Payload: md.ReqJoinGame{GameUuid: GameUuid}},
			respPayload:  md.NewMessage[md.RespJoinGame](md.CodeJoinGame),
			conn:         JoinConn,
		},
		{
			name:         "invalid game uuid",
			expectedCode: md.CodeJoinGame,
			expectedErr:  cerr.ErrGameNotExists("-1invalid").Error(),
			reqPayload:   md.Message[md.ReqJoinGame]{Code: md.CodeJoinGame, Payload: md.ReqJoinGame{GameUuid: "-1invalid"}},
			respPayload:  md.NewMessage[md.RespJoinGame](md.CodeJoinGame),
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

			if test.respPayload.Error.ErrorDetails != test.expectedErr {
				t.Fatalf("expected error: %s\t got: %s", test.reqPayload.Error.ErrorDetails, test.expectedErr)
			}

			if test.respPayload.Error.ErrorDetails == "" {
				if GameUuid != test.respPayload.Payload.GameUuid {
					t.Fatal("incoming game uuid did not match the req uuid after join")
				}
				// if it was successful, join player id is set to the response
				JoinPlayerId = test.respPayload.Payload.PlayerUuid

				// Read extra message of success to host
				// we have to read it so it frees up the queue for the next steps of host read
				// when join player is added, a select grid code is sent to both players
				var respSelectGridHost md.Message[md.NoPayload]
				if err := HostConn.ReadJSON(&respSelectGridHost); err != nil {
					t.Fatal(err)
				}
				if respSelectGridHost.Error.ErrorDetails != "" {
					t.Fatalf("failed to receive select ready message for host: %s", respSelectGridHost.Error.ErrorDetails)
				}

				var respSelectGridJoin md.Message[md.NoPayload]
				if err := JoinConn.ReadJSON(&respSelectGridJoin); err != nil {
					t.Fatal(err)
				}
				if respSelectGridJoin.Error.ErrorDetails != "" {
					t.Fatalf("failed to receive select ready message for join: %s", respSelectGridJoin.Error.ErrorDetails)
				}
			}
		})
	}
}

func TestReadyGame(t *testing.T) {
	defenceGridHost := md.GridInt{
		{0, md.PositionStateDefenceDestroyer, md.PositionStateDefenceDestroyer, 0, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{md.PositionStateDefenceCruiser, 0, 0, md.PositionStateDefenceBattleship, 0},
		{0, 0, 0, md.PositionStateDefenceBattleship, 0},
	}

	defenceGridJoin := md.GridInt{
		{0, md.PositionStateDefenceDestroyer, md.PositionStateDefenceDestroyer, 0, 0},
		{md.PositionStateDefenceCruiser, 0, 0, 0, md.PositionStateDefenceBattleship},
		{md.PositionStateDefenceCruiser, 0, 0, 0, md.PositionStateDefenceBattleship},
		{md.PositionStateDefenceCruiser, 0, 0, 0, md.PositionStateDefenceBattleship},
		{0, 0, 0, 0, md.PositionStateDefenceBattleship},
	}

	tests := []Test[md.Message[md.ReqReadyPlayer], md.Message[md.NoPayload]]{
		{
			name:         "set defence grid ready host",
			expectedCode: md.CodeReady,
			expectedErr:  "",
			reqPayload: md.Message[md.ReqReadyPlayer]{
				Code: md.CodeReady,
				Payload: md.ReqReadyPlayer{
					DefenceGrid: defenceGridHost,
					GameUuid:    GameUuid,
					PlayerUuid:  HostPlayerId,
				},
			},
			respPayload: md.Message[md.NoPayload]{},
			conn:        HostConn,
		},
		{
			name:         "set defence grid ready join",
			expectedCode: md.CodeReady,
			expectedErr:  "",
			reqPayload: md.Message[md.ReqReadyPlayer]{
				Code: md.CodeReady,
				Payload: md.ReqReadyPlayer{
					DefenceGrid: defenceGridJoin,
					GameUuid:    GameUuid,
					PlayerUuid:  JoinPlayerId,
				},
			},
			respPayload: md.Message[md.NoPayload]{},
			conn:        JoinConn,
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

			if test.respPayload.Error.ErrorDetails != test.expectedErr {
				t.Fatalf("expected error: %s\t got: %s", test.reqPayload.Error.ErrorDetails, test.expectedErr)
			}

			// After the success of second test
			// start game code will be sent to both parties
			if test.name == "set defence grid ready join" {
				// Reading game ready codes

				// Host
				var respStartGameHost md.Message[md.NoPayload]
				if err := HostConn.ReadJSON(&respStartGameHost); err != nil {
					t.Fatal(err)
				}

				// Join
				var respStartGameJoin md.Message[md.NoPayload]
				if err := JoinConn.ReadJSON(&respStartGameJoin); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
