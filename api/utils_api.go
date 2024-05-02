package api

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/saeidalz13/battleship-backend/models"
)

func CreateGame(s *Server, ws *websocket.Conn) error {
	newGameUuid := uuid.NewString()[:6]
	newPlayerUuid := uuid.NewString()

	newPlayer := models.NewPlayer(newPlayerUuid, ws, true)
	newGame := models.NewGame(newGameUuid, newPlayer)

	s.Games[newGameUuid] = newGame
	s.Players[newPlayerUuid] = newPlayer

	newResp := models.CreateGameResp{
		GameUuid: newGame.Uuid,
		HostUuid: newPlayer.Uuid,
	}

	if err := ws.WriteJSON(newResp); err != nil {
		return fmt.Errorf("failed to write json to ws conn, remote addr: %s; %v", ws.RemoteAddr().String(), err)
	}

	return nil
}

func ManageReadyPlayer(s *Server, ws *websocket.Conn, payload []byte) error {
	var readyPlayerReq models.ReadyPlayerReq
	if err := json.Unmarshal(payload, &readyPlayerReq); err != nil {
		return err
	}

	game, prs := s.Games[readyPlayerReq.GameUuid]
	if !prs {
		return fmt.Errorf("game with this uuid does not exist, uuid: %s", readyPlayerReq.GameUuid)
	}
	player, prs := s.Players[readyPlayerReq.PlayerUuid]
	if !prs {
		return fmt.Errorf("player with this uuid does not exist, uuid: %s", readyPlayerReq.PlayerUuid)
	}

	// Change player properties
	player.DefenceGrid = readyPlayerReq.DefenceGrid
	player.IsReady = true

	// send response to the player that sent the request
	if err := ws.WriteJSON(models.ReadyPlayerResp{Success: true}); err != nil {
		return err
	}

	if game.Host.IsReady && game.Join.IsReady {
		for _, player := range game.GetPlayers() {
			if err := player.WsConn.WriteJSON(models.SignalStruct{Code: models.CodeStartGame}); err != nil {
				return err
			}
		}
	}
	return nil
}
