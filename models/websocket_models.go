package models

import (
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
	CodeCreateGame int = iota
	CodeJoinGame
	CodeSelectGrid
	CodeReady
	CodeStartGame
	CodeAttack
	CodeEndGame
	CodeInvalidSignal
	CodeSignalAbsent // if the req msg does not contain "code" field

	CodeOtherPlayerDisconnected
	CodeOtherPlayerReconnected
	CodeOtherPlayerGracePeriod
	CodeSessionID
	CodeReceivedInvalidSessionID
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
	SessionID   string
	CurrentGame *Game
}

func NewPlayer(ws *websocket.Conn, currentGame *Game, isHost, isTurn bool, sessionID string) *Player {
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
}

func (p *Player) SetDefenceGrid(newGrid GridInt) {
	p.DefenceGrid = newGrid
}

func (p *Player) SunkShip() {
	p.SunkenShips++
}

type Game struct {
	IsFinished bool
	Uuid       string
	HostPlayer *Player
	JoinPlayer *Player
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

func (g *Game) FindPlayer(playerUuid string) (*Player, error) {
	switch playerUuid {
	case g.HostPlayer.Uuid:
		return g.HostPlayer, nil
	case g.JoinPlayer.Uuid:
		return g.JoinPlayer, nil
	default:
		return nil, cerr.ErrPlayerNotExist(playerUuid)
	}
}

func (g *Game) CreateJoinPlayer(ws *websocket.Conn, sessionID string) *Player {
	joinPlayer := NewPlayer(ws, g, false, false, sessionID)
	g.JoinPlayer = joinPlayer
	return joinPlayer
}

func (g *Game) CreateHostPlayer(ws *websocket.Conn, sessionID string) *Player {
	hostPlayer := NewPlayer(ws, g, true, true, sessionID)
	g.HostPlayer = hostPlayer
	return hostPlayer
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
	ships[PositionStateDefenceDestroyer] = NewShip(PositionStateDefenceDestroyer, 2)
	ships[PositionStateDefenceCruiser] = NewShip(PositionStateDefenceCruiser, 3)
	ships[PositionStateDefenceBattleship] = NewShip(PositionStateDefenceBattleship, 4)

	return ships
}

func (sh *Ship) GotHit() {
	sh.hits++
}

func (sh *Ship) IsSunk() bool {
	return sh.hits == sh.length
}

const (
	ManageGameCodeSuccess int = iota
	ManageGameCodePlayerDisconnect
	ManageGameCodeMaxTimeReached
)
