package api

import (
	"encoding/json"
	"log"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

type RequestHandler interface {
	HandleCreateGame() *mc.Message[mc.RespCreateGame]
	HandleReadyPlayer() (mc.Message[mc.NoPayload], *mb.Game)
	HandleJoinPlayer() (*mc.Message[mc.RespJoinGame], *mb.Game)
	HandleAttack() (*mc.Message[mc.RespAttack], *mb.Player)
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

func (r *Request) HandleCreateGame() *mc.Message[mc.RespCreateGame] {
	var reqCreateGame mc.Message[mc.ReqCreateGame]
	resp := mc.NewMessage[mc.RespCreateGame](mc.CodeCreateGame)

	if err := json.Unmarshal(r.Payload, &reqCreateGame); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return &resp
	}

	gameDifficulty := reqCreateGame.Payload.GameDifficulty
	if gameDifficulty != mb.GameDifficultyEasy && gameDifficulty != mb.GameDifficultyHard {
		resp.AddError(cerr.ErrInvalidGameDifficulty().Error(), "")
		return &resp
	}

	game := r.Session.GameManager.AddGame(gameDifficulty)
	r.Session.GameUuid = game.Uuid

	hostPlayer := game.CreateHostPlayer(r.Session.ID)
	r.Session.Player = hostPlayer

	resp.AddPayload(mc.RespCreateGame{GameUuid: game.Uuid, HostUuid: hostPlayer.Uuid})
	return &resp
}

// User will choose the configurations of ships on defence grid.
// Then the grid is sent to backend and adjustment happens accordingly.
func (r *Request) HandleReadyPlayer() (mc.Message[mc.NoPayload], *mb.Game) {
	var readyPlayerReq mc.Message[mc.ReqReadyPlayer]
	resp := mc.NewMessage[mc.NoPayload](mc.CodeReady)

	if err := json.Unmarshal(r.Payload, &readyPlayerReq); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp, nil
	}

	game, player, err := r.Session.GameManager.FindGameAndPlayer(readyPlayerReq.Payload.GameUuid, readyPlayerReq.Payload.PlayerUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrReady)
		return resp, nil
	}

	// Check to see if rows and cols are equal to game's grid size
	rows := len(readyPlayerReq.Payload.DefenceGrid)
	if rows != game.GridSize {
		resp.AddError(cerr.ErrDefenceGridRowsOutOfBounds(rows, game.GridSize).Error(), cerr.ConstErrReady)
		return resp, nil
	}
	cols := len(readyPlayerReq.Payload.DefenceGrid[0])
	if cols != game.GridSize {
		resp.AddError(cerr.ErrDefenceGridColsOutOfBounds(cols, game.GridSize).Error(), cerr.ConstErrReady)
		return resp, nil
	}

	player.SetDefenceGrid(readyPlayerReq.Payload.DefenceGrid)
	player.IsReady = true
	return resp, game
}

// Join user sends the game uuid and if this game exists,
// a new join player is created and added to the database
func (r *Request) HandleJoinPlayer() (*mc.Message[mc.RespJoinGame], *mb.Game) {
	var joinGameReq mc.Message[mc.ReqJoinGame]
	resp := mc.NewMessage[mc.RespJoinGame](mc.CodeJoinGame)

	if err := json.Unmarshal(r.Payload, &joinGameReq); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return &resp, nil
	}

	game, err := r.Session.GameManager.FindGame(joinGameReq.Payload.GameUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrJoin)
		return &resp, nil
	}

	joinPlayer := game.CreateJoinPlayer(r.Session.ID)

	r.Session.GameUuid = game.Uuid
	r.Session.Player = joinPlayer

	resp.AddPayload(mc.RespJoinGame{GameUuid: game.Uuid, PlayerUuid: joinPlayer.Uuid})
	return &resp, game
}

// Handle the attack logic for the incoming request
func (r *Request) HandleAttack() (*mc.Message[mc.RespAttack], *mb.Player) {
	var reqAttack mc.Message[mc.ReqAttack]
	resp := mc.NewMessage[mc.RespAttack](mc.CodeAttack)

	if err := json.Unmarshal(r.Payload, &reqAttack); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return &resp, nil
	}
	
	game, attacker, err := r.Session.GameManager.FindGameAndPlayer(reqAttack.Payload.GameUuid, reqAttack.Payload.PlayerUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	x := reqAttack.Payload.X
	y := reqAttack.Payload.Y
	if x > game.ValidUpperBound || y > game.ValidUpperBound || x < game.ValidLowerBound || y < game.ValidLowerBound {
		resp.AddError(cerr.ErrXorYOutOfGridBound(x, y).Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	// If attacker has the correct IsTurn Field
	if !attacker.IsTurn {
		resp.AddError(cerr.ErrNotTurnForAttacker(attacker.Uuid).Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	// Check if the attack position was already hit before (invalid position to attack)
	if attacker.AttackGrid[x][y] != mb.PositionStateAttackGridEmpty {
		resp.AddError(cerr.ErrAttackPositionAlreadyFilled(x, y).Error(), cerr.ConstErrAttack)
		return &resp, nil
	}

	// Idenitify the defender
	defender := game.HostPlayer
	if attacker.IsHost {
		defender = game.JoinPlayer
	}

	// Check what is in the position of attack in defence grid matrix of defender
	positionCode, err := defender.IdentifyHitCoordsEssence(x, y)
	if err != nil {
		// Invalid position in defender defence grid (already hit)
		resp.AddError(err.Error(), cerr.ConstErrAttack)
		return &resp, defender
	}

	// Change the status of players turn
	attacker.IsTurn = false
	defender.IsTurn = true

	// If the attacker missed
	if positionCode == mb.PositionStateAttackGridMiss {
		// adjust attack grid for attacker
		attacker.AttackGrid[x][y] = mb.PositionStateAttackGridMiss

		resp.AddPayload(mc.RespAttack{
			X:               x,
			Y:               y,
			PositionState:   mb.PositionStateAttackGridMiss,
			SunkenShipsHost: game.HostPlayer.SunkenShips,
			SunkenShipsJoin: game.JoinPlayer.SunkenShips,
		})
		return &resp, defender
	}

	// ! Passed this line, positionCode is a ship code used to extract ship from map
	// Apply the attack to the position for both defender and attacker
	defender.HitShip(positionCode, x, y)
	attacker.AttackGrid[x][y] = mb.PositionStateAttackGridHit

	// Initialize the response payload
	resp.AddPayload(mc.RespAttack{
		X:             x,
		Y:             y,
		PositionState: mb.PositionStateAttackGridHit,
	})

	// Check if the attack caused the ship to sink
	if defender.IsShipSunken(positionCode) {
		resp.Payload.DefenderSunkenShipsCoords = defender.Ships[positionCode].GetHitCoordinates()

		// Check if this sunken ship was the last one and the attacker is lost
		if defender.IsLoser() {
			defender.MatchStatus = mb.PlayerMatchStatusLost
			attacker.MatchStatus = mb.PlayerMatchStatusWon
			game.FinishGame()
		}
	}

	resp.Payload.SunkenShipsHost = game.HostPlayer.SunkenShips
	resp.Payload.SunkenShipsJoin = game.JoinPlayer.SunkenShips
	return &resp, defender
}
