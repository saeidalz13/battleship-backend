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
	CodeSignalAbsent // if the req msg does not contain "code" field
	CodeOtherPlayerDisconnected
	CodeOtherPlayerReconnected
	CodeOtherPlayerGracePeriod
	CodeReconnectionSessionInfo
	// CodeRequestRematchFromServer
	// CodeRequestRematchFromOtherPlayer

	// // This code is sent from both players if they want rematch
	// CodeRematch
)

type Signal struct {
	Code int `json:"code"`
}

func NewSignal(code int) Signal {
	return Signal{Code: code}
}
