package app

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
)

type LogLevel int32

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger interface {
	// Debugf logs a debug-level message.
	Debugf(format string, args ...any)
	// Infof logs an info-level message.
	Infof(format string, args ...any)
	// Warnf logs a warning message.
	Warnf(format string, args ...any)
	// Errorf logs an error-level message.
	Errorf(format string, args ...any)
}

type stdLogger struct {
	level atomic.Int32
	l     *log.Logger
}

// NewLogger creates a logger instance that writes to stdout with the requested
// minimum level.
func NewLogger(level string) (Logger, error) {
	lvl, err := parseLevel(level)
	if err != nil {
		return nil, err
	}
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	sl := &stdLogger{l: logger}
	sl.level.Store(int32(lvl))
	return sl, nil
}

// parseLevel converts the textual representation into the internal log level.
func parseLevel(level string) (LogLevel, error) {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error", "err":
		return LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", level)
	}
}

func (l *stdLogger) enabled(level LogLevel) bool {
	return LogLevel(l.level.Load()) <= level
}

func (l *stdLogger) logf(level LogLevel, prefix, format string, args ...any) {
	if !l.enabled(level) {
		return
	}
	l.l.Printf(prefix+format, args...)
}

func (l *stdLogger) Debugf(format string, args ...any) {
	l.logf(LevelDebug, "[DEBUG] ", format, args...)
}

func (l *stdLogger) Infof(format string, args ...any) {
	l.logf(LevelInfo, "[INFO] ", format, args...)
}

func (l *stdLogger) Warnf(format string, args ...any) {
	l.logf(LevelWarn, "[WARN] ", format, args...)
}

func (l *stdLogger) Errorf(format string, args ...any) {
	l.logf(LevelError, "[ERROR] ", format, args...)
}
