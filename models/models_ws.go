package models

import (
	"log"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Create game
	CodeReqCreateGame = iota
	CodeSuccessCreateGame
	CodeFailCreateGame

	// Start game
	CodeRespStartGame
	CodeRespEndGame

	// Join game
	CodeReqJoinGame
	CodeRespSuccessJoinGame
	CodeRespFailJoinGame

	// Select grid
	// CodeReqSelectGrid
	// CodeRespSuccessSelectGrid
	// CodeRespFailSelectGrid

	// Attack
	CodeReqAttack
	CodeRespSuccessAttack
	CodeRespFailAttack

	// Ready
	CodeReqReady
	CodeRespSuccessReady
	CodeRespFailReady

	// Misc
	CodeRespInvalidSignal
)

const (
	KeyGameUuid    string = "game_uuid"
	KeyPlayerUuid  string = "player_uuid"
	KeyDefenceGrid string = "defence_grid"
)

type Signal struct {
	Code int `json:"code"`
}

func NewSignal(code int) Signal {
	return Signal{Code: code}
}

type Message struct {
	Code    int         `json:"code"`
	Payload interface{} `json:"payload,omitempty"`
	Error   RespFail    `json:"error,omitempty"`
}

type MessageOption func(*Message) error

func NewMessage(code int, opts ...MessageOption) Message {
	message := Message{Code: code}

	for _, opt := range opts {
		if err := opt(&message); err != nil {
			log.Println("failed to create new message: ", err)
			return message
		}
	}
	return message
}

func WithPayload(p interface{}) MessageOption {
	return func(m *Message) error {
		m.Payload = p
		return nil
	}
}

func WithError(errorDetails, message string) MessageOption {
	respFail := NewRespFail(errorDetails, message)
	return func(m *Message) error {
		m.Error = *respFail
		return nil
	}
}

type GridInt [][]int

func NewGrid() GridInt {
	grid := make(GridInt, 0)
	col := []int{0, 0, 0, 0, 0}

	rowColSize := 5
	for i := 0; i <= rowColSize; i++ {
		grid = append(grid, col)
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

func (g *Game) AddJoinPlayer(ws *websocket.Conn) {
	joinPlayer := NewPlayer(ws, false, false)
	g.JoinPlayer = joinPlayer
	log.Printf("join player created and added to game: %+v\n", joinPlayer.Uuid)
}

func (g *Game) AddHostPlayer(ws *websocket.Conn) {
	hostPlayer := NewPlayer(ws, true, true)
	g.HostPlayer = hostPlayer
	log.Printf("host player created and added to game: %+v\n", hostPlayer.Uuid)
}
