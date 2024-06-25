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
	GridSizeEasy   int = 5
	GridSizeNormal int = 6
	GridSizeHard   int = 7
)

type Game struct {
	isFinished      bool
	Uuid            string
	HostPlayer      *Player
	JoinPlayer      *Player
	Players         map[string]*Player
	Difficulty      int
	GridSize        int
	ValidLowerBound int
	ValidUpperBound int
}

func NewGame(difficulty int) Game {
	game := Game{
		Uuid:            uuid.NewString()[:6],
		isFinished:      false,
		Players:         make(map[string]*Player),
		Difficulty:      difficulty,
		ValidLowerBound: 0,
	}

	var gsize int
	if difficulty == GameDifficultyEasy {
		gsize = GridSizeEasy
	} else if difficulty == GameDifficultyNormal {
		gsize = GridSizeNormal
	} else {
		gsize = GridSizeHard
	}

	game.GridSize = gsize
	game.ValidUpperBound = gsize - 1

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
	joinPlayer := NewPlayer(g, false, false, sessionID, g.GridSize)
	g.JoinPlayer = joinPlayer

	g.Players[joinPlayer.Uuid] = joinPlayer
	return joinPlayer
}

func (g *Game) CreateHostPlayer(sessionID string) *Player {
	hostPlayer := NewPlayer(g, true, true, sessionID, g.GridSize)
	g.HostPlayer = hostPlayer

	g.Players[hostPlayer.Uuid] = hostPlayer
	return hostPlayer
}
