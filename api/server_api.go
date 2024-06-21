package api

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	md "github.com/saeidalz13/battleship-backend/models"
)

const (
	StageProd = "prod"
	StageDev  = "dev"
)

const (
	maxWriteWsRetries       int           = 2
	backOffFactor           int           = 2
	maxTimeGame             time.Duration = time.Minute * 30
	connHealthCheckInterval time.Duration = time.Second * 45

	URLQuerySessionIDKeyword string = "sessionID"
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
	port           *int
	stage          string
	GameManager    *GameManager
	SessionManager *SessionManager
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

	server.SessionManager = NewSessionManager()
	server.GameManager = NewGameManager()

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

func (s *Server) HandleWs(w http.ResponseWriter, r *http.Request) {
	// use Upgrade method to make a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		return
	}

	sessionIdQuery := r.URL.Query().Get(URLQuerySessionIDKeyword)
	switch sessionIdQuery {
	case "":
		// creating a new URL compatible session ID
		newSessionIdRaw := uuid.New().String()
		sessionIdUrlCompatible := base64.RawURLEncoding.EncodeToString([]byte(newSessionIdRaw))

		session := NewSession(conn, sessionIdUrlCompatible, s.GameManager, s.SessionManager)
		s.SessionManager.Sessions[sessionIdUrlCompatible] = session

		resp := md.NewMessage[md.RespSessionId](md.CodeSessionID)
		resp.AddPayload(md.RespSessionId{SessionID: sessionIdUrlCompatible})
		_ = conn.WriteJSON(resp)

		log.Println("a new connection established\tRemote Addr: ", conn.RemoteAddr().String())
		go session.run()

	default:
		session, prs := s.SessionManager.Sessions[sessionIdQuery]
		if !prs {
			// This either means an expired session or invalid session ID
			conn.WriteJSON(md.NewMessage[md.NoPayload](md.CodeReceivedInvalidSessionID))
			conn.Close()
			return
		}
		// Signal for reconnection
		close(session.StopRetry)
		session.GraceTimer.Stop()

		// Setting the new fields for the session
		session.Conn = conn
		session.StopRetry = make(chan struct{})

		log.Printf("session %s reconnected\n", session.ID)
	}
}
