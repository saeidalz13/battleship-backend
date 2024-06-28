package battleship

import (
	"github.com/google/uuid"
)

const (
	PlayerMatchStatusLost      = -1
	PlayerMatchStatusUndefined = 0
	PlayerMatchStatusWon       = 1

	PositionStateAttackGridMiss  = -1
	PositionStateAttackGridEmpty = 0
	PositionStateAttackGridHit   = 1
)

type Player struct {
	Uuid        string
	IsTurn      bool
	IsHost      bool
	IsReady     bool
	MatchStatus int
	SunkenShips int
	AttackGrid  Grid
	DefenceGrid Grid
	Ships       map[int]*Ship
	SessionID   string
}

func NewPlayer(isHost, isTurn bool, sessionID string, gridSize int) *Player {
	return &Player{
		IsTurn:      isTurn,
		IsHost:      isHost,
		IsReady:     false,
		MatchStatus: PlayerMatchStatusUndefined,
		SunkenShips: 0,
		Uuid:        uuid.NewString()[:10],
		AttackGrid:  NewGrid(gridSize),
		DefenceGrid: NewGrid(gridSize),
		Ships:       NewShipsMap(),
		SessionID:   sessionID,
	}
}

func (p *Player) IsLoser() bool {
	return p.SunkenShips == SunkenShipsToLose
}

func (p *Player) IsShipSunken(code int) bool {
	if p.Ships[code].IsSunk() {
		p.SunkenShips++
		return true
	}
	return false
}

func (p *Player) HitShip(code int, coordinates Coordinates) {
	p.DefenceGrid[coordinates.X][coordinates.Y] = PositionStateDefenceGridHit
	p.Ships[code].GotHit()
	p.Ships[code].hitCoordinates = append(p.Ships[code].hitCoordinates, coordinates)
}

func (p *Player) SetAttackGrid(newGrid Grid) {
	p.AttackGrid = newGrid
}

func (p *Player) SetDefenceGrid(newGrid Grid) {
	p.DefenceGrid = newGrid
}

func (p *Player) SunkShip() {
	p.SunkenShips++
}

func (p *Player) DidAttackThisCoordinatesBefore(coordinates Coordinates) bool {
	return p.AttackGrid[coordinates.X][coordinates.Y] != PositionStateAttackGridEmpty
}

func (p *Player) IsIncomingAttackMiss(coordinates Coordinates) bool {
	return p.DefenceGrid[coordinates.X][coordinates.Y] == PositionStateDefenceGridEmpty
}

func (p *Player) AreCoordinatesAlreadyHit(coordinates Coordinates) bool {
	return p.DefenceGrid[coordinates.X][coordinates.Y] == PositionStateDefenceGridHit
}
