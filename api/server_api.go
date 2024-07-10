package api

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	mc "github.com/saeidalz13/battleship-backend/models/connection"
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
	defaultPort string = "8000"
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
	port           string
	stage          string
	Db             *sql.DB
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
	if server.port == "" {
		server.port = defaultPort
	}

	server.SessionManager = NewSessionManager()
	server.GameManager = NewGameManager()

	return &server
}

func WithPort(port string) Option {
	return func(s *Server) error {
		s.port = port
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

func WithDb(db *sql.DB) Option {
	return func(s *Server) error {
		s.Db = db
		return nil
	}
}

func (s *Server) getServerIpNet(localAddr string) (net.IPNet, error) {
	host, _, err := net.SplitHostPort(localAddr)
	if err != nil {
		log.Println("failed to extract host from local addr")
		return net.IPNet{}, err
	}

	parsedIP := net.ParseIP(host)
	log.Println(parsedIP)

	return net.IPNet{
		IP:   parsedIP,
		Mask: net.CIDRMask(32, 32),
	}, nil
}

func (s *Server) HandleWs(w http.ResponseWriter, r *http.Request) {
	// use Upgrade method to make a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "could not open websocket connection", http.StatusBadRequest)
		return
	}

	serverIpNet, err := s.getServerIpNet(conn.LocalAddr().String())
	if err != nil {
		panic(err)
	}

	sessionIdQuery := r.URL.Query().Get(URLQuerySessionIDKeyword)
	switch sessionIdQuery {
	case "":
		// creating a new URL compatible session ID
		newSessionIdRaw := uuid.New().String()
		sessionIdUrlCompatible := base64.RawURLEncoding.EncodeToString([]byte(newSessionIdRaw))

		session := NewSession(conn, sessionIdUrlCompatible, s.GameManager, s.SessionManager, serverIpNet, s.Db)
		s.SessionManager.Sessions[sessionIdUrlCompatible] = session

		resp := mc.NewMessage[mc.RespSessionId](mc.CodeSessionID)
		resp.AddPayload(mc.RespSessionId{SessionID: sessionIdUrlCompatible})
		_ = conn.WriteJSON(resp)

		log.Println("a new connection established\tRemote Addr: ", conn.RemoteAddr().String())
		go session.run()

	default:
		session, prs := s.SessionManager.Sessions[sessionIdQuery]
		if !prs {
			// This either means an expired session or invalid session ID
			conn.WriteJSON(mc.NewMessage[mc.NoPayload](mc.CodeReceivedInvalidSessionID))
			conn.Close()
			return
		}

		// Signal for reconnection
		close(session.StopRetry)

		// Setting the new fields for the session
		session.Conn = conn
		session.StopRetry = make(chan struct{})

		/*
			we discussed that if app total closure or crash happens
			it is not the server's fault. Hence, the server doesn not need
			to provide the session information upon reconnection
			Send the session data to update client information
		*/

		// game, err := session.GameManager.FindGame(session.GameUuid)
		// if err != nil {
		// 	// This either means an expired session or invalid session ID
		// 	conn.WriteJSON(mc.NewMessage[mc.NoPayload](mc.CodeReceivedInvalidSessionID))
		// 	conn.Close()
		// 	return
		// }
		// msg := mc.NewMessage[mc.RespReconnect](mc.CodeReconnectionSessionInfo)
		// msg.AddPayload(mc.NewRespReconnect(session.Player, game))
		// _ = session.Conn.WriteJSON(msg)
	}
}
