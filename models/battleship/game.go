package battleship

import (
	"sync"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
)

const (
	GameDifficultyEasy uint8 = iota
	GameDifficultyNormal
	GameDifficultyHard
)

const (
	GridSizeEasy   uint8 = 6
	GridSizeNormal uint8 = 7
	GridSizeHard   uint8 = 8

	ValidLowerBound uint8 = 0
)

type Game struct {
	uuid                    string
	hostPlayer              *BattleshipPlayer
	joinPlayer              *BattleshipPlayer
	difficulty              uint8
	gridSize                uint8
	validUpperBound         uint8
	rematchAlreadyRequested bool
	mu                      sync.Mutex
}

func newGame(difficulty uint8, uuid string) *Game {
	game := &Game{
		uuid:       uuid,
		difficulty: difficulty,
	}

	var newGridSize uint8
	if difficulty == GameDifficultyEasy {
		newGridSize = GridSizeEasy
	} else if difficulty == GameDifficultyNormal {
		newGridSize = GridSizeNormal
	} else {
		newGridSize = GridSizeHard
	}

	game.gridSize = newGridSize
	game.validUpperBound = newGridSize - 1

	return game
}

func (g *Game) GetUuid() string {
	return g.uuid
}

func (g *Game) CreateHostPlayer(sessionId string) *BattleshipPlayer {
	g.hostPlayer = newPlayer(true, true, sessionId, g.gridSize)
	return g.hostPlayer
}

func (g *Game) CreateJoinPlayer(sessionId string) *BattleshipPlayer {
	g.joinPlayer = newPlayer(false, false, sessionId, g.gridSize)
	return g.joinPlayer
}

func (g *Game) GetDifficulty() uint8 {
	return g.difficulty
}

func (g *Game) GetHostPlayer() *BattleshipPlayer {
	return g.hostPlayer
}

func (g *Game) GetJoinPlayer() *BattleshipPlayer {
	return g.joinPlayer
}

func (g *Game) GetPlayer(isHost bool) *BattleshipPlayer {
	if isHost {
		return g.hostPlayer
	}

	return g.joinPlayer
}

func (g *Game) IsReadyToStart() bool {
	return g.joinPlayer.IsReady() && g.hostPlayer.IsReady()
}

func (g *Game) IsRematchAlreadyCalled() bool {
	return g.rematchAlreadyRequested
}

func (g *Game) ResetRematchForGame() error {
	g.mu.Lock()
	g.rematchAlreadyRequested = false
	g.mu.Unlock()

	for _, player := range []*BattleshipPlayer{g.hostPlayer, g.joinPlayer} {
		if player == nil {
			return cerr.ErrPlayerNotExistForRematch()
		}
		player.PrepareForRematch(g.gridSize)
	}

	return nil
}

func (g *Game) CallRematchForGame() {
	g.mu.Lock()
	g.rematchAlreadyRequested = true
	g.mu.Unlock()
}

func (g *Game) AreAttackCoordinatesValid(coordinates Coordinates) bool {
	return !(coordinates.X > g.validUpperBound || coordinates.Y > g.validUpperBound || coordinates.X < ValidLowerBound || coordinates.Y < ValidLowerBound)
}

func (g *Game) SetPlayerReadyForGame(player Player, selectedGrid Grid) error {
	rows := uint8(len(selectedGrid))
	if rows != g.gridSize {
		return cerr.ErrDefenceGridRowsOutOfBounds(rows, g.gridSize)
	}
	cols := uint8(len(selectedGrid[0]))
	if cols != g.gridSize {
		return cerr.ErrDefenceGridColsOutOfBounds(cols, g.gridSize)
	}

	player.SetReady(selectedGrid)

	return nil
}
