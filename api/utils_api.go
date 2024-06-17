package api

import (
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ConnLoopCodeBreak int = iota
	ConnLoopCodeContinue
	ConnLoopCodePassThrough
	ConnLoopCodeRetry
	ConnLoopAbnormalClosureRetry
)

func IdentifyWsConnErrAction(err error) int {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		log.Println("timeout error:", err)
		return ConnLoopCodeRetry
	}

	if websocket.IsCloseError(err, websocket.CloseTryAgainLater) {
		log.Println("high server load/traffic error:", err)
		return ConnLoopCodeRetry
	}

	// Happens if the IOS client goes to background
	if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
		log.Println("abnormal closure error:", err)
		return ConnLoopAbnormalClosureRetry
	}

	if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
		log.Println("close error:", err)
		return ConnLoopCodeBreak
	}

	if websocket.IsCloseError(err, websocket.CloseProtocolError, websocket.CloseInternalServerErr, websocket.CloseTLSHandshake, websocket.CloseMandatoryExtension) {
		log.Println("critical error:", err)
		return ConnLoopCodeBreak
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
		return ConnLoopCodeBreak
	}

	log.Println("unexpected error:", err)
	return ConnLoopCodeBreak
}

func WriteJSONWithRetry(conn *websocket.Conn, resp interface{}) int {
	var retries int

writeJsonLoop:
	for {
		if err := conn.WriteJSON(resp); err != nil {
			switch IdentifyWsConnErrAction(err) {
			case ConnLoopCodeRetry:
				if retries < maxWriteWsRetries {
					retries++
					log.Printf("writing json failed to ws [%s]; retrying... (retry no. %d)\n", conn.RemoteAddr().String(), retries)
					time.Sleep(time.Duration(retries*backOffFactor) * time.Second)
					continue writeJsonLoop

				} else {
					log.Printf("max retries reached for writing to ws [%s]:%s", conn.RemoteAddr().String(), err)
					return ConnLoopCodeBreak
				}

			case ConnLoopAbnormalClosureRetry:
				return ConnLoopAbnormalClosureRetry

			case ConnLoopCodeBreak:
				log.Println("breaking writeJsonLoop due to:", err)
				return ConnLoopCodeBreak
			}

			// Successful write and continue the main ws loop
		} else {
			return ConnLoopCodePassThrough
		}
	}
}
