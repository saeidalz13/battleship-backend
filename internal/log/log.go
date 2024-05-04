package log

import "log"

func LogCustom(msg string, v interface{}) {
	log.Printf(msg+" %+v", v)
}
