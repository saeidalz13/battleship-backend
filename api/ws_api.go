package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	md "github.com/saeidalz13/battleship-backend/models"
)

const (
	StageProd = "prod"
	StageDev  = "dev"
)

var (
	defaultPort    int = 8000
	allowedOrigins     = map[string]bool{
		"https://www.allowed_url.com": true,
	}
	upgrader = websocket.Upgrader{

		// good average time since this is not a high-latency operation such as video streaming
		HandshakeTimeout: time.Second * 5,

		// probably more that enough but this is a good average size
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

type Server struct {
	port    *int
	stage   string
	Games   map[string]*md.Game
	Players map[string]*md.Player
	mu      sync.RWMutex
}

func (s *Server) AddGame() *md.Game {
	s.mu.Lock()
	defer s.mu.Unlock()

	newGame := md.NewGame()
	s.Games[newGame.Uuid] = newGame
	return newGame
}

func (s *Server) AddHostPlayer(game *md.Game, ws *websocket.Conn) *md.Player {
	s.mu.Lock()
	defer s.mu.Unlock()

	game.CreateHostPlayer(ws)
	s.Players[game.HostPlayer.Uuid] = game.HostPlayer
	return game.HostPlayer
}

func (s *Server) AddJoinPlayer(gameUuid string, ws *websocket.Conn) (*md.Game, error) {
	game, err := s.FindGame(gameUuid)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	game.CreateJoinPlayer(ws)
	s.Players[game.JoinPlayer.Uuid] = game.JoinPlayer
	return game, nil
}

func (s *Server) FindGame(gameUuid string) (*md.Game, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	game, prs := s.Games[gameUuid]
	if !prs {
		return nil, cerr.ErrGameNotExists(gameUuid)
	}
	log.Printf("game found: %s", gameUuid)
	return game, nil
}

func (s *Server) FindPlayer(playerUuid string) (*md.Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	player, prs := s.Players[playerUuid]
	if !prs {
		return nil, cerr.ErrPlayerNotExist(playerUuid)
	}
	log.Printf("player found: %s", playerUuid)
	return player, nil
}

type Option func(*Server) error

func NewServer(optFuncs ...Option) *Server {
	var server Server
	for _, opt := range optFuncs {
		if err := opt(&server); err != nil {
			panic(err)
		}
	}
	if server.port == nil {
		server.port = &defaultPort
	}
	server.Games = make(map[string]*md.Game)
	server.Players = make(map[string]*md.Player)

	if server.stage == StageProd {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return allowedOrigins[origin]
		}
		return nil
	}
	return &server
}

func WithPort(port int) Option {
	return func(s *Server) error {
		if port > 10000 {
			panic("choose a port less than 10000")
		}
    
		s.port = &port
		return nil
	}
}


func WithStage(stage string) Option {
	return func(s *Server) error {
		if stage != StageProd && stage != StageDev {
			return fmt.Errorf("invalid type of development stage: %s", stage)
		}
		s.stage = stage
		return nil
	}
}

func (s *Server) manageWsConn(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		log.Println("connection closed:", ws.RemoteAddr().String())
	}()

	for {
		// A WebSocket frame can be one of 6 types: text=1, binary=2, ping=9, pong=10, close=8 and continuation=0
		// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
		_, payload, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
			}
			// whatever else is not really an error. would be normal closure
			break
		}

		// the incoming message must be of type json containing the field "code"
		// which would allow us to determine what action is required
		// any other format of incoming message is invalid and will be ignored
		var signal md.Signal
		if err := json.Unmarshal(payload, &signal); err != nil {
			log.Println(err)
			if err := ws.WriteMessage(websocket.TextMessage, []byte("incoming message is invalid; must be of type json")); err != nil {
				log.Println(err)
				break
			}
			continue
		}

		// This is where we choose the action based on the code in incoming json
		switch signal.Code {

		case md.CodeCreateGame:
			// Finalized
			req := NewRequest(s, ws)
			resp, _ := req.HandleCreateGame()
			if err := ws.WriteJSON(resp); err != nil {
				log.Printf("failed to create new game: %v\n", err)
				continue
			}

		case md.CodeAttack:
			req := NewRequest(s, nil, payload)
			resp, defender := req.HandleAttack()

			if err := ws.WriteJSON(resp); err != nil {
				log.Println(err)
				continue
			}

			// If this attack caused the game to end
			// Both attacker and defender (host or join) will get a end game message
			// indicating if they lost or won
			if defender.MatchStatus == md.PlayerMatchStatusLost {

				// Sending victory code to the attacker
				respAttacker := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respAttacker.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusWon})
				if err := ws.WriteJSON(respAttacker); err != nil {
					log.Println(err)
					continue
				}

				// Sending failure code to the defender
				respDefender := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respDefender.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusLost})
				if err := defender.WsConn.WriteJSON(respDefender); err != nil {
					log.Println(err)
					continue
				}
			}

		case md.CodeReady:
			req := NewRequest(s, nil, payload)
			resp, _, err := req.HandleReadyPlayer()

			if err != nil {
				log.Printf("failed to make the player ready: %v\n", err)
				respErr := md.NewMessage[any](md.CodeReady)
				respErr.AddError(err.Error(), "failed to make the player ready")
				if err := ws.WriteJSON(respErr); err != nil {
					log.Println(err)
				}
				continue
			} else {
				if err := ws.WriteJSON(resp); err != nil {
					log.Println(err)
					continue
				}

				// respStartGame := md.NewMessage[any](md.CodeStartGame)
				// if game.HostPlayer.IsReady && game.JoinPlayer.IsReady {
				// 	if err := SendMsgToBothPlayers(game, &respStartGame, &respStartGame); err != nil {
				// 		log.Println(err)
				// 	}
				// 	continue
				// }
			}

		case md.CodeJoinGame:
			// Finalized
			req := NewRequest(s, ws, payload)
			resp, game, err := req.HandleJoinPlayer()
			if err != nil {
				log.Printf("failed to join player: %v\n", err)
				respErr := md.NewMessage[any](md.CodeJoinGame)
				respErr.AddError(err.Error(), "failed to join the player")
				if err := ws.WriteJSON(respErr); err != nil {
					log.Println(err)
				}

			} else {
				if err := ws.WriteJSON(resp); err != nil {
					log.Printf("failed to join player: %v\n", err)
				}
				readyResp := md.NewMessage[any](md.CodeSelectGrid)
				if err := SendMsgToBothPlayers(game, &readyResp, &readyResp); err != nil {
					log.Println(err)
					continue
				}
			}

		default:
			respInvalidSignal := md.NewMessage[md.NoPayload](md.CodeInvalidSignal)
			if err := ws.WriteJSON(respInvalidSignal); err != nil {
				log.Println(err)
				continue
			}
		}
	}
}

func (s *Server) HandleWs(w http.ResponseWriter, r *http.Request) {
	// use Upgrade method to make a websocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		return
	}

	log.Println("a new connection established!\tRemote Addr: ", ws.RemoteAddr().String())

	// managing connection on another goroutine
	go s.manageWsConn(ws)
}
