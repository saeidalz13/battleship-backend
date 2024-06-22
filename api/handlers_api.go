package api

import (
	"encoding/json"
	"log"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

type RequestHandler interface {
	HandleCreateGame() *md.Message[md.RespCreateGame]
	HandleReadyPlayer() (*md.Message[md.NoPayload], *md.Game)
	HandleJoinPlayer() (*md.Message[md.RespJoinGame], *md.Game)
	HandleAttack() (*md.Message[md.RespAttack], *md.Player)
}

// Every incoming valid request will have this structure
// The request then is handled in line with WsRequestHandler interface
type Request struct {
	Payload []byte
	Session *Session
}

// This tells the compiler that WsRequest struct must be of type of WsRequestHandler
var _ RequestHandler = (*Request)(nil)

func NewRequest(session *Session, payload ...[]byte) *Request {
	if len(payload) > 1 {
		log.Println("cannot accept more than one payload")
		return nil
	}

	wsReq := Request{
		Session: session,
	}
	if len(payload) != 0 {
		wsReq.Payload = payload[0]
	}
	return &wsReq
}

func (r *Request) HandleCreateGame() *md.Message[md.RespCreateGame] {
	game := r.Session.GameManager.AddGame()
	r.Session.GameUuid = game.Uuid

	hostPlayer := game.CreateHostPlayer(r.Session.Conn, r.Session.ID)
	r.Session.Player = hostPlayer

	resp := md.NewMessage[md.RespCreateGame](md.CodeCreateGame)
	resp.AddPayload(md.RespCreateGame{GameUuid: game.Uuid, HostUuid: hostPlayer.Uuid})
	return &resp
}

// User will choose the configurations of ships on defence grid.
// Then the grid is sent to backend and adjustment happens accordingly.
func (r *Request) HandleReadyPlayer() (*md.Message[md.NoPayload], *md.Game) {
	var readyPlayerReq md.Message[md.ReqReadyPlayer]
	resp := md.NewMessage[md.NoPayload](md.CodeReady)

	if err := json.Unmarshal(r.Payload, &readyPlayerReq); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return &resp, nil
	}

	game, player, err := r.Session.GameManager.FindGameAndPlayer(readyPlayerReq.Payload.GameUuid, readyPlayerReq.Payload.PlayerUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrReady)
		return &resp, nil
	}

	// Check to see if rows and cols are equal to game's grid size
	rows := len(readyPlayerReq.Payload.DefenceGrid)
	if rows != md.GameGridSize {
		resp.AddError(cerr.ErrDefenceGridRowsOutOfBounds(rows, md.GameGridSize).Error(), cerr.ConstErrReady)
		return &resp, nil
	}
	cols := len(readyPlayerReq.Payload.DefenceGrid[0])
	if cols != md.GameGridSize {
		resp.AddError(cerr.ErrDefenceGridColsOutOfBounds(cols, md.GameGridSize).Error(), cerr.ConstErrReady)
		return &resp, nil
	}

	player.SetDefenceGrid(readyPlayerReq.Payload.DefenceGrid)
	player.IsReady = true
	return &resp, game
}

// Join user sends the game uuid and if this game exists,
// a new join player is created and added to the database
func (r *Request) HandleJoinPlayer() (*md.Message[md.RespJoinGame], *md.Game) {
	var joinGameReq md.Message[md.ReqJoinGame]
	resp := md.NewMessage[md.RespJoinGame](md.CodeJoinGame)

	if err := json.Unmarshal(r.Payload, &joinGameReq); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return &resp, nil
	}

	game, err := r.Session.GameManager.FindGame(joinGameReq.Payload.GameUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrJoin)
		return &resp, nil
	}

	joinPlayer := game.CreateJoinPlayer(r.Session.Conn, r.Session.ID)

	r.Session.GameUuid = game.Uuid
	r.Session.Player = joinPlayer

	resp.AddPayload(md.RespJoinGame{GameUuid: game.Uuid, PlayerUuid: joinPlayer.Uuid})
	return &resp, game
}

// Handle the attack logic for the incoming request
func (r *Request) HandleAttack() (*md.Message[md.RespAttack], *md.Player) {
	var reqAttack md.Message[md.ReqAttack]
	resp := md.NewMessage[md.RespAttack](md.CodeAttack)

	if err := json.Unmarshal(r.Payload, &reqAttack); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return &resp, nil
	}

	x := reqAttack.Payload.X
	y := reqAttack.Payload.Y
	if x > md.GameValidUpperBound || y > md.GameValidUpperBound || x < md.GameValidLowerBound || y < md.GameValidLowerBound {
		resp.AddError(cerr.ErrXorYOutOfGridBound(x, y).Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	game, attacker, err := r.Session.GameManager.FindGameAndPlayer(reqAttack.Payload.GameUuid, reqAttack.Payload.PlayerUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	// If attacker has the correct IsTurn Field
	if !attacker.IsTurn {
		resp.AddError(cerr.ErrNotTurnForAttacker(attacker.Uuid).Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	// Check if the attack position was already hit before (invalid position to attack)
	if attacker.AttackGrid[x][y] != md.PositionStateAttackGridEmpty {
		resp.AddError(cerr.ErrAttackPositionAlreadyFilled(x, y).Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	// Idenitify the defender
	defender := game.HostPlayer
	if attacker.IsHost {
		defender = game.JoinPlayer
	}

	// Check what is in the position of attack in defence grid matrix of defender
	positionCode, err := defender.FetchDefenceGridPositionCode(x, y)
	if err != nil {
		// Invalid position in defender defence grid (already hit)
		resp.AddError(err.Error(), cerr.ConstErrAttack)
		return &resp, defender
	}

	// Change the status of players turn
	attacker.IsTurn = false
	defender.IsTurn = true

	// If the attacker missed
	if positionCode == md.PositionStateAttackGridMiss {
		// adjust attack grid for attacker
		attacker.AttackGrid[x][y] = md.PositionStateAttackGridMiss

		resp.AddPayload(md.RespAttack{
			X:               x,
			Y:               y,
			PositionState:   md.PositionStateAttackGridMiss,
			SunkenShipsHost: game.HostPlayer.SunkenShips,
			SunkenShipsJoin: game.JoinPlayer.SunkenShips,
		})
		return &resp, defender
	}

	// Apply the attack to the position for both defender and attacker
	defender.HitShip(positionCode, x, y)
	attacker.AttackGrid[x][y] = md.PositionStateAttackGridHit

	// Initialize the payload
	resp.AddPayload(md.RespAttack{
		X:             x,
		Y:             y,
		PositionState: md.PositionStateAttackGridHit,
	})

	// Check if the attack cause the ship to sink
	if defender.IsShipSunken(positionCode) {
		// Check if this sunken ship was the last one and the attacker is lost
		if defender.IsLoser() {
			defender.MatchStatus = md.PlayerMatchStatusLost
			attacker.MatchStatus = md.PlayerMatchStatusWon
			game.FinishGame()
		}
	}

	resp.Payload.SunkenShipsHost = game.HostPlayer.SunkenShips
	resp.Payload.SunkenShipsJoin = game.JoinPlayer.SunkenShips
	return &resp, defender
}
