package api

import (
	"log"
	"sync"
	"time"
)

var GlobalSessionManager = NewSessionManager()

const (
	// Assuming this capacity for the slice when
	// we're cleaning up the sessions map.
	assumedClosedConns               = 5
	cleanupInterval    time.Duration = time.Minute * 15
)

type SessionManager struct {
	Sessions          map[string]*Session
	CommunicationChan chan SessionMessage
	DeletionChan      chan string
	mu                sync.Mutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		Sessions:          make(map[string]*Session),
		CommunicationChan: make(chan SessionMessage),
		DeletionChan:      make(chan string),
	}
}

func (sm *SessionManager) ManageSessionsDeletion() {
	for {
		sessionId := <-sm.DeletionChan

		sm.mu.Lock()
		delete(sm.Sessions, sessionId)
		sm.mu.Unlock()

		log.Println("session closed:", sessionId)
	}
}

func (sm *SessionManager) ManageCommunication() {
	for {
		msg := <-sm.CommunicationChan

		sm.mu.Lock()
		receiverSession, prs := sm.Sessions[msg.ReceiverID]
		if !prs {
			// It should never be the case that the other session
			// is not found. The sender session should terminate
			msg.SenderSession.terminate()
			continue
		}

		if receiverSession.Game.Uuid != msg.GameUuid {
			panic("receiver session msg game is not the same as game uuid; this error should never happen")
		}

		switch WriteJSONWithRetry(receiverSession.Conn, msg.Payload) {
		case ConnLoopAbnormalClosureRetry:
			switch receiverSession.handleAbnormalClosure() {
			case ConnLoopCodeBreak:
				receiverSession.terminate()

			case ConnLoopCodeContinue:
			}

		case ConnLoopCodeBreak:
			receiverSession.terminate()

		case ConnLoopCodePassThrough:
		}

		sm.mu.Unlock()
	}
}

// To ensure that there is no dangling connections,
// server session manager marks the connections with a
// lifetime of more than 20 mins as stale and deletes them.
func (sm *SessionManager) CleanUpPeriodically() {
	for {
		time.Sleep(cleanupInterval)

		sm.mu.Lock()

		toDelete := make([]string, 0, assumedClosedConns)

		for ID, session := range sm.Sessions {
			if time.Since(session.CreatedAt) > cleanupInterval {
				toDelete = append(toDelete, ID)
			}
		}

		for _, ID := range toDelete {
			delete(sm.Sessions, ID)
		}

		sm.mu.Unlock()
	}
}
