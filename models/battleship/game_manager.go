package battleship

import (
	"github.com/google/uuid"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"

	"sync"
)

type GameManager interface {
	CreateGame(difficulty uint8) (*Game, error)
	GetGame(gameUuid string) (*Game, error)

	isDifficultyValid(uint8) bool
}

type BattleshipGameManager struct {
	games map[string]*Game
	mu    sync.RWMutex
}

var _ GameManager = (*BattleshipGameManager)(nil)

func NewBattleshipGameManager() *BattleshipGameManager {
	return &BattleshipGameManager{
		games: make(map[string]*Game, 10),
	}
}
func (bgm *BattleshipGameManager) CreateGame(difficulty uint8) (*Game, error) {
	if !bgm.isDifficultyValid(difficulty) {
		return nil, cerr.ErrInvalidGameDifficulty()
	}

	gameUuid := uuid.NewString()[:6]
	bgm.games[gameUuid] = newGame(difficulty, gameUuid)

	return bgm.games[gameUuid], nil
}

func (bgm *BattleshipGameManager) GetGame(gameUuid string) (*Game, error) {
	bgm.mu.RLock()
	game, prs := bgm.games[gameUuid]
	bgm.mu.RUnlock()
	if !prs {
		return nil, cerr.ErrGameNotExists(gameUuid)
	}

	return game, nil
}

func (bgm *BattleshipGameManager) isDifficultyValid(difficulty uint8) bool {
	return !(difficulty != GameDifficultyEasy && difficulty != GameDifficultyNormal && difficulty != GameDifficultyHard)
}
