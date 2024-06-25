package battleship

import (
	cerr "github.com/saeidalz13/battleship-backend/internal/error"

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
	AttackGrid  GridInt
	DefenceGrid GridInt
	Ships       map[int]*Ship
	SessionID   string
	CurrentGame *Game
}

func NewPlayer(currentGame *Game, isHost, isTurn bool, sessionID string, gridSize int) *Player {
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
		CurrentGame: currentGame,
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

func (p *Player) HitShip(code, x, y int) {
	p.DefenceGrid[x][y] = PositionStateDefenceGridHit
	p.Ships[code].GotHit()
	p.Ships[code].hitCoordinates = append(p.Ships[code].hitCoordinates, NewCoordinates(x, y))
}

func (p *Player) IdentifyHitCoordsEssence(x, y int) (int, error) {
	positionCode := p.DefenceGrid[x][y]
	if positionCode == PositionStateDefenceGridHit {
		return PositionStateAttackGridHit, cerr.ErrDefenceGridPositionAlreadyHit(x, y)
	}
	if positionCode == PositionStateDefenceGridEmpty {
		return PositionStateAttackGridMiss, nil
	}

	// Passed this line means that positionCode is a ship code
	// since the attacker has hit a ship
	return positionCode, nil
}

func (p *Player) SetAttackGrid(newGrid GridInt) {
	p.AttackGrid = newGrid
}

func (p *Player) SetDefenceGrid(newGrid GridInt) {
	p.DefenceGrid = newGrid
}

func (p *Player) SunkShip() {
	p.SunkenShips++
}
