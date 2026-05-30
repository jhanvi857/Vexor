package observability

import (
	"io"
	"log"
	"os"
	"sync"
)

type Logger struct {
	mu sync.Mutex
	l  *log.Logger
}

func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{l: log.New(out, prefix, flag)}
}

func (lg *Logger) Info(v ...any) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.l.SetPrefix("INFO: ")
	lg.l.Println(v...)
}

func (lg *Logger) Error(v ...any) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.l.SetPrefix("ERROR: ")
	lg.l.Println(v...)
}

func (lg *Logger) Debug(v ...any) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.l.SetPrefix("DEBUG: ")
	lg.l.Println(v...)
}

func (lg *Logger) Printf(format string, v ...any) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.l.Printf(format, v...)
}

func (lg *Logger) SetOutput(out io.Writer) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.l.SetOutput(out)
}
