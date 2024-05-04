package api

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/internal/log"
	"github.com/saeidalz13/battleship-backend/models"
)

func CreateGame(s *Server, ws *websocket.Conn) *models.RespCreateGame {
	newGame := models.NewGame()
	s.Games[newGame.Uuid] = newGame

	newGame.AddHostPlayer(ws)
	s.Players[newGame.HostPlayer.Uuid] = newGame.HostPlayer

	return &models.RespCreateGame{
		Code:     models.CodeSuccessCreateGame,
		GameUuid: newGame.Uuid,
		HostUuid: newGame.HostPlayer.Uuid,
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

func JoinPlayerToGame(s *Server, ws *websocket.Conn, payload []byte) (*models.Game, models.RespJoinGame, error) {
	var joinGameReq models.ReqJoinGame
	if err := json.Unmarshal(payload, &joinGameReq); err != nil {
		return nil, models.RespJoinGame{}, err
	}
	log.LogCustom("unmarshaled join game payload:", joinGameReq)

	game, prs := s.Games[joinGameReq.GameUuid]
	if !prs {
		return nil, models.RespJoinGame{}, ErrorGameNotExist(joinGameReq.GameUuid)
	}
	log.LogCustom("found game in database:", game)

	game.AddJoinPlayer(ws)
	resp := models.RespJoinGame{Code: models.CodeRespSuccessJoinGame, PlayerUuid: game.JoinPlayer.Uuid}
	return game, resp, nil
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
		game.HostPlayer.AttackGrid = reqAttack.AttackGrid
		game.HostPlayer.IsTurn = false
	} else {
		game.JoinPlayer.AttackGrid = reqAttack.AttackGrid
		game.JoinPlayer.IsTurn = false
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
		log.LogCustom("message sent to player:", player.Uuid)
	}
	return nil
}
