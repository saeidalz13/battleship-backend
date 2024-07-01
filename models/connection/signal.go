package connection

const (
	CodeSessionID int = iota
	CodeReceivedInvalidSessionID
	CodeCreateGame
	CodeJoinGame
	CodeSelectGrid
	CodeReady
	CodeStartGame
	CodeAttack
	CodeEndGame
	CodeInvalidSignal

	// if the req msg does not contain "code" field
	CodeSignalAbsent

	CodeOtherPlayerDisconnected
	CodeOtherPlayerReconnected
	CodeOtherPlayerGracePeriod

	// Ask the server to message the other player
	// if they want a rematch too
	CodeRematchCall

	// Other player also wants a rematch
	// This code is sent from both players if they want rematch
	CodeRematchCallAccepted
	CodeRematchCallRejected
)

type Signal struct {
	Code int `json:"code"`
}

func NewSignal(code int) Signal {
	return Signal{Code: code}
}

/*
1. Player sends CodeRematchAskOtherPlayer to server
2. Server asks the other player with CodeRematchAskOtherPlayer code
3. Now we have 3 modes
	- Other Player got disconnected and no longer available
	- Other Player says YES to rematch
	- Other Player says NO to rematch

- In case of disconnection, break the loop and terminate the session

- In case of YES or NO, Other player sends CodeRematchOtherPlayerResponse of which
its payload indicates if the answer is NO or YES (0 or 1)
to server which means reseting Game and Player attributes. IOS Client
must clean up everything itself and go back to a mode similar to
what Create Game has.
*/
