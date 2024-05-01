package models

const (
	CodeStartGame = iota
	CodeEndGame
	CodeAttack
	CodeReady
)

type SignalStruct struct {
	Code int `json:"code"`
}

type StartGameResp struct {
	GameUuid string
	HostUuid string
	HostGrid [][]int
}

// type IncomingMessage struct {
//   ActionSignal string
//   Payload      interface{}
// }
//
// type ActionSignalStruct struct {
// 	StartGame string
// 	EndGame   string
// }
//
// var ActionSignal = &ActionSignalStruct{
// 	StartGame: "0",
// 	EndGame:   "1",
// }
//
// type StatusSignalStruct struct {
// 	Success string
// 	Failure string
// }
//
// var StatusSignal = &StatusSignalStruct{
// 	Success: "0",
// 	Failure: "1",
// }

// type IncomingMessage struct {
// 	ActionSignal string `json:"action_signal"`
// 	Payload      interface{}
// }
