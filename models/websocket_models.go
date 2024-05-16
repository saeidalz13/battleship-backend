package models

import (
	"log"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const GameGridSize = 5

const (
	CodeCreateGame = iota
	CodeStartGame
	CodeEndGame
	CodeJoinGame
	CodeSelectGrid
	CodeAttack
	CodeReady
	CodeInvalidSignal
)

const (
	PositionStateNeutral = iota
	PositionStateMiss
	PositionStateHit
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
	IsReady     bool
	IsTurn      bool
	IsHost      bool
	Uuid        string
	AttackGrid  GridInt
	DefenceGrid GridInt
	WsConn      *websocket.Conn
}

func NewPlayer(ws *websocket.Conn, isHost, isTurn bool) *Player {
	return &Player{
		IsReady:     false,
		IsTurn:      isTurn,
		IsHost:      isHost,
		Uuid:        uuid.NewString()[:10],
		AttackGrid:  NewGrid(),
		DefenceGrid: NewGrid(),
		WsConn:      ws,
	}
}

func (p *Player) SetAttackGrid(newGrid GridInt) {
	p.AttackGrid = newGrid
	log.Printf("player %s attack grid set to: %+v\n", p.Uuid, p.AttackGrid)
}

func (p *Player) SetReady(newGrid GridInt) {
	p.DefenceGrid = newGrid
	p.IsReady = true
	log.Printf("player %s defence grid set to: %+v\n", p.Uuid, p.AttackGrid)
}

type Game struct {
	Uuid       string
	HostPlayer *Player
	JoinPlayer *Player
}

func NewGame() *Game {
	return &Game{
		Uuid: uuid.NewString()[:6],
	}
}

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
