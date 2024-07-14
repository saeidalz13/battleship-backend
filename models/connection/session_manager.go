package connection

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	cerr "github.com/saeidalz13/battleship-backend/internal/error"
	mb "github.com/saeidalz13/battleship-backend/models/battleship"
)

type SessionManager interface {
	GenerateNewSession(conn *websocket.Conn) *Session
	CleanupPeriodically()

	FindSession(sessionId string) (*Session, error)
	TerminateSession(session *Session)
	ReconnectSession(session *Session, conn *websocket.Conn)
	Communicate(receiverSessionId string, msg interface{}, msgType uint8) error
	HandleAbnormalClosureSession(session *Session) error
	GetSessionId(session *Session) string

	GetSessionGame(session *Session) *mb.Game
	GetSessionPlayer(session *Session) *mb.BattleshipPlayer

	SetSessionGame(session *Session, game *mb.Game)
	SetSessionPlayer(session *Session, player *mb.BattleshipPlayer)
}

type BattleshipSessionManager struct {
	cleanupInterval time.Duration
	sessions        map[string]*Session
	mu              sync.RWMutex
}

func NewBattleshipSessionManager() *BattleshipSessionManager {
	initMapSize := 10

	return &BattleshipSessionManager{
		sessions:        make(map[string]*Session, initMapSize),
		cleanupInterval: time.Minute * 20,
	}
}

var _ SessionManager = (*BattleshipSessionManager)(nil)

func (bsm *BattleshipSessionManager) GetSessionGame(session *Session) *mb.Game {
	return session.game
}

func (bsm *BattleshipSessionManager) SetSessionGame(session *Session, game *mb.Game) {
	session.game = game
}

func (bsm *BattleshipSessionManager) GetSessionPlayer(session *Session) *mb.BattleshipPlayer {
	return session.player
}

func (bsm *BattleshipSessionManager) SetSessionPlayer(session *Session, player *mb.BattleshipPlayer) {
	session.player = player
}

func (bsm *BattleshipSessionManager) GenerateNewSession(conn *websocket.Conn) *Session {
	sessionId := base64.RawURLEncoding.EncodeToString([]byte(uuid.New().String()))
	bsm.sessions[sessionId] = NewSession(sessionId, conn)

	return bsm.sessions[sessionId]
}

func (bsm *BattleshipSessionManager) FindSession(sessionId string) (*Session, error) {
	bsm.mu.RLock()
	defer bsm.mu.RUnlock()

	session, prs := bsm.sessions[sessionId]
	if !prs {
		return nil, cerr.ErrSessionNotFound(sessionId)
	}

	if session == nil {
		return nil, cerr.ErrSessionIsNil(sessionId)
	}

	return session, nil
}

func (bsm *BattleshipSessionManager) TerminateSession(session *Session) {
	// _ = session.conn.Close()
	delete(bsm.sessions, session.id)
}

func (bsm *BattleshipSessionManager) ReconnectSession(session *Session, conn *websocket.Conn) {
	session.reconnectionAfterAbnormalClosure(conn)
}

// This method sends the msg from one session to another
func (bsm *BattleshipSessionManager) Communicate(receiverSessionId string, msg interface{}, msgType uint8) error {
	receiverSession, err := bsm.FindSession(receiverSessionId)
	if err != nil {
		return err
	}
	return bsm.WriteToSessionConn(receiverSession, msg, msgType)
}

// To ensure that there is no dangling connections,
// server session manager marks the connections with a
// lifetime of more than 20 mins as stale and deletes them.
func (bsm *BattleshipSessionManager) CleanupPeriodically() {
	assumedClosedConns := 10

	for {
		time.Sleep(bsm.cleanupInterval)

		bsm.mu.Lock()
		toDelete := make([]string, 0, assumedClosedConns)

		for ID, session := range bsm.sessions {
			if time.Since(session.createdAt) > bsm.cleanupInterval {
				toDelete = append(toDelete, ID)
			}
		}

		log.Println("Clean up sessions:")
		for _, ID := range toDelete {
			delete(bsm.sessions, ID)
			log.Printf("removed: %s", ID)
		}
		bsm.mu.Unlock()
	}
}

