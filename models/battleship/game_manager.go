package battleship

import (
	"github.com/google/uuid"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"

	"sync"
)

type GameManager interface {
	CreateGame(difficulty, mode uint8) (*Game, error)
	FetchGame(gameUuid string) (*Game, error)
	TerminateGame(gameUuid string)

	isDifficultyValid(uint8) bool
	isModeValid(uint8) bool
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
func (bgm *BattleshipGameManager) CreateGame(difficulty, mode uint8) (*Game, error) {
	if !bgm.isDifficultyValid(difficulty) {
		return nil, cerr.ErrInvalidGameDifficulty(difficulty)
	}

	if !bgm.isModeValid(mode) {
		return nil, cerr.ErrInvalidGameMode(mode)
	}

	gameUuid := uuid.NewString()[:6]
	bgm.games[gameUuid] = newGame(difficulty, mode, gameUuid)

	return bgm.games[gameUuid], nil
}

func (bgm *BattleshipGameManager) FetchGame(gameUuid string) (*Game, error) {
	bgm.mu.RLock()
	game, prs := bgm.games[gameUuid]
	bgm.mu.RUnlock()
	if !prs {
		return nil, cerr.ErrGameNotExists(gameUuid)
	}

	return game, nil
}

func (bgm *BattleshipGameManager) TerminateGame(gameUuid string) {
	bgm.mu.Lock()
	defer bgm.mu.Unlock()

	delete(bgm.games, gameUuid)
}

func (bgm *BattleshipGameManager) isDifficultyValid(difficulty uint8) bool {
	return !(difficulty != GameDifficultyEasy && difficulty != GameDifficultyNormal && difficulty != GameDifficultyHard)
}

func (bgm *BattleshipGameManager) isModeValid(mode uint8) bool {
	return mode == GameModeDefault || mode == GameModeMine
}
