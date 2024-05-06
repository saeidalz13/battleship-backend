package api

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

type WsRequestHandler interface {
	CreateGame() (*md.Message, error)
	ManageReadyPlayer() (*md.Message, *md.Game, error)
	JoinPlayerToGame() (*md.Message, *md.Game, error)
}

type WsRequest struct {
	Server  *Server
	Ws      *websocket.Conn
	Payload []byte
}

// This tells the compiler that WsRequest struct must be of type of WsRequestHandler
var _ WsRequestHandler = (*WsRequest)(nil)

func NewWsRequest(server *Server, ws *websocket.Conn, payloads ...[]byte) *WsRequest {
	if len(payloads) > 1 {
		log.Println("cannot accept more than one payload")
		return nil
	}

	wsReq := WsRequest{
		Server: server,
		Ws:     ws,
	}
	if len(payloads) != 0 {
		wsReq.Payload = payloads[0]
	}
	return &wsReq
}

func (w *WsRequest) CreateGame() (*md.Message, error) {
	newGame := md.NewGame()
	w.Server.Games[newGame.Uuid] = newGame

	newGame.AddHostPlayer(w.Ws)
	w.Server.Players[newGame.HostPlayer.Uuid] = newGame.HostPlayer

	resp := md.NewMessage(md.CodeSuccessCreateGame,
		md.WithPayload(
			md.RespCreateGame{
				GameUuid: newGame.Uuid,
				HostUuid: newGame.HostPlayer.Uuid,
			},
		))
	return &resp, nil
}

func (w *WsRequest) ManageReadyPlayer() (*md.Message, *md.Game, error) {
	var readyPlayerReq md.Message
	if err := json.Unmarshal(w.Payload, &readyPlayerReq); err != nil {
		return nil, nil, err
	}
	log.Printf("unmarshaled ready player payload: %+v\n", readyPlayerReq)

	correctedPayload, err := TypeAssertStringPayload(readyPlayerReq.Payload, md.KeyGameUuid, md.KeyPlayerUuid)
	if err != nil {
		return &md.Message{}, nil, err
	}

	game := w.Server.FindGame(correctedPayload[md.KeyGameUuid])
	if game == nil {
		return nil, nil, cerr.ErrorGameNotExists(correctedPayload[md.KeyGameUuid])
	}

	player := w.Server.FindPlayer(correctedPayload[md.KeyPlayerUuid])
	if player == nil {
		return nil, nil, cerr.ErrorPlayerNotExist(correctedPayload[md.KeyPlayerUuid])
	}

	defenceGrid, err := TypeAssertGridIntPayload(readyPlayerReq.Payload, md.KeyDefenceGrid)
	if err != nil {
		return &md.Message{}, nil, err
	}
	player.SetReady(defenceGrid)
	resp := md.NewMessage(md.CodeRespSuccessReady)
	return &resp, game, nil
}

func (w *WsRequest) JoinPlayerToGame() (*md.Message, *md.Game, error) {
	var joinGameReq md.Message
	if err := json.Unmarshal(w.Payload, &joinGameReq); err != nil {
		return &md.Message{}, nil, err
	}
	log.Printf("unmarshaled join game payload: %+v\n", joinGameReq)

	correctedPayload, err := TypeAssertStringPayload(joinGameReq.Payload, md.KeyGameUuid)
	if err != nil {
		return &md.Message{}, nil, err
	}
	game := w.Server.FindGame(correctedPayload[md.KeyGameUuid])
	if game == nil {
		return &md.Message{}, nil, cerr.ErrorGameNotExists(correctedPayload[md.KeyGameUuid])
	}
	log.Printf("found game in database: %+v\n", game)

	game.AddJoinPlayer(w.Ws)
	resp := md.NewMessage(md.CodeRespSuccessJoinGame, md.WithPayload(md.RespJoinGame{PlayerUuid: game.JoinPlayer.Uuid}))

	return &resp, game, nil
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
