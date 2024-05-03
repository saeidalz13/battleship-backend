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

	newPlayer := models.NewPlayer(newPlayerUuid, ws, true, true)
	newGame := models.NewGame(newGameUuid, newPlayer)

	s.Games[newGameUuid] = newGame
	s.Players[newPlayerUuid] = newPlayer

	newResp := models.RespCreateGame{
		Code:     models.CodeRespCreateGame,
		GameUuid: newGame.Uuid,
		HostUuid: newPlayer.Uuid,
	}

	if err := ws.WriteJSON(newResp); err != nil {
		return fmt.Errorf("failed to write json to ws conn, remote addr: %s; %v", ws.RemoteAddr().String(), err)
	}

	return nil
}

func ManageReadyPlayer(s *Server, ws *websocket.Conn, payload []byte) error {
	var readyPlayerReq models.ReqReadyPlayer
	if err := json.Unmarshal(payload, &readyPlayerReq); err != nil {
		return err
	}

	game, prs := s.Games[readyPlayerReq.GameUuid]
	if !prs {
		return ErrorGameNotExist(readyPlayerReq.GameUuid)
	}
	player, prs := s.Players[readyPlayerReq.PlayerUuid]
	if !prs {
		return ErrorPlayerNotExist(readyPlayerReq.PlayerUuid)
	}

	// Change player properties
	player.DefenceGrid = readyPlayerReq.DefenceGrid
	player.IsReady = true

	// send response to the player that sent the request
	if err := ws.WriteJSON(models.RespReadyPlayer{Success: true}); err != nil {
		return err
	}

	if game.Host.IsReady && game.Join.IsReady {
		jsonResp := models.Signal{Code: models.CodeRespSuccessStartGame}
		if err := SendJSONBothPlayers(game, jsonResp); err != nil {
			return err
		}
	}
	return nil
}

func JoinPlayerToGame(s *Server, ws *websocket.Conn, payload []byte) error {
	var joinGameReq models.ReqJoinGame
	if err := json.Unmarshal(payload, &joinGameReq); err != nil {
		return err
	}

	game, prs := s.Games[joinGameReq.GameUuid]
	if !prs {
		return ErrorGameNotExist(joinGameReq.GameUuid)
	}

	joinPlayerUuid := uuid.NewString()
	joinPlayer := models.NewPlayer(joinPlayerUuid, ws, false, false)

	game.Join = joinPlayer

	jsonPlayerJoined := models.RespJoinGame{Code: models.CodeRespSuccessJoinGame, PlayerUuid: joinPlayerUuid}
	if err := SendJSONBothPlayers(game, jsonPlayerJoined); err != nil {
		return err
	}
	return nil
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

	// TODO: should you give me x and y? or the entire grid? seems redundant...
	if err := ws.WriteJSON(models.RespSuccessAttack{Code: models.CodeRespSuccessAttack, IsTurn: false}); err != nil {
		return err
	}
	
	// TODO: Notify the other player about this event and tell them it's their turn
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
