package api

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/customerr"
	md "github.com/saeidalz13/battleship-backend/models"
)

func CreateGame(s *Server, ws *websocket.Conn) *md.Message {
	newGame := md.NewGame()
	s.Games[newGame.Uuid] = newGame

	newGame.AddHostPlayer(ws)
	s.Players[newGame.HostPlayer.Uuid] = newGame.HostPlayer

	return &md.Message{
		Code: md.CodeSuccessCreateGame,
		Payload: md.RespCreateGame{
			GameUuid: newGame.Uuid,
			HostUuid: newGame.HostPlayer.Uuid,
		},
	}
}

func ManageReadyPlayer(s *Server, ws *websocket.Conn, payload []byte) (*md.Game, error) {
	var readyPlayerReq md.ReqReadyPlayer
	if err := json.Unmarshal(payload, &readyPlayerReq); err != nil {
		return nil, err
	}
	log.Printf("unmarshaled ready player payload: %+v\n", readyPlayerReq)

	game := s.FindGame(readyPlayerReq.GameUuid)
	if game == nil {
		return nil, cerr.ErrorGameNotExists(readyPlayerReq.GameUuid)
	}

	player := s.FindPlayer(readyPlayerReq.PlayerUuid)
	if player == nil {
		return nil, cerr.ErrorPlayerNotExist(readyPlayerReq.PlayerUuid)
	}

	player.SetReady(readyPlayerReq.DefenceGrid)
	return game, nil
}

func JoinPlayerToGame(s *Server, ws *websocket.Conn, payload []byte) (*md.Game, md.Message, error) {
	var joinGameReq md.ReqJoinGame
	if err := json.Unmarshal(payload, &joinGameReq); err != nil {
		return nil, md.Message{}, err
	}
	log.Printf("unmarshaled join game payload: %+v\n", joinGameReq)

	game, prs := s.Games[joinGameReq.GameUuid]
	if !prs {
		return nil, md.Message{}, cerr.ErrorGameNotExists(joinGameReq.GameUuid)
	}
	log.Printf("found game in database: %+v\n", game)

	game.AddJoinPlayer(ws)
	resp := md.NewMessage(md.CodeRespSuccessJoinGame, md.WithPayload(md.RespJoinGame{PlayerUuid: game.JoinPlayer.Uuid}))

	return game, resp, nil
}

func Attack(s *Server, ws *websocket.Conn, payload []byte) error {
	var reqAttack md.ReqAttack
	if err := json.Unmarshal(payload, &reqAttack); err != nil {
		return err
	}

	game, prs := s.Games[reqAttack.GameUuid]
	if !prs {
		return cerr.ErrorGameNotExists(reqAttack.GameUuid)
	}
	player, prs := s.Players[reqAttack.PlayerUuid]
	if !prs {
		return cerr.ErrorPlayerNotExist(reqAttack.PlayerUuid)
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

func SendJSONBothPlayers(game *md.Game, v interface{}) error {
	playerOfGames := game.GetPlayers()
	for _, player := range playerOfGames {
		if err := player.WsConn.WriteJSON(v); err != nil {
			return err
		}
		log.Printf("message sent to player: %+v\n", player.Uuid)
	}
	return nil
}
