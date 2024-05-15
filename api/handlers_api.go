package api

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	// cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

type RequestHandler interface {
	HandleCreateGame() (*md.Message[md.RespCreateGame], error)
	HandleReadyPlayer() (*md.Message[md.RespReadyPlayer], *md.Game, error)
	HandleJoinPlayer() (*md.Message[md.RespJoinGame], *md.Game, error)
	// HandleAttack() (*md.Game, error)
}

// Every incoming valid request will have this structure
// The request then is handled in line with WsRequestHandler interface
type Request struct {
	Server  *Server
	Ws      *websocket.Conn
	Payload []byte
}

// This tells the compiler that WsRequest struct must be of type of WsRequestHandler
var _ RequestHandler = (*Request)(nil)

func NewWsRequest(server *Server, ws *websocket.Conn, payload ...[]byte) *Request {
	if len(payload) > 1 {
		log.Println("cannot accept more than one payload")
		return nil
	}

	wsReq := Request{
		Server: server,
		Ws:     ws,
	}
	if len(payload) != 0 {
		wsReq.Payload = payload[0]
	}
	return &wsReq
}

func (w *Request) HandleCreateGame() (*md.Message[md.RespCreateGame], error) {
	game := w.Server.AddGame()
	hostPlayer := w.Server.AddHostPlayer(game, w.Ws)

	resp := md.NewMessage[md.RespCreateGame](md.CodeCreateGame)
	resp.AddPayload(md.RespCreateGame{GameUuid: game.Uuid, HostUuid: hostPlayer.Uuid})
	return &resp, nil
}

// User will choose the configurations of ships on defence grid.
// Then the grid is sent to backend and adjustment happens accordingly.
func (w *Request) HandleReadyPlayer() (*md.Message[md.RespReadyPlayer], *md.Game, error) {
	var readyPlayerReq md.Message[md.ReqReadyPlayer]
	if err := json.Unmarshal(w.Payload, &readyPlayerReq); err != nil {
		return nil, nil, err
	}
	log.Printf("unmarshaled ready player payload: %+v\n", readyPlayerReq)

	player, err := w.Server.FindPlayer(readyPlayerReq.Payload.PlayerUuid)
	if err != nil {
		return nil, nil, err
	}
	game, err := w.Server.FindGame(readyPlayerReq.Payload.GameUuid)
	if err != nil {
		return nil, nil, err
	}

	player.SetReady(readyPlayerReq.Payload.DefenceGrid)

	resp := md.NewMessage[md.RespReadyPlayer](md.CodeReady)
	return &resp, game, nil
}

// Join user sends the game uuid and if this game exists,
// a new join player is created and added to the database
func (w *Request) HandleJoinPlayer() (*md.Message[md.RespJoinGame], *md.Game, error) {
	var joinGameReq md.Message[md.ReqJoinGame]
	if err := json.Unmarshal(w.Payload, &joinGameReq); err != nil {
		return nil, nil, err
	}
	log.Printf("unmarshaled join game payload: %+v\n", joinGameReq)

	gameUuid := joinGameReq.Payload.GameUuid

	game, err := w.Server.AddJoinPlayer(gameUuid, w.Ws)
	if err != nil {
		return nil, nil, err
	}

	resp := md.NewMessage[md.RespJoinGame](md.CodeJoinGame)
	resp.AddPayload(md.RespJoinGame{PlayerUuid: game.JoinPlayer.Uuid})
	return &resp, game, nil
}

// func (w *Request) HandleAttack() (*md.Game, error) {
// 	var reqAttack md.Message[md.ReqAttack]
// 	if err := json.Unmarshal(w.Payload, &reqAttack); err != nil {
// 		return nil, err
// 	}

// 	initMap, err := TypeAssertPayloadToMap(reqAttack.Payload)
// 	if err != nil {
// 		return nil, err
// 	}

// 	attackInfo, err := TypeAssertIntPayload(initMap, md.KeyX, md.KeyY, md.KeyPositionState)
// 	if err != nil {
// 		return nil, err
// 	}
// 	x, y, positionState := attackInfo[0], attackInfo[1], attackInfo[2]

// 	if x >= md.GameGridSize || y >= md.GameGridSize {
// 		return nil, cerr.ErrXorYOutOfGridBound(x, y)
// 	}

// 	game, player, err := ExtractFindGamePlayer(w.Server, initMap)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// TODO: Assumption is that the frontend decides if the attack is hit or miss
// 	if player.IsHost {
// 		game.HostPlayer.AttackGrid[x][y] = positionState
// 		game.HostPlayer.IsTurn = false
// 		game.JoinPlayer.IsTurn = true
// 	} else {
// 		game.JoinPlayer.AttackGrid[x][y] = positionState
// 		game.JoinPlayer.IsTurn = false
// 		game.HostPlayer.IsTurn = true
// 	}

// 	return game, nil
// }

// func EndGame(s *Server, ws *websocket.Conn, payload []byte) error {
// 	return nil
// }
