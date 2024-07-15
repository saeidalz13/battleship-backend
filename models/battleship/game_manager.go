package battleship

import (
	"github.com/google/uuid"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"

	"sync"
)

type GameManager interface {
	CreateGame(difficulty uint8) (*Game, error)
	GetGame(gameUuid string) (*Game, error)
	GetPlayer(game *Game, isHost bool) *BattleshipPlayer
	GetOtherPlayerForGame(game *Game, player Player) *BattleshipPlayer
	GetGameUuid(game *Game) string
	GetGameDifficulty(game *Game) uint8
	CreateHostPlayerForGame(game *Game, sessionId string) *BattleshipPlayer
	CreateJoinPlayerForGame(game *Game, sessionId string) *BattleshipPlayer
	SetPlayerReadyForGame(game *Game, player Player, selectedGrid Grid) error
	AreAttackCoordinatesValid(game *Game, coordinates Coordinates) bool
	CallRematchForGame(game *Game)
	ResetRematchForGame(game *Game) error
	IsRematchAlreadyCalled(game *Game) bool
	IsGameReadyToStart(game *Game) bool
	GetJoinPlayerSunkenShips(game *Game) uint8
	GetHostPlayerSunkenShips(game *Game) uint8

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

	if game == nil {
		bgm.mu.Lock()
		delete(bgm.games, gameUuid)
		bgm.mu.Unlock()
		return nil, cerr.ErrGameIsNil(gameUuid)
	}

	return game, nil
}

func (bgm *BattleshipGameManager) GetPlayer(game *Game, isHost bool) *BattleshipPlayer {
	if isHost {
		return game.hostPlayer
	}

	return game.joinPlayer
}

func (bgm *BattleshipGameManager) GetOtherPlayerForGame(game *Game, player Player) *BattleshipPlayer {
	if player.IsHost() {
		return game.joinPlayer
	}
	return game.hostPlayer
}

func (bgm *BattleshipGameManager) GetGameUuid(game *Game) string {
	return game.uuid
}

func (bgm *BattleshipGameManager) GetGameDifficulty(game *Game) uint8 {
	return game.difficulty
}

func (bgm *BattleshipGameManager) CreateHostPlayerForGame(game *Game, sessionId string) *BattleshipPlayer {
	game.hostPlayer = newPlayer(true, true, sessionId, game.gridSize)
	return game.hostPlayer
}

func (bgm *BattleshipGameManager) CreateJoinPlayerForGame(game *Game, sessionId string) *BattleshipPlayer {
	game.joinPlayer = newPlayer(false, false, sessionId, game.gridSize)
	return game.joinPlayer
}

func (bgm *BattleshipGameManager) GetHostPlayerSunkenShips(game *Game) uint8 {
	return game.hostPlayer.sunkenShips
}

func (bgm *BattleshipGameManager) GetJoinPlayerSunkenShips(game *Game) uint8 {
	return game.joinPlayer.sunkenShips
}

func (bgm *BattleshipGameManager) SetPlayerReadyForGame(game *Game, player Player, selectedGrid Grid) error {
	rows := uint8(len(selectedGrid))
	if rows != game.gridSize {
		return cerr.ErrDefenceGridRowsOutOfBounds(rows, game.gridSize)
	}
	cols := uint8(len(selectedGrid[0]))
	if cols != game.gridSize {
		return cerr.ErrDefenceGridColsOutOfBounds(cols, game.gridSize)
	}

	player.SetReady(selectedGrid)

	return nil
}

func (bgm *BattleshipGameManager) AreAttackCoordinatesValid(game *Game, coordinates Coordinates) bool {
	return !(coordinates.X > game.validUpperBound || coordinates.Y > game.validUpperBound || coordinates.X < ValidLowerBound || coordinates.Y < ValidLowerBound)
}

func (bgm *BattleshipGameManager) CallRematchForGame(game *Game) {
	game.mu.Lock()
	game.rematchAlreadyRequested = true
	game.mu.Unlock()
}

func (bgm *BattleshipGameManager) ResetRematchForGame(game *Game) error {
	game.mu.Lock()
	game.rematchAlreadyRequested = false
	game.mu.Unlock()

	for _, player := range []*BattleshipPlayer{game.hostPlayer, game.joinPlayer} {
		if player == nil {
			return cerr.ErrPlayerNotExistForRematch()
		}
		player.PrepareForRematch(game.gridSize)
	}

	return nil
}
func (bgm *BattleshipGameManager) IsRematchAlreadyCalled(game *Game) bool {
	return game.rematchAlreadyRequested
}

func (bgm *BattleshipGameManager) IsGameReadyToStart(game *Game) bool {
	return game.joinPlayer.IsReady() && game.hostPlayer.IsReady()
}

func (bgm *BattleshipGameManager) isDifficultyValid(difficulty uint8) bool {
	return !(difficulty != GameDifficultyEasy && difficulty != GameDifficultyNormal && difficulty != GameDifficultyHard)
}
