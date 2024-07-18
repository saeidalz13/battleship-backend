package api

import (
	"encoding/json"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

type RequestHandler interface {
	HandleCreateGame(gm mb.GameManager, sessionId string) (*mb.Game, mb.Player, mc.Message[mc.RespCreateGame])
	HandleReadyPlayer(gm mb.GameManager, sessionGame *mb.Game, sessionPlayer mb.Player) mc.Message[mc.NoPayload]
	HandleJoinPlayer(gm mb.GameManager, sessionId string) (*mb.Game, mb.Player, mc.Message[mc.RespJoinGame])
	HandleAttack(*mb.Game, mb.Player, mb.Player, mb.GameManager) mc.Message[mc.RespAttack]
	HandleCallRematch(bgm mb.GameManager, sessionGame *mb.Game) (mc.Message[mc.NoPayload], error)
	HandleAcceptRematchCall(bgm mb.GameManager, sessionGame *mb.Game, sessionPlayer, otherSessionPlayer mb.Player) (mc.Message[mc.RespRematch], mc.Message[mc.RespRematch], error)
}

// Every incoming valid request will have this structure
// The request then is handled in line with WsRequestHandler interface
type Request struct {
	payload []byte
}

// This tells the compiler that WsRequest struct must be of type of WsRequestHandler
var _ RequestHandler = (*Request)(nil)

func NewRequest(payloads ...[]byte) Request {
	if len(payloads) > 1 {
		panic("request cannot accept more than one payload")
	}
	r := Request{}
	if len(payloads) == 1 {
		r.payload = payloads[0]
	}
	return r
}

func (r Request) HandleCreateGame(gm mb.GameManager, sessionId string) (*mb.Game, mb.Player, mc.Message[mc.RespCreateGame]) {
	var reqCreateGame mc.Message[mc.ReqCreateGame]
	respMsg := mc.NewMessage[mc.RespCreateGame](mc.CodeCreateGame)

	if err := json.Unmarshal(r.payload, &reqCreateGame); err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return nil, nil, respMsg
	}

	game, err := gm.CreateGame(reqCreateGame.Payload.GameDifficulty)
	if err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrCreateGame)
		return nil, nil, respMsg
	}

	hostPlayer := game.CreateHostPlayer(sessionId)

	respMsg.AddPayload(mc.RespCreateGame{GameUuid: game.Uuid(), HostUuid: hostPlayer.Uuid()})
	return game, hostPlayer, respMsg
}

// Join user sends the game uuid and if this game exists,
// a new join player is created and added to the database
func (r Request) HandleJoinPlayer(gm mb.GameManager, sessionId string) (*mb.Game, mb.Player, mc.Message[mc.RespJoinGame]) {
	var joinGameReq mc.Message[mc.ReqJoinGame]
	respMsg := mc.NewMessage[mc.RespJoinGame](mc.CodeJoinGame)

	if err := json.Unmarshal(r.payload, &joinGameReq); err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return nil, nil, respMsg
	}

	game, err := gm.FetchGame(joinGameReq.Payload.GameUuid)
	if err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrJoin)
		return nil, nil, respMsg
	}

	joinPlayer := game.CreateJoinPlayer(sessionId)

	respMsg.AddPayload(mc.RespJoinGame{GameUuid: game.Uuid(), PlayerUuid: joinPlayer.Uuid(), GameDifficulty: game.Difficulty()})
	return game, joinPlayer, respMsg
}

// User will choose the configurations of ships on defence grid.
// Then the grid is sent to backend and adjustment happens accordingly.
func (r Request) HandleReadyPlayer(bgm mb.GameManager, game *mb.Game, sessionPlayer mb.Player) mc.Message[mc.NoPayload] {
	var readyPlayerReq mc.Message[mc.ReqReadyPlayer]
	resp := mc.NewMessage[mc.NoPayload](mc.CodeReady)

	if err := json.Unmarshal(r.payload, &readyPlayerReq); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp
	}

	// Check to see if rows and cols are equal to game's grid size
	if err := game.SetPlayerReadyForGame(sessionPlayer, readyPlayerReq.Payload.DefenceGrid); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrReady)
		return resp
	}

	return resp
}

