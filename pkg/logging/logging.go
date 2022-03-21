package logging

import (
	"fmt"
	"log"
	"sync"

	ct "github.com/seago/go-colortext"
)

type Logger struct {
	mu    sync.Mutex // ensures atomic writes; protects the following fields
	level int        // One of DEBUG, ERROR, INFO
}

var std *Logger

const (
	DEBUG = 1 << iota
	INFO
	ERROR
)

func init() {
	std = &Logger{}
}

func (logger Logger) Output(lvl int, text string) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	log.Println(text)
}
func Error(v ...interface{}) {
	if std.level <= ERROR {
		ct.ChangeColor(ct.Red, true, ct.None, false)
		s := fmt.Sprintf("ERROR: %v", v...)
		std.Output(2, s)
		ct.ResetColor()
	}
}

func Info(format string, v ...interface{}) {
	if std.level <= INFO {
		s := fmt.Sprintf("INFO: "+format, v...)
		std.Output(2, s)
	}
}

func Debug(v ...interface{}) {
	if std.level <= DEBUG {
		s := fmt.Sprintf("DEBUG: %v", v...)
		std.Output(2, s)
	}
}

func SetLogLevel(lvl int) {
	std.level = lvl
}
