package connection

import (
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxWriteWsRetries uint8         = 2
	backOffFactor     uint8         = 2
	gracePeriod       time.Duration = time.Minute * 2
)

const (
	MessageTypeBytes uint8 = iota
	MessageTypeJSON
)

type ConnectionHandler interface {
	reconnectionAfterAbnormalClosure(conn *websocket.Conn)
	handleReadFromConnErr(err error, retries uint8) uint8
	writeToConnWithRetry(msg interface{}, msgType uint8) error
	onConnErr(err error) uint8
}

type Session struct {
	id                     string
	conn                   *websocket.Conn
	reconnectionSignalChan chan bool
	createdAt              time.Time
}

func NewSession(id string, conn *websocket.Conn) *Session {
	return &Session{
		id:                     id,
		conn:                   conn,
		reconnectionSignalChan: make(chan bool),
		createdAt:              time.Now(),
	}
}

func (s *Session) Id() string {
	return s.id
}

func (s *Session) Conn() *websocket.Conn {
	return s.conn
}

func (s *Session) onConnErr(err error) uint8 {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		log.Println("timeout error:", err)
		return ConnLoopRetry
	}

	if websocket.IsCloseError(err, websocket.CloseTryAgainLater) {
		log.Println("high server load/traffic error:", err)
		return ConnLoopRetry
	}

	// Happens if the IOS client goes to background
	if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
		log.Println("abnormal closure error:", err)
		return ConnLoopAbnormalClosureRetry
	}

	if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
		log.Println("close error:", err)
		return ConnLoopBreak
	}

	if websocket.IsCloseError(err, websocket.CloseProtocolError, websocket.CloseInternalServerErr, websocket.CloseTLSHandshake, websocket.CloseMandatoryExtension) {
		log.Println("critical error:", err)
		return ConnLoopBreak
	}

	/*
		This might mean that the client is not from the application.
		Breaking not to overwhelm the server with invalid payloads (e.g. binary data)

		CloseUnsupportedData (1003):
		- Client sends a binary message to a server that only supports text messages.
		- Server closes the connection with CloseUnsupportedData because it cannot handle binary data.

		CloseInvalidFramePayloadData (1007):
		- Client sends a text message with a payload that is not properly encoded as UTF-8.
		- Server attempts to decode the text message but fails due to invalid encoding.
		- Server closes the connection with CloseInvalidFramePayloadData because the payload data is invalid.
	*/
	if websocket.IsCloseError(err, websocket.CloseInvalidFramePayloadData, websocket.CloseUnsupportedData, websocket.CloseMessageTooBig, websocket.ClosePolicyViolation, websocket.CloseServiceRestart, websocket.CloseNoStatusReceived) {
		log.Println("non-critical error:", err)
		return ConnLoopBreak
	}

	log.Println("unexpected error:", err)
	return ConnLoopBreak
}

// Writes to the connection of that session. It also
// handles the abnormal or other types of errors of
// writing to a websocket connection.
func (s *Session) writeToConnWithRetry(msg interface{}, msgType uint8) error {
	var retries uint8

writeJsonLoop:
	for {
		var err error

		switch msgType {
		case MessageTypeJSON:
			err = s.conn.WriteJSON(msg)

		case MessageTypeBytes:
			respBytes, ok := msg.([]byte)
			if ok {
				err = s.conn.WriteMessage(websocket.TextMessage, respBytes)
			} else {
				return NewConnErr(ConnInvalidMsgType).AddDesc("msg type expected: []byte got invalid")
			}

		default:
			return NewConnErr(ConnInvalidMsgType).AddDesc("invalid meessage type to write with retry")
		}

		if err != nil {
			switch s.onConnErr(err) {
			case ConnLoopRetry:
				if retries < maxWriteWsRetries {
					retries++
					log.Printf("writing json failed to ws [%s]; retrying... (retry no. %d)\n", s.conn.RemoteAddr().String(), retries)
					time.Sleep(time.Duration(retries*backOffFactor) * time.Second)
					continue writeJsonLoop

				} else {
					log.Printf("max retries reached for writing to ws [%s]:%s", s.conn.RemoteAddr().String(), err)
					return NewConnErr(ConnLoopBreak)
				}

			case ConnLoopAbnormalClosureRetry:
				return NewConnErr(ConnLoopAbnormalClosureRetry)

			case ConnLoopBreak:
				return NewConnErr(ConnLoopBreak).AddDesc("breaking writeJsonLoop due to:" + err.Error())
			}
		}
		return nil
	}
}

// Handles the errors that occurs when reading from
// ws connection. `ConnLoopCodeContinue` will results in
// terminating the session and removing `run` from stack
func (s *Session) handleReadFromConnErr(err error, retries uint8) uint8 {
	switch s.onConnErr(err) {
	case ConnLoopAbnormalClosureRetry:
		return ConnLoopAbnormalClosureRetry

	case ConnLoopRetry:
		if retries < maxWriteWsRetries {
			log.Printf("failed to read from ws conn [%s]; retrying... (retry no. %d)\n", s.conn.RemoteAddr().String(), retries)
			time.Sleep(time.Duration(retries*backOffFactor) * time.Second)
			return ConnLoopContinue

		} else {
			return ConnLoopBreak
		}

	case ConnLoopBreak:
		log.Printf("break ws conn loop [%s] due to: %s\n", s.conn.RemoteAddr().String(), err)
		return ConnLoopBreak

		// will never reach this
	default:
		return ConnLoopBreak
	}
}

func (s *Session) reconnectionAfterAbnormalClosure(conn *websocket.Conn) {
	// Signal for reconnection
	close(s.reconnectionSignalChan)

	// Setting the new fields for the session
	s.conn = conn
	s.reconnectionSignalChan = make(chan bool)
}

var _ ConnectionHandler = (*Session)(nil)
