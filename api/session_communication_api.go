package api

const (
	TypeSessionMessageBytes int8 = iota
	TypeSessionMessageJSON
)

type SessionMessage struct {
	PayloadType int8
	ReceiverID  string
	GameUuid    string
	Payload     interface{}
}

func NewSessionMessageJSON(receiverId string, gameUuid string, p interface{}) SessionMessage {
	return SessionMessage{
		PayloadType: TypeSessionMessageJSON,
		ReceiverID:  receiverId,
		GameUuid:    gameUuid,
		Payload:     p,
	}
}

func NewSessionMessageBytes(receiverId string, gameUuid string, p interface{}) SessionMessage {
	return SessionMessage{
		PayloadType: TypeSessionMessageBytes,
		ReceiverID:  receiverId,
		GameUuid:    gameUuid,
		Payload:     p,
	}
}
