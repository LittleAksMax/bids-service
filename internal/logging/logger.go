package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const timestampFormat = "2006/01/02 15:04:05"

type Logger struct {
	component string
	base      *log.Logger
	closers   []io.Closer
	closeOnce sync.Once
}

func NewReceiverLogger(writers ...io.Writer) *Logger {
	return newLogger("[Receiver]", writers...)
}

func NewProcessorLogger(id int, writers ...io.Writer) *Logger {
	return newLogger(fmt.Sprintf("[Processor %d]", id), writers...)
}

func newLogger(component string, writers ...io.Writer) *Logger {
	closers := make([]io.Closer, 0, len(writers))
	for _, writer := range writers {
		// Extract out Closer from Writer object
		closer, ok := writer.(io.Closer)
		if !ok {
			continue
		}

		if file, ok := writer.(*os.File); ok {
			if file == os.Stdout || file == os.Stderr || file == os.Stdin {
				continue
			}
		}

		closers = append(closers, closer)
	}

	return &Logger{
		component: component,
		base:      log.New(io.MultiWriter(writers...), "", 0),
		closers:   closers,
	}
}

func (l *Logger) Infof(format string, args ...any) {
	l.logf("INFO", format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.logf("WARN", format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.logf("ERROR", format, args...)
}

func (l *Logger) logf(level string, format string, args ...any) {
	// We include this silent failure because the retries.Retry decorator doesn't require a logger
	// and it's easier to make sure that doesn't fail here
	if l == nil {
		return
	}
	l.base.Printf("%s [%s] %s %s", l.component, level, time.Now().UTC().Format(timestampFormat), fmt.Sprintf(format, args...))
}

func (l *Logger) Close() error {
	var closeErr error
	l.closeOnce.Do(func() {
		for _, closer := range l.closers {
			if err := closer.Close(); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	})

	return closeErr
}