// This function takes care of abnormal closures happening
// to either of the clients. This happens due to backgrounding
// in IOS clients or any other unexpected reasons for web apps.
func (bsm *BattleshipSessionManager) HandleAbnormalClosureSession(s *Session) error {
	// This means there is no game and abnormal closure is happening
	// which means this session is invalid and should end
	if s.game == nil || s.player == nil {
		return NewConnErr(ConnLoopBreak).AddDesc("game or player is nil")
	}

	otherPlayer := s.game.GetOtherPlayer(s.player)
	if otherPlayer == nil {
		return NewConnErr(ConnLoopBreak).AddDesc("othre player is nil; invalid session")
	}

	// Absence of otherPlayer session means this game is invalid
	otherSession, err := bsm.FindSession(otherPlayer.GetSessionId())
	if err != nil {
		return NewConnErr(ConnLoopBreak).AddDesc("other session is nil; invalid session")
	}

	// If the other session connection is faulty too, there is no need to continue
	if err := otherSession.writeToConnWithRetry(NewMessage[NoPayload](CodeOtherPlayerGracePeriod), MessageTypeJSON); err != nil {
		return err
	}

	timer := time.NewTimer(gracePeriod)
	select {
	case <-timer.C:
		if otherSession != nil {
			if err := otherSession.writeToConnWithRetry(NewMessage[NoPayload](CodeOtherPlayerDisconnected), MessageTypeJSON); err != nil {
				return err
			}
		}

		log.Printf("session terminated: %s\n", s.id)
		return NewConnErr(ConnLoopBreak).AddDesc("grace period is over for session: %d" + s.id)

	case <-s.reconnectionSignalChan:
		if otherSession != nil {
			if err := otherSession.writeToConnWithRetry(NewMessage[NoPayload](CodeOtherPlayerReconnected), MessageTypeJSON); err != nil {
				return err
			}
		}
		log.Printf("player reconnected, session: %s\n", s.id)
		return nil
	}
}

func (bsm *BattleshipSessionManager) WriteToSessionConn(session *Session, msg interface{}, msgType uint8) error {
	err := session.writeToConnWithRetry(msg, msgType)

	if err != nil {
		connErr, ok := err.(ConnErr)
		if !ok {
			panic("this will never happen")
		}

		switch connErr.Code() {
		case ConnLoopBreak, ConnInvalidMsgType:
			return connErr

		case ConnLoopAbnormalClosureRetry:
			if err := bsm.HandleAbnormalClosureSession(session); err != nil {
				return connErr
			}
		}
	}

	return nil
}

func (bsm *BattleshipSessionManager) ReadFromSessionConn(session *Session) (int, []byte, error) {
	var retries uint8

	for {
		messageType, payload, err := session.conn.ReadMessage()
		if err == nil {
			return messageType, payload, nil
		}

		switch session.handleReadFromConnErr(err, retries) {
		case ConnLoopContinue:
			retries++
			continue

		case ConnLoopAbnormalClosureRetry:
			if err := bsm.HandleAbnormalClosureSession(session); err != nil {
				return -1, []byte{}, err
			}

		default:
			return -1, []byte{}, err
		}
	}
}

func (bsm *BattleshipSessionManager) GetSessionId(session *Session) string {
	return session.id
}

func (bsm *BattleshipSessionManager) FetchCodeFromMsg(session *Session, payload []byte) (uint8, error) {
	var signal Signal
	const randomInvalidCode uint8 = 255

	if err := json.Unmarshal(payload, &signal); err != nil {
		return randomInvalidCode, err
	}

	return signal.Code, nil
}
