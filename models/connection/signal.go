package connection

const (
	CodeSessionID uint8 = iota
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
	CodeRematch

	// Players can send template texts and emojis to each other
	CodePlayerInteraction
)

type Signal struct {
	Code uint8 `json:"code"`
}

func NewSignal(code uint8) Signal {
	return Signal{Code: code}
}
