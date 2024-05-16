package api

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

type RequestHandler interface {
	HandleCreateGame() (*md.Message[md.RespCreateGame], error)
	HandleReadyPlayer() (*md.Message[md.RespReadyPlayer], *md.Game, error)
	HandleJoinPlayer() (*md.Message[md.RespJoinGame], *md.Game, error)
	HandleAttack() (*md.Message[md.RespAttack], *md.Game, error)
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

func (w *Request) HandleAttack() (*md.Message[md.RespAttack], *md.Game, error) {
	var reqAttack md.Message[md.ReqAttack]
	if err := json.Unmarshal(w.Payload, &reqAttack); err != nil {
		return nil, nil, err
	}

	x := reqAttack.Payload.X
	y := reqAttack.Payload.Y
	if x > md.GameValidBound || y > md.GameValidBound {
		return nil, nil, cerr.ErrXorYOutOfGridBound(x, y)
	}

	game, err := w.Server.FindGame(reqAttack.Payload.GameUuid)
	if err != nil {
		return nil, nil, err
	}
	player, err := w.Server.FindPlayer(reqAttack.Payload.PlayerUuid)
	if err != nil {
		return nil, nil, err
	}

	if player.AttackGrid[x][y] != md.PositionStateAttackNeutral {
		return nil, nil, cerr.ErrAttackPositionAlreadyFilled(x, y)
	}

	defender := game.HostPlayer
	attacker := game.JoinPlayer
	if player.IsHost {
		defender = game.JoinPlayer
		attacker = game.HostPlayer
	}

	// Change the status of players turn
	attacker.IsTurn = false
	defender.IsTurn = true

	var resultPositionState int
	if defender.DefenceGrid[x][y] == md.PositionStateDefenceGridShip {
		resultPositionState = md.PositionStateAttackHit
		attacker.AttackGrid[x][y] = resultPositionState
	} else {
		resultPositionState = md.PositionStateAttackMiss
		attacker.AttackGrid[x][y] = resultPositionState
	}

	resp := md.NewMessage[md.RespAttack](md.CodeAttack)
	resp.AddPayload(md.RespAttack{
		IsTurn:        false,
		PositionState: resultPositionState,
	})

	return &resp, game, nil
}
