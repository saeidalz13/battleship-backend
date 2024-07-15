package battleship

import (
	"github.com/google/uuid"
)

const (
	PlayerMatchStatusUndefined uint8 = iota
	PlayerMatchStatusLost
	PlayerMatchStatusWon
)

const (
	PositionStateAttackGridEmpty uint8 = iota
	PositionStateAttackGridMiss
	PositionStateAttackGridHit
)

type Player interface {
	GetSessionId() string
	GetUuid() string

	AreAllShipsSunken() bool
	IsShipSunken(uint8) bool
	IsWinner() bool
	IncrementShipHit(code uint8, coordinates Coordinates)
	SetAttackGrid(newGrid Grid)
	SetReady(newGrid Grid)
	IncrementSunkenShips()
	IsAttackGridEmptyInCoordinates(coordinates Coordinates) bool
	IsDefenceGridAlreadyHitInCoordinates(coordinates Coordinates) bool
	IsAttackMiss(coordinates Coordinates) bool
	PrepareForRematch(uint8)

	SetAttackGridToHit(coordinates Coordinates)
	SetAttackGridToMiss(coordinates Coordinates)

	SetMatchStatusToWon()
	SetMatchStatusToLost()

	SetTurnTrue()
	SetTurnFalse()
	IsHost() bool

	GetShipCode(coordinates Coordinates) uint8
	GetShipHitCoordinates(shipCode uint8) []Coordinates

	GetMatchStatus() uint8

	IsReady() bool
	IsTurn() bool
}

type BattleshipPlayer struct {
	isTurn      bool
	isHost      bool
	isReady     bool
	matchStatus uint8
	sunkenShips uint8
	uuid        string
	sessionID   string
	attackGrid  Grid
	defenceGrid Grid
	ships       map[uint8]*Ship
}

func newPlayer(isHost, isTurn bool, sessionID string, gridSize uint8) *BattleshipPlayer {
	return &BattleshipPlayer{
		isTurn:      isTurn,
		isHost:      isHost,
		isReady:     false,
		matchStatus: PlayerMatchStatusUndefined,
		sunkenShips: 0,
		uuid:        uuid.NewString()[:10],
		attackGrid:  NewGrid(gridSize),
		defenceGrid: NewGrid(gridSize),
		ships:       NewShipsMap(),
		sessionID:   sessionID,
	}
}

func (bp *BattleshipPlayer) GetSessionId() string {
	return bp.sessionID
}

func (bp *BattleshipPlayer) GetUuid() string {
	return bp.uuid
}

func (bp *BattleshipPlayer) IsAttackGridEmptyInCoordinates(coordinates Coordinates) bool {
	return bp.attackGrid[coordinates.X][coordinates.Y] == PositionStateAttackGridEmpty
}

func (bp *BattleshipPlayer) IsDefenceGridAlreadyHitInCoordinates(coordinates Coordinates) bool {
	return bp.defenceGrid[coordinates.X][coordinates.Y] == PositionStateDefenceGridHit
}

func (bp *BattleshipPlayer) IsAttackMiss(coordinates Coordinates) bool {
	return bp.defenceGrid[coordinates.X][coordinates.Y] == PositionStateDefenceGridEmpty
}

func (bp *BattleshipPlayer) SetAttackGridToMiss(coordinates Coordinates) {
	bp.attackGrid[coordinates.X][coordinates.Y] = PositionStateAttackGridMiss
}

func (bp *BattleshipPlayer) SetAttackGridToHit(coordinates Coordinates) {
	bp.attackGrid[coordinates.X][coordinates.Y] = PositionStateAttackGridHit
}

func (bp *BattleshipPlayer) IncrementShipHit(code uint8, coordinates Coordinates) {
	bp.defenceGrid[coordinates.X][coordinates.Y] = PositionStateDefenceGridHit
	bp.ships[code].GotHit()
	bp.ships[code].hitCoordinates = append(bp.ships[code].hitCoordinates, coordinates)
}

func (bp *BattleshipPlayer) SetMatchStatusToWon() {
	bp.matchStatus = PlayerMatchStatusWon
}

func (bp *BattleshipPlayer) SetMatchStatusToLost() {
	bp.matchStatus = PlayerMatchStatusLost
}

func (bp *BattleshipPlayer) GetShipCode(coordinates Coordinates) uint8 {
	return bp.defenceGrid[coordinates.X][coordinates.Y]
}

func (bp *BattleshipPlayer) GetShipHitCoordinates(shipCode uint8) []Coordinates {
	return bp.ships[shipCode].GetHitCoordinates()
}

func (bp *BattleshipPlayer) AreAllShipsSunken() bool {
	return bp.sunkenShips == sunkenShipsToLose
}

func (bp *BattleshipPlayer) IsShipSunken(shipCode uint8) bool {
	return bp.ships[shipCode].IsSunk()
}

func (bp *BattleshipPlayer) SetAttackGrid(newGrid Grid) {
	bp.attackGrid = newGrid
}

func (bp *BattleshipPlayer) SetReady(newGrid Grid) {
	bp.defenceGrid = newGrid
	bp.isReady = true
}

func (bp *BattleshipPlayer) IncrementSunkenShips() {
	bp.sunkenShips++
}

func (bp *BattleshipPlayer) IsWinner() bool {
	return bp.matchStatus == PlayerMatchStatusWon
}

func (bp *BattleshipPlayer) PrepareForRematch(gridSize uint8) {
	bp.matchStatus = PlayerMatchStatusUndefined
	bp.isReady = false
	bp.ships = NewShipsMap()
	bp.sunkenShips = 0
	bp.attackGrid = NewGrid(gridSize)
	bp.defenceGrid = NewGrid(gridSize)
}

func (bp *BattleshipPlayer) SetTurnTrue() {
	bp.isTurn = true
}

func (bp *BattleshipPlayer) SetTurnFalse() {
	bp.isTurn = false
}

func (bp *BattleshipPlayer) IsTurn() bool {
	return bp.isTurn
}

func (bp *BattleshipPlayer) IsHost() bool {
	return bp.isHost
}

func (bp *BattleshipPlayer) GetMatchStatus() uint8 {
	return bp.matchStatus
}

func (bp *BattleshipPlayer) IsReady() bool {
	return bp.isReady
}

var _ Player = (*BattleshipPlayer)(nil)
