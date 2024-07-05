package api

type SessionMessage struct {
	SenderSession *Session
	ReceiverID    string
	GameUuid      string
	Payload       interface{}
}

func NewSessionMessage(senderSession *Session, receiverId string, gameUuid string, p interface{}) SessionMessage {
	return SessionMessage{
		ReceiverID: receiverId,
		GameUuid:   gameUuid,
		Payload:    p,
	}
}
