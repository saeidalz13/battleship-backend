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

const (
	maxWriteWsRetries int = 3
	backOffFactor     int = 2
)

var (
	defaultPort int = 8000
	// allowedOrigins     = map[string]bool{
	// 	"https://www.allowed_url.com": true,
	// }
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
	mu      sync.RWMutex
	endGame chan string
	Games   map[string]*md.Game
	Players map[string]*md.Player
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

	game.CreateHostPlayer(ws, game)
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

	game.CreateJoinPlayer(ws, game)
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
	return game, nil
}

func (s *Server) FindPlayer(playerUuid string) (*md.Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	player, prs := s.Players[playerUuid]
	if !prs {
		return nil, cerr.ErrPlayerNotExist(playerUuid)
	}
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

	// if server.stage == StageProd {
	// 	upgrader.CheckOrigin = func(r *http.Request) bool {
	// 		origin := r.Header.Get("Origin")
	// 		return allowedOrigins[origin]
	// 	}
	// 	return nil
	// }

	server.endGame = make(chan string)
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
		s.removePlayer(ws.RemoteAddr().String())
		ws.Close()
		log.Println("connection closed:", ws.RemoteAddr().String())
	}()

wsLoop:
	for {
		// A WebSocket frame can be one of 6 types: text=1, binary=2, ping=9, pong=10, close=8 and continuation=0
		// https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
		retries := 0
		_, payload, err := ws.ReadMessage()
		if err != nil {
			switch IdentifyWsErrorAction(err) {
			case RetryWriteConn:
				if retries < maxWriteWsRetries {
					retries++
					log.Printf("failed to read from ws conn [%s]; retrying... (retry no. %d)\n", ws.RemoteAddr().String(), retries)
					time.Sleep(time.Duration(retries*backOffFactor) * time.Second)
					continue wsLoop

				} else {
					break wsLoop
				}

			case BreakLoop:
				log.Printf("break ws conn loop [%s] due to: %s\n", ws.RemoteAddr().String(), err)
				break wsLoop

			case ContinueLoop:
				continue wsLoop
			}
		}

		// the incoming message must be of type json containing the field "code"
		// which would allow us to determine what action is required
		// In case of absence of "code" field, the message is invalid
		var signal md.Signal
		if err := json.Unmarshal(payload, &signal); err != nil {
			log.Println("incoming msg does not contain 'code':", err)
			resp := md.NewMessage[md.NoPayload](md.CodeSignalAbsent)
			resp.AddError("incoming req payload must contain 'code' field", "")

			switch WriteJsonWithRetry(ws, resp) {
			case BreakLoop:
				break wsLoop
			default:
				continue wsLoop
			}
		}

		// This is where we choose the action based on the code in incoming json
		switch signal.Code {

		case md.CodeCreateGame:
			req := NewRequest(s, ws)
			resp := req.HandleCreateGame()

			switch WriteJsonWithRetry(ws, resp) {
			case BreakLoop:
				break wsLoop
			default:
				continue wsLoop
			}

		case md.CodeAttack:
			req := NewRequest(s, nil, payload)
			// response will have the IsTurn field of attacker
			resp, defender, game := req.HandleAttack()

			if resp.Error.ErrorDetails != "" {
				switch WriteJsonWithRetry(ws, resp) {
				case BreakLoop:
					break wsLoop
				default:
					continue wsLoop
				}
			}

			// attacker turn is set to false
			resp.Payload.IsTurn = false
			switch WriteJsonWithRetry(ws, resp) {
			case BreakLoop:
				break wsLoop
			case ContinueLoop:
				continue wsLoop
			case PassThrough:
			}

			// defender turn is set to true
			resp.Payload.IsTurn = true
			switch WriteJsonWithRetry(defender.WsConn, resp) {
			case BreakLoop:
				break wsLoop
			case ContinueLoop:
				continue wsLoop
			case PassThrough:
			}

			// If this attack caused the game to end.
			// Both attacker and defender will get a end game
			// message indicating if they lost or won
			if defender.MatchStatus == md.PlayerMatchStatusLost {
				// Sending victory code to the attacker
				respAttacker := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respAttacker.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusWon})
				switch WriteJsonWithRetry(ws, respAttacker) {
				case BreakLoop:
					break wsLoop
				case ContinueLoop:
					continue wsLoop
				case PassThrough:
				}

				// Sending failure code to the defender
				respDefender := md.NewMessage[md.RespEndGame](md.CodeEndGame)
				respDefender.AddPayload(md.RespEndGame{PlayerMatchStatus: md.PlayerMatchStatusLost})
				switch WriteJsonWithRetry(defender.WsConn, respDefender) {
				case BreakLoop:
					break wsLoop
				case ContinueLoop:
					continue wsLoop
				case PassThrough:
				}

				// For now I put this here:
				// Sending signal to channel that game is over
				s.endGame <- game.Uuid
			}

		case md.CodeReady:
			req := NewRequest(s, nil, payload)
			resp, game := req.HandleReadyPlayer()

			if resp.Error.ErrorDetails != "" {
				switch WriteJsonWithRetry(ws, resp) {
				case BreakLoop:
					break wsLoop
				default:
					continue wsLoop
				}
			}

			switch WriteJsonWithRetry(ws, resp) {
			case BreakLoop:
				break wsLoop
			case ContinueLoop:
				continue wsLoop
			case PassThrough:
			}

			if game.HostPlayer.IsReady && game.JoinPlayer.IsReady {
				respStartGame := md.NewMessage[md.NoPayload](md.CodeStartGame)
				switch SendMsgToBothPlayers(game, &respStartGame, &respStartGame) {
				case BreakLoop:
					break wsLoop
				case PassThrough:
				}
			}

		case md.CodeJoinGame:
			req := NewRequest(s, ws, payload)
			resp, game := req.HandleJoinPlayer()

			switch WriteJsonWithRetry(ws, resp) {
			case BreakLoop:
				break wsLoop
			case ContinueLoop:
				continue wsLoop
			case PassThrough:
			}

			// If the second playerd joined successfully, then `CodeSelectGrid`
			// is sent to both players as an indication of grid selection
			if resp.Error.ErrorDetails == "" {
				readyResp := md.NewMessage[md.NoPayload](md.CodeSelectGrid)
				switch SendMsgToBothPlayers(game, &readyResp, &readyResp) {
				case BreakLoop:
					break wsLoop
				case PassThrough:
				}
			}

		default:
			respInvalidSignal := md.NewMessage[md.NoPayload](md.CodeInvalidSignal)
			respInvalidSignal.AddError("", "invalid code in the incoming payload")
			switch WriteJsonWithRetry(ws, respInvalidSignal) {
			case BreakLoop:
				break wsLoop
			default:
				continue wsLoop
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

func (s *Server) ManageGames() {
	for {
		gameUuid := <-s.endGame
		game, err := s.FindGame(gameUuid)
		// err means the game was not found.
		if err != nil {
			continue
		}

		// Find all the players associated with the game
		// then delete both players and game from map
		players := game.GetPlayers()
		for _, player := range players {
			delete(s.Players, player.Uuid)
		}
		delete(s.Games, gameUuid)
		log.Printf("deleted game %s and its associated players\n", gameUuid)
	}
}

// Remove the player from Players map.
// For now this is O(n).
func (s *Server) removePlayer(remoteAddr string) {
	for _, player := range s.Players {
		if remoteAddr == player.WsConn.RemoteAddr().String() {

			// If the player gets removed, then the other player
			// needs to be notified that the game has ended.
			// For now this logic, can change based on what we want.
			var otherPlayer *md.Player
			if player.IsHost {
				player.CurrentGame.HostPlayer = nil
				otherPlayer = player.CurrentGame.JoinPlayer
			} else {
				player.CurrentGame.JoinPlayer = nil
				otherPlayer = player.CurrentGame.HostPlayer
			}
			
			respAttacker := md.NewMessage[md.NoPayload](md.CodeOtherPlayerDisconnected)
			// Whatever happens to writing to this connection, next steps
			// must happen. Ignoring the outcome
			_ = WriteJsonWithRetry(otherPlayer.WsConn, respAttacker) 

			delete(s.Players, player.Uuid)
			log.Printf("removed player from map: %s\n", player.Uuid)
			return
		}
	}
}
