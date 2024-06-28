package api

import (
	"encoding/json"
	"log"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

type RequestHandler interface {
	HandleCreateGame() mc.Message[mc.RespCreateGame]
	HandleReadyPlayer() (mc.Message[mc.NoPayload], *mb.Game)
	HandleJoinPlayer() (mc.Message[mc.RespJoinGame], *mb.Game)
	HandleAttack() (mc.Message[mc.RespAttack], *mb.Player)
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

func (r *Request) HandleCreateGame() mc.Message[mc.RespCreateGame] {
	var reqCreateGame mc.Message[mc.ReqCreateGame]
	resp := mc.NewMessage[mc.RespCreateGame](mc.CodeCreateGame)

	if err := json.Unmarshal(r.Payload, &reqCreateGame); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp
	}

	gameDifficulty := reqCreateGame.Payload.GameDifficulty
	if !r.Session.GameManager.isDifficultyValid(gameDifficulty) {
		resp.AddError(cerr.ErrInvalidGameDifficulty().Error(), "")
		return resp
	}

	game := r.Session.GameManager.AddGame(gameDifficulty)
	r.Session.GameUuid = game.Uuid

	hostPlayer := game.CreateHostPlayer(r.Session.ID)
	r.Session.Player = hostPlayer

	resp.AddPayload(mc.RespCreateGame{GameUuid: game.Uuid, HostUuid: hostPlayer.Uuid})
	return resp
}

// Join user sends the game uuid and if this game exists,
// a new join player is created and added to the database
func (r *Request) HandleJoinPlayer() (mc.Message[mc.RespJoinGame], *mb.Game) {
	var joinGameReq mc.Message[mc.ReqJoinGame]
	resp := mc.NewMessage[mc.RespJoinGame](mc.CodeJoinGame)

	if err := json.Unmarshal(r.Payload, &joinGameReq); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp, nil
	}

	game, err := r.Session.GameManager.FindGame(joinGameReq.Payload.GameUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrJoin)
		return resp, nil
	}

	joinPlayer := game.CreateJoinPlayer(r.Session.ID)

	r.Session.GameUuid = game.Uuid
	r.Session.Player = joinPlayer

	resp.AddPayload(mc.RespJoinGame{GameUuid: game.Uuid, PlayerUuid: joinPlayer.Uuid})
	return resp, game
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

	game, err := r.Session.GameManager.FindGame(readyPlayerReq.Payload.GameUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrReady)
		return resp, nil
	}

	player, err := game.FindPlayer(readyPlayerReq.Payload.PlayerUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrReady)
		return resp, nil
	}

	// Check to see if rows and cols are equal to game's grid size
	gameGridSize := game.GridSize
	rows := len(readyPlayerReq.Payload.DefenceGrid)
	if rows != gameGridSize {
		resp.AddError(cerr.ErrDefenceGridRowsOutOfBounds(rows, game.GridSize).Error(), cerr.ConstErrReady)
		return resp, nil
	}
	cols := len(readyPlayerReq.Payload.DefenceGrid[0])
	if cols != gameGridSize {
		resp.AddError(cerr.ErrDefenceGridColsOutOfBounds(cols, game.GridSize).Error(), cerr.ConstErrReady)
		return resp, nil
	}

	player.SetReady(readyPlayerReq.Payload.DefenceGrid)
	return resp, game
}

// Handle the attack logic for the incoming request
func (r *Request) HandleAttack() (mc.Message[mc.RespAttack], *mb.Player) {
	var reqAttack mc.Message[mc.ReqAttack]
	resp := mc.NewMessage[mc.RespAttack](mc.CodeAttack)

	if err := json.Unmarshal(r.Payload, &reqAttack); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp, nil
	}

	game, attacker, err := r.Session.GameManager.FindGameAndPlayer(reqAttack.Payload.GameUuid, reqAttack.Payload.PlayerUuid)
	if err != nil {
		resp.AddError(err.Error(), cerr.ConstErrAttack)
		return resp, nil
	}

	coordinates := mb.NewCoordinates(reqAttack.Payload.X, reqAttack.Payload.Y)
	if game.AreIncomingCoordinatesInvalid(coordinates) {
		resp.AddError(cerr.ErrXorYOutOfGridBound(coordinates.X, coordinates.Y).Error(), cerr.ConstErrAttack)
		return resp, nil
	}

	// Attack validity check
	if !attacker.IsTurn {
		resp.AddError(cerr.ErrNotTurnForAttacker(attacker.Uuid).Error(), cerr.ConstErrAttack)
		return resp, nil
	}
	if attacker.DidAttackThisCoordinatesBefore(coordinates) {
		resp.AddError(cerr.ErrAttackPositionAlreadyFilled(coordinates.X, coordinates.Y).Error(), cerr.ConstErrAttack)
		return resp, nil
	}

	// Idenitify the defender
	defender := game.HostPlayer
	if attacker.IsHost {
		defender = game.JoinPlayer
	}

	if defender.AreCoordinatesAlreadyHit(coordinates) {
		resp.AddError(cerr.ErrDefenceGridPositionAlreadyHit(coordinates.X, coordinates.Y).Error(), cerr.ConstErrAttack)
		return resp, defender
	}

	attacker.IsTurn = false
	defender.IsTurn = true

	if defender.IsIncomingAttackMiss(coordinates) {
		attacker.AttackGrid[coordinates.X][coordinates.Y] = mb.PositionStateAttackGridMiss
		resp.AddPayload(mc.RespAttack{
			X:               coordinates.X,
			Y:               coordinates.Y,
			PositionState:   mb.PositionStateAttackGridMiss,
			SunkenShipsHost: game.HostPlayer.SunkenShips,
			SunkenShipsJoin: game.JoinPlayer.SunkenShips,
		})
		return resp, defender
	}

	shipCode := defender.DefenceGrid[coordinates.X][coordinates.Y]

	defender.HitShip(shipCode, coordinates)
	attacker.AttackGrid[coordinates.X][coordinates.Y] = mb.PositionStateAttackGridHit

	// Initialize the response payload
	resp.AddPayload(mc.RespAttack{
		X:             coordinates.X,
		Y:             coordinates.Y,
		PositionState: mb.PositionStateAttackGridHit,
	})

	// Check if the attack caused the ship to sink
	if defender.IsShipSunken(shipCode) {
		resp.Payload.DefenderSunkenShipsCoords = defender.Ships[shipCode].GetHitCoordinates()

		// Check if this sunken ship was the last one and the attacker is lost
		if defender.IsLoser() {
			defender.MatchStatus = mb.PlayerMatchStatusLost
			attacker.MatchStatus = mb.PlayerMatchStatusWon
			game.FinishGame()
		}
	}

	resp.Payload.SunkenShipsHost = game.HostPlayer.SunkenShips
	resp.Payload.SunkenShipsJoin = game.JoinPlayer.SunkenShips
	return resp, defender
}
