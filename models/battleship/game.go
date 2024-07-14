package battleship

import (
	"sync"
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

func (g *Game) GetOtherPlayer(player *BattleshipPlayer) *BattleshipPlayer {
	if player.isHost {
		return g.joinPlayer
	}
	return g.hostPlayer
}