// Handle the attack logic for the incoming request
func (r Request) HandleAttack(game *mb.Game, attacker mb.Player, defender mb.Player, gm mb.GameManager) mc.Message[mc.RespAttack] {
	var reqAttack mc.Message[mc.ReqAttack]
	resp := mc.NewMessage[mc.RespAttack](mc.CodeAttack)

	if err := json.Unmarshal(r.payload, &reqAttack); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp
	}

	coordinates := mb.NewCoordinates(reqAttack.Payload.X, reqAttack.Payload.Y)
	if !game.AreAttackCoordinatesValid(coordinates) {
		resp.AddError(cerr.ErrXorYOutOfGridBound(coordinates.X, coordinates.Y).Error(), cerr.ConstErrAttack)
		return resp
	}

	// Attack validity check
	if !attacker.IsTurn() {
		resp.AddError(cerr.ErrNotTurnForAttacker(attacker.Uuid()).Error(), cerr.ConstErrAttack)
		return resp
	}

	if !attacker.IsAttackGridEmptyInCoordinates(coordinates) {
		resp.AddError(cerr.ErrAttackPositionAlreadyFilled(coordinates.X, coordinates.Y).Error(), cerr.ConstErrAttack)
		return resp
	}

	if defender.IsDefenceGridAlreadyHitInCoordinates(coordinates) {
		resp.AddError(cerr.ErrDefenceGridPositionAlreadyHit(coordinates.X, coordinates.Y).Error(), cerr.ConstErrAttack)
		return resp
	}

	attacker.SetTurnFalse()
	defender.SetTurnTrue()

	hostPlayer := game.HostPlayer()
	joinPlayer := game.JoinPlayer()

	// TODO: Check for game mode for this section
	if defender.DidAttackerHitMine(coordinates) {
		attacker.SetMatchStatusToLost()
		defender.SetMatchStatusToWon()

		resp.AddPayload(mc.RespAttack{
			X:               coordinates.X,
			Y:               coordinates.Y,
			PositionState:   mb.PositionStateMine,
			SunkenShipsHost: hostPlayer.SunkenShips(),
			SunkenShipsJoin: joinPlayer.SunkenShips(),
			IsTurn:          attacker.IsTurn(),
		})

		return resp
	}

	if defender.IsAttackMiss(coordinates) {
		attacker.SetAttackGridToMiss(coordinates)

		resp.AddPayload(mc.RespAttack{
			X:               coordinates.X,
			Y:               coordinates.Y,
			PositionState:   mb.PositionStateAttackGridMiss,
			SunkenShipsHost: hostPlayer.SunkenShips(),
			SunkenShipsJoin: joinPlayer.SunkenShips(),
			IsTurn:          attacker.IsTurn(),
		})
		return resp
	}

	shipCode := defender.ShipCode(coordinates)
	defender.IncrementShipHit(shipCode, coordinates)
	attacker.SetAttackGridToHit(coordinates)

	// Initialize the response payload
	resp.AddPayload(mc.RespAttack{
		X:             coordinates.X,
		Y:             coordinates.Y,
		PositionState: mb.PositionStateAttackGridHit,
		IsTurn:        attacker.IsTurn(),
	})

	// Check if the attack caused the ship to sink
	if defender.IsShipSunken(shipCode) {
		defender.IncrementSunkenShips()
		resp.Payload.DefenderSunkenShipsCoords = defender.ShipHitCoordinates(shipCode)

		// Check if this sunken ship was the last one and the attacker is lost
		if defender.AreAllShipsSunken() {
			defender.SetMatchStatusToLost()
			attacker.SetMatchStatusToWon()
		}
	}

	resp.Payload.SunkenShipsHost = hostPlayer.SunkenShips()
	resp.Payload.SunkenShipsJoin = joinPlayer.SunkenShips()
	return resp
}

func (r Request) HandleCallRematch(bgm mb.GameManager, game *mb.Game) (mc.Message[mc.NoPayload], error) {
	respMsg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCall)

	if game.IsRematchAlreadyCalled() {
		return respMsg, cerr.ErrGameAleardyRecalled()
	}

	game.CallRematchForGame()
	return respMsg, nil
}

func (r Request) HandleAcceptRematchCall(
	bgm mb.GameManager,
	game *mb.Game,
	sessionPlayer, otherSessionPlayer mb.Player,
) (mc.Message[mc.RespRematch], mc.Message[mc.RespRematch], error) {

	if err := game.ResetRematchForGame(); err != nil {
		return mc.NewMessage[mc.RespRematch](mc.CodeRematch), mc.NewMessage[mc.RespRematch](mc.CodeRematch), err
	}

	sessionPlayer.SetTurnFalse()
	msgPlayer := mc.NewMessage[mc.RespRematch](mc.CodeRematch)
	msgPlayer.AddPayload(mc.RespRematch{IsTurn: sessionPlayer.IsTurn()})

	otherSessionPlayer.SetTurnTrue()
	msgOtherPlayer := mc.NewMessage[mc.RespRematch](mc.CodeRematch)
	msgOtherPlayer.AddPayload(mc.RespRematch{IsTurn: otherSessionPlayer.IsTurn()})

	return msgPlayer, msgOtherPlayer, nil
}
