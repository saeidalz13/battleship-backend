package api

import (
	"encoding/json"
	"log"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
)

type RequestHandler interface {
	HandleCreateGame(bgm *mb.BattleshipGameManager, sessionId string) (*mb.Game, *mb.BattleshipPlayer, mc.Message[mc.RespCreateGame])
	HandleReadyPlayer(bgm *mb.BattleshipGameManager, sessionGame *mb.Game, sessionPlayer *mb.BattleshipPlayer) mc.Message[mc.NoPayload]
	HandleJoinPlayer(bgm *mb.BattleshipGameManager, sessionId string) (*mb.Game, *mb.BattleshipPlayer, mc.Message[mc.RespJoinGame])
	HandleAttack(*mb.Game, *mb.BattleshipPlayer, *mb.BattleshipPlayer, *mb.BattleshipGameManager) mc.Message[mc.RespAttack]
	HandleCallRematch(bgm *mb.BattleshipGameManager, sessionGame *mb.Game) (mc.Message[mc.NoPayload], error)
	HandleAcceptRematchCall(bgm *mb.BattleshipGameManager, sessionGame *mb.Game, sessionPlayer, otherSessionPlayer *mb.BattleshipPlayer) (mc.Message[mc.RespRematch], mc.Message[mc.RespRematch], error)
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

func (r Request) HandleCreateGame(bgm *mb.BattleshipGameManager, sessionId string) (*mb.Game, *mb.BattleshipPlayer, mc.Message[mc.RespCreateGame]) {
	var reqCreateGame mc.Message[mc.ReqCreateGame]
	respMsg := mc.NewMessage[mc.RespCreateGame](mc.CodeCreateGame)

	if err := json.Unmarshal(r.payload, &reqCreateGame); err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return nil, nil, respMsg
	}

	game, err := bgm.CreateGame(reqCreateGame.Payload.GameDifficulty)
	if err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrCreateGame)
		return nil, nil, respMsg
	}

	hostPlayer := bgm.CreateHostPlayerForGame(game, sessionId)

	respMsg.AddPayload(mc.RespCreateGame{GameUuid: bgm.GetGameUuid(game), HostUuid: hostPlayer.GetUuid()})
	return game, hostPlayer, respMsg
}

// Join user sends the game uuid and if this game exists,
// a new join player is created and added to the database
func (r Request) HandleJoinPlayer(bgm *mb.BattleshipGameManager, sessionId string) (*mb.Game, *mb.BattleshipPlayer, mc.Message[mc.RespJoinGame]) {
	var joinGameReq mc.Message[mc.ReqJoinGame]
	respMsg := mc.NewMessage[mc.RespJoinGame](mc.CodeJoinGame)

	if err := json.Unmarshal(r.payload, &joinGameReq); err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return nil, nil, respMsg
	}

	game, err := bgm.GetGame(joinGameReq.Payload.GameUuid)
	if err != nil {
		respMsg.AddError(err.Error(), cerr.ConstErrJoin)
		return nil, nil, respMsg
	}

	joinPlayer := bgm.CreateJoinPlayerForGame(game, sessionId)

	respMsg.AddPayload(mc.RespJoinGame{GameUuid: bgm.GetGameUuid(game), PlayerUuid: joinPlayer.GetUuid(), GameDifficulty: bgm.GetGameDifficulty(game)})
	return game, joinPlayer, respMsg
}

// User will choose the configurations of ships on defence grid.
// Then the grid is sent to backend and adjustment happens accordingly.
func (r Request) HandleReadyPlayer(bgm *mb.BattleshipGameManager, sessionGame *mb.Game, sessionPlayer *mb.BattleshipPlayer) mc.Message[mc.NoPayload] {
	var readyPlayerReq mc.Message[mc.ReqReadyPlayer]
	resp := mc.NewMessage[mc.NoPayload](mc.CodeReady)

	if err := json.Unmarshal(r.payload, &readyPlayerReq); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp
	}

	// Check to see if rows and cols are equal to game's grid size
	if err := bgm.SetPlayerReadyForGame(sessionGame, sessionPlayer, readyPlayerReq.Payload.DefenceGrid); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrReady)
		return resp
	}

	return resp
}

// Handle the attack logic for the incoming request
func (r Request) HandleAttack(
	game *mb.Game,
	attacker *mb.BattleshipPlayer,
	defender *mb.BattleshipPlayer,
	bgm *mb.BattleshipGameManager,
) mc.Message[mc.RespAttack] {
	var reqAttack mc.Message[mc.ReqAttack]
	resp := mc.NewMessage[mc.RespAttack](mc.CodeAttack)

	if err := json.Unmarshal(r.payload, &reqAttack); err != nil {
		resp.AddError(err.Error(), cerr.ConstErrInvalidPayload)
		return resp
	}

	coordinates := mb.NewCoordinates(reqAttack.Payload.X, reqAttack.Payload.Y)
	if !bgm.AreAttackCoordinatesValid(game, coordinates) {
		resp.AddError(cerr.ErrXorYOutOfGridBound(coordinates.X, coordinates.Y).Error(), cerr.ConstErrAttack)
		return resp
	}

	// Attack validity check
	if !attacker.IsTurn() {
		resp.AddError(cerr.ErrNotTurnForAttacker(attacker.GetUuid()).Error(), cerr.ConstErrAttack)
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

	if defender.IsAttackMiss(coordinates) {
		attacker.SetAttackGridToMiss(coordinates)

		resp.AddPayload(mc.RespAttack{
			X:               coordinates.X,
			Y:               coordinates.Y,
			PositionState:   mb.PositionStateAttackGridMiss,
			SunkenShipsHost: bgm.GetHostPlayerSunkenShips(game),
			SunkenShipsJoin: bgm.GetJoinPlayerSunkenShips(game),
			IsTurn:          attacker.IsTurn(),
		})
		return resp
	}

	shipCode := defender.GetShipCode(coordinates)
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
		resp.Payload.DefenderSunkenShipsCoords = defender.GetShipHitCoordinates(shipCode)

		// Check if this sunken ship was the last one and the attacker is lost
		if defender.AreAllShipsSunken() {
			defender.SetMatchStatusToLost()
			attacker.SetMatchStatusToWon()
		}
	}

	log.Println("attack complete")
	resp.Payload.SunkenShipsHost = bgm.GetHostPlayerSunkenShips(game)
	resp.Payload.SunkenShipsJoin = bgm.GetJoinPlayerSunkenShips(game)
	return resp
}

func (r Request) HandleCallRematch(bgm *mb.BattleshipGameManager, sessionGame *mb.Game) (mc.Message[mc.NoPayload], error) {
	respMsg := mc.NewMessage[mc.NoPayload](mc.CodeRematchCall)

	if bgm.IsRematchAlreadyCalled(sessionGame) {
		return respMsg, cerr.ErrGameAleardyRecalled()
	}

	bgm.CallRematchForGame(sessionGame)
	return respMsg, nil
}

func (r Request) HandleAcceptRematchCall(
	bgm *mb.BattleshipGameManager,
	sessionGame *mb.Game,
	sessionPlayer, otherSessionPlayer *mb.BattleshipPlayer,
) (mc.Message[mc.RespRematch], mc.Message[mc.RespRematch], error) {

	if err := bgm.ResetRematchForGame(sessionGame); err != nil {
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
