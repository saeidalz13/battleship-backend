package connection

import "fmt"

const (
	ConnLoopBreak uint8 = iota
	ConnLoopRetry
	ConnLoopAbnormalClosureRetry
	ConnLoopContinue
	ConnLoopPassThrough
	ConnInvalidMsgType
)

type ConnErr struct {
	code uint8
	desc string
}

func NewConnErr(code uint8) ConnErr {
	return ConnErr{code: code}
}

func (c ConnErr) AddDesc(desc string) ConnErr {
	c.desc = desc
	return c
}

func (c ConnErr) Error() string {
	return fmt.Sprintf("Connection error - Code: %d\tdesc: %s\n", c.code, c.desc)
}

func (c ConnErr) Code() uint8 {
	return c.code
}
