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
	HandleAttack() (*md.Message[md.RespAttack], *md.Player)
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

func NewRequest(server *Server, ws *websocket.Conn, payload ...[]byte) *Request {
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

	player.SetDefenceGrid(readyPlayerReq.Payload.DefenceGrid)

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

func (w *Request) HandleAttack() (*md.Message[md.RespAttack], *md.Player) {
	var reqAttack md.Message[md.ReqAttack]
	resp := md.NewMessage[md.RespAttack](md.CodeAttack)

	if err := json.Unmarshal(w.Payload, &reqAttack); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrAttackFailed)
		return &resp, nil
	}

	x := reqAttack.Payload.X
	y := reqAttack.Payload.Y
	if x > md.GameValidBound || y > md.GameValidBound {
		resp.AddError(cerr.ErrXorYOutOfGridBound(x, y).Error(), cerr.ConstErrAttackFailed)
		return &resp, nil
	}

	game, err := w.Server.FindGame(reqAttack.Payload.GameUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrAttackFailed)
		return &resp, nil
	}
	player, err := w.Server.FindPlayer(reqAttack.Payload.PlayerUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrAttackFailed)
		return &resp, nil
	}

	if player.AttackGrid[x][y] != md.PositionStateAttackGridEmpty {
		resp.AddError(cerr.ErrAttackPositionAlreadyFilled(x, y).Error(), cerr.ConstErrAttackFailed)
		return &resp, nil
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

	// Check what is in the position of attack in defence grid matrix of defender
	positionCode, err := defender.FetchDefenceGridPositionCode(x, y)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrAttackFailed)
		return &resp, defender
	}

	// If the attacker missed
	if positionCode == md.PositionStateDefenceGridEmpty {
		resp.AddPayload(md.RespAttack{
			PositionState: md.PositionStateAttackGridMiss,
			IsTurn:        attacker.IsTurn,
		})
		return &resp, defender
	}

	// Apply the attack to the position
	defender.HitShip(positionCode, x, y)

	// Check if the attack cause the ship to sink
	if defender.IsShipSunken(positionCode) {

		// Check if this sunken ship was the last one and the player is lost
		if defender.IsLoser() {
			defender.MatchStatus = md.PlayerMatchStatusLost
			attacker.MatchStatus = md.PlayerMatchStatusWon

			resp.AddPayload(md.RespAttack{
				PositionState: md.PositionStateAttackGridHit,
				IsTurn:        attacker.IsTurn,
			})
			game.FinishGame()
			return &resp, defender
		}
	}

	resp.AddPayload(md.RespAttack{
		IsTurn:        attacker.IsTurn,
		PositionState: md.PositionStateAttackGridHit,
	})

	return &resp, defender
}
