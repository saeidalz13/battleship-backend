package models

import (
	"log"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
)

const (
	GameGridSize        = 5
	GameValidLowerBound = 0
	GameValidUpperBound = GameGridSize - 1

	SunkenShipsToLose = 3
)

const (
	PlayerMatchStatusLost      = -1
	PlayerMatchStatusUndefined = 0
	PlayerMatchStatusWon       = 1
)

const (
	CodeCreateGame = iota
	CodeJoinGame
	CodeSelectGrid
	CodeReady
	CodeStartGame
	CodeAttack
	CodeEndGame
	CodeInvalidSignal
)

const (
	PositionStateAttackGridMiss  = -1
	PositionStateAttackGridEmpty = 0
	PositionStateAttackGridHit   = 1
)

const (
	PositionStateDefenceGridHit    = -1
	PositionStateDefenceGridEmpty  = 0
	PositionStateDefenceDestroyer  = 2
	PositionStateDefenceCruiser    = 3
	PositionStateDefenceBattleship = 4
)

type Signal struct {
	Code int `json:"code"`
}

func NewSignal(code int) Signal {
	return Signal{Code: code}
}

type Message[T any] struct {
	Code    int     `json:"code"`
	Payload T       `json:"payload,omitempty"`
	Error   RespErr `json:"error,omitempty"`
}

type NoPayload bool

type MessageOption[T any] func(*Message[T]) error

func NewMessage[T any](code int) Message[T] {
	return Message[T]{Code: code}
}

func (m *Message[T]) AddPayload(payload T) {
	m.Payload = payload
}

func (m *Message[T]) AddError(errorDetails, message string) {
	m.Error = *NewRespErr(errorDetails, message)
}

type GridInt [][]int

// Creates a new default grid
// All indexes are zero/PositionStatusNeutral
func NewGrid() GridInt {
	grid := make(GridInt, GameGridSize)
	for i := 0; i < GameGridSize; i++ {
		grid[i] = make([]int, GameGridSize)
	}
	return grid
}

type Player struct {
	Uuid        string
	IsTurn      bool
	IsHost      bool
	IsReady     bool
	MatchStatus int
	SunkenShips int
	AttackGrid  [][]int
	DefenceGrid [][]int
	Ships       map[int]*Ship
	WsConn      *websocket.Conn
}

func NewPlayer(ws *websocket.Conn, isHost, isTurn bool) *Player {
	return &Player{
		IsTurn:      isTurn,
		IsHost:      isHost,
		IsReady:     false,
		MatchStatus: PlayerMatchStatusUndefined,
		SunkenShips: 0,
		Uuid:        uuid.NewString()[:10],
		AttackGrid:  NewGrid(),
		DefenceGrid: NewGrid(),
		Ships:       NewShipsMap(),
		WsConn:      ws,
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
}

func (p *Player) FetchDefenceGridPositionCode(x, y int) (int, error) {
	positionCode := p.DefenceGrid[x][y]
	if positionCode == PositionStateDefenceGridHit {
		return PositionStateAttackGridHit, cerr.ErrDefenceGridPositionAlreadyHit(x, y)
	}
	if positionCode == PositionStateDefenceGridEmpty {
		return PositionStateAttackGridMiss, nil
	}
	return positionCode, nil
}

func (p *Player) SetAttackGrid(newGrid GridInt) {
	p.AttackGrid = newGrid
	log.Printf("player %s attack grid set to: %+v\n", p.Uuid, p.AttackGrid)
}

func (p *Player) SetDefenceGrid(newGrid GridInt) {
	p.DefenceGrid = newGrid
	log.Printf("player %s defence grid set to: %+v\n", p.Uuid, p.DefenceGrid)
}

func (p *Player) SunkShip() {
	p.SunkenShips++
}

type Game struct {
	Uuid       string
	HostPlayer *Player
	JoinPlayer *Player
	IsFinished bool
}

func NewGame() *Game {
	return &Game{
		Uuid:       uuid.NewString()[:6],
		IsFinished: false,
	}
}

func (g *Game) FinishGame() {
	g.IsFinished = true
}

// returns a slice of players in the order of host then join.
func (g *Game) GetPlayers() []*Player {
	return []*Player{g.HostPlayer, g.JoinPlayer}
}

func (g *Game) CreateJoinPlayer(ws *websocket.Conn) {
	joinPlayer := NewPlayer(ws, false, false)
	g.JoinPlayer = joinPlayer
	log.Printf("join player created and added to game: %+v\n", joinPlayer.Uuid)
}

func (g *Game) CreateHostPlayer(ws *websocket.Conn) {
	hostPlayer := NewPlayer(ws, true, true)
	g.HostPlayer = hostPlayer
	log.Printf("host player created and added to game: %+v\n", hostPlayer.Uuid)
}

type Ship struct {
	Code   int
	length int
	hits   int
}

func NewShip(code, length int) *Ship {
	return &Ship{
		Code:   code,
		length: length,
		hits:   0,
	}
}

func NewShipsMap() map[int]*Ship {
	ships := make(map[int]*Ship, SunkenShipsToLose)
	ship1 := NewShip(PositionStateDefenceDestroyer, 2)
	ship2 := NewShip(PositionStateDefenceCruiser, 3)
	ship3 := NewShip(PositionStateDefenceBattleship, 4)

	ships[PositionStateDefenceDestroyer] = ship1
	ships[PositionStateDefenceCruiser] = ship2
	ships[PositionStateDefenceBattleship] = ship3

	return ships
}

func (sh *Ship) GotHit() {
	sh.hits++
}

func (sh *Ship) IsSunk() bool {
	return sh.hits == sh.length
}
