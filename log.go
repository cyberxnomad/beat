package cron

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	rfc3339mil = "2006-01-02T15:04:05.000Z07:00"
)

type Logger interface {
	Debug(keyvals ...any)
	Info(keyvals ...any)
	Warn(keyvals ...any)
	Error(keyvals ...any)
}

var defaultLogger Logger = &logger{
	writer: os.Stdout,
	pool: &sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	},
}

type logger struct {
	writer io.Writer
	lock   sync.Mutex
	pool   *sync.Pool
}

func (l *logger) log(level string, keyvals ...any) {
	if len(keyvals)&1 == 1 {
		keyvals = append(keyvals, "!UNPAIRED")
	}

	buf := l.pool.Get().(*bytes.Buffer)
	defer l.pool.Put(buf)

	buf.WriteString(level)
	buf.WriteByte(' ')

	buf.WriteString("ts=")
	buf.WriteString(time.Now().Format(rfc3339mil))

	for i := 0; i < len(keyvals); i += 2 {
		fmt.Fprintf(buf, " %v=%v", keyvals[i], keyvals[i+1])
	}
	buf.WriteByte('\n')
	defer buf.Reset()

	l.lock.Lock()
	defer l.lock.Unlock()

	l.writer.Write(buf.Bytes())
}

func (l *logger) Debug(keyvals ...any) {
	l.log("DEBUG", keyvals...)
}

func (l *logger) Info(keyvals ...any) {
	l.log("INFO", keyvals...)
}

func (l *logger) Warn(keyvals ...any) {
	l.log("WARN", keyvals...)
}

func (l *logger) Error(keyvals ...any) {
	l.log("ERROR", keyvals...)
}
