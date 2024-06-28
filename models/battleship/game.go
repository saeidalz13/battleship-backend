package battleship

import (
	cerr "github.com/saeidalz13/battleship-backend/internal/error"

	"github.com/google/uuid"
)

// const (
// 	GameValidLowerBound = 0
// 	GameValidUpperBound = GameGridSize - 1
// )

const (
	GameDifficultyEasy int = iota
	GameDifficultyNormal
	GameDifficultyHard
)

const (
	GridSizeEasy   int = 6
	GridSizeNormal int = 7
	GridSizeHard   int = 8

	ValidLowerBound int = 0
)

type Game struct {
	isFinished      bool
	Uuid            string
	HostPlayer      *Player
	JoinPlayer      *Player
	Players         map[string]*Player
	Difficulty      int
	GridSize        int
	ValidUpperBound int
}

func NewGame(difficulty int) Game {
	game := Game{
		Uuid:            uuid.NewString()[:6],
		isFinished:      false,
		Players:         make(map[string]*Player),
		Difficulty:      difficulty,
	}

	var newGridSize int
	if difficulty == GameDifficultyEasy {
		newGridSize = GridSizeEasy
	} else if difficulty == GameDifficultyNormal {
		newGridSize = GridSizeNormal
	} else {
		newGridSize = GridSizeHard
	}

	game.GridSize = newGridSize
	game.ValidUpperBound = newGridSize - 1

	return game
}

func (g *Game) FinishGame() {
	g.isFinished = true
}

// returns a slice of players in the order of host then join.
func (g *Game) GetPlayers() []*Player {
	return []*Player{g.HostPlayer, g.JoinPlayer}
}

func (g *Game) FindPlayer(playerUuid string) (*Player, error) {
	player, prs := g.Players[playerUuid]
	if !prs {
		return nil, cerr.ErrPlayerNotExist(playerUuid)
	}

	return player, nil
}

func (g *Game) CreateJoinPlayer(sessionID string) *Player {
	joinPlayer := NewPlayer(false, false, sessionID, g.GridSize)
	g.JoinPlayer = joinPlayer

	g.Players[joinPlayer.Uuid] = joinPlayer
	return joinPlayer
}

func (g *Game) CreateHostPlayer(sessionID string) *Player {
	hostPlayer := NewPlayer(true, true, sessionID, g.GridSize)
	g.HostPlayer = hostPlayer

	g.Players[hostPlayer.Uuid] = hostPlayer
	return hostPlayer
}
