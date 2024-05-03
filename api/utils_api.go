package api

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/models"
)

func CreateGame(s *Server, ws *websocket.Conn) *models.RespCreateGame {
	newGameUuid := uuid.NewString()[:6]
	newPlayerUuid := uuid.NewString()

	newPlayer := models.NewPlayer(newPlayerUuid, ws, true, true)
	newGame := models.NewGame(newGameUuid, newPlayer)

	s.Games[newGameUuid] = newGame
	s.Players[newPlayerUuid] = newPlayer

	return &models.RespCreateGame{
		Code:     models.CodeRespCreateGame,
		GameUuid: newGame.Uuid,
		HostUuid: newPlayer.Uuid,
	}

}

func ManageReadyPlayer(s *Server, ws *websocket.Conn, payload []byte) (*models.Game, error) {
	var readyPlayerReq models.ReqReadyPlayer
	if err := json.Unmarshal(payload, &readyPlayerReq); err != nil {
		return nil, err
	}

	game, prs := s.Games[readyPlayerReq.GameUuid]
	if !prs {
		return nil, ErrorGameNotExist(readyPlayerReq.GameUuid)
	}
	player, prs := s.Players[readyPlayerReq.PlayerUuid]
	if !prs {
		return nil, ErrorPlayerNotExist(readyPlayerReq.PlayerUuid)
	}

	// Change player properties
	player.DefenceGrid = readyPlayerReq.DefenceGrid
	player.IsReady = true

	return game, nil
}

func JoinPlayerToGame(s *Server, ws *websocket.Conn, payload []byte) (*models.Game, *models.RespJoinGame, error) {
	var joinGameReq models.ReqJoinGame
	if err := json.Unmarshal(payload, &joinGameReq); err != nil {
		return nil, nil, err
	}

	game, prs := s.Games[joinGameReq.GameUuid]
	if !prs {
		return nil, nil, ErrorGameNotExist(joinGameReq.GameUuid)
	}

	joinPlayerUuid := uuid.NewString()
	joinPlayer := models.NewPlayer(joinPlayerUuid, ws, false, false)

	game.Join = joinPlayer
	resp := models.RespJoinGame{Code: models.CodeRespSuccessJoinGame, PlayerUuid: joinPlayerUuid}
	return game, &resp, nil
}

func Attack(s *Server, ws *websocket.Conn, payload []byte) error {
	var reqAttack models.ReqAttack
	if err := json.Unmarshal(payload, &reqAttack); err != nil {
		return err
	}

	game, prs := s.Games[reqAttack.GameUuid]
	if !prs {
		return ErrorGameNotExist(reqAttack.GameUuid)
	}
	player, prs := s.Players[reqAttack.PlayerUuid]
	if !prs {
		return ErrorPlayerNotExist(reqAttack.PlayerUuid)
	}
	player.IsTurn = false

	if player.IsHost {
		game.Host.AttackGrid = reqAttack.AttackGrid
		game.Host.IsTurn = false
	} else {
		game.Join.AttackGrid = reqAttack.AttackGrid
		game.Join.IsTurn = false
	}
	return nil
}

func EndGame(s *Server, ws *websocket.Conn, payload []byte) error {
	return nil
}

func SendJSONBothPlayers(game *models.Game, v interface{}) error {
	playerOfGames := game.GetPlayers()
	for _, player := range playerOfGames {
		if err := player.WsConn.WriteJSON(v); err != nil {
			return err
		}
	}
	return nil
}
