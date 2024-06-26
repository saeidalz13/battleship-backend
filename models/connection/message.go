package connection

type NoPayload bool
type Message[T any] struct {
	Code    int     `json:"code"`
	Payload T       `json:"payload,omitempty"`
	Error   *RespErr `json:"error,omitempty"`
}

func NewMessage[T any](code int) Message[T] {
	return Message[T]{Code: code}
}

func (m *Message[T]) AddPayload(payload T) {
	m.Payload = payload
}

func (m *Message[T]) AddError(errorDetails, message string) {
	m.Error = NewRespErr(errorDetails, message)
}
