package connection

type SessionMessage struct {
	PayloadType uint8
	ReceiverID  string
	GameUuid    string
	Payload     interface{}
}

func NewSessionMessageJSON(receiverId string, gameUuid string, p interface{}) SessionMessage {
	return SessionMessage{
		PayloadType: MessageTypeJSON,
		ReceiverID:  receiverId,
		GameUuid:    gameUuid,
		Payload:     p,
	}
}

func NewSessionMessageBytes(receiverId string, gameUuid string, p interface{}) SessionMessage {
	return SessionMessage{
		PayloadType: MessageTypeBytes,
		ReceiverID:  receiverId,
		GameUuid:    gameUuid,
		Payload:     p,
	}
}
