package connection

type NoPayload bool
type Message[T any] struct {
	Code    uint8      `json:"code"`
	Payload T        `json:"payload,omitempty"`
	Error   *RespErr `json:"error,omitempty"`
}

func NewMessage[T any](code uint8) Message[T] {
	return Message[T]{Code: code}
}

func (m *Message[T]) AddPayload(payload T) {
	m.Payload = payload
}

func (m *Message[T]) AddError(errorDetails, message string) {
	m.Error = NewRespErr(errorDetails, message)
}
