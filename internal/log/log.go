package log

import (
	"context"
	"log"
	"os"
)

type Logger struct {
	verbose bool
	*log.Logger
}

type contextKey string

const loggerKey contextKey = "logger"

var DefaultLogger = New(false)

func New(verbose bool) *Logger {
	return &Logger{
		verbose: verbose,
		Logger:  log.New(os.Stdout, "", 0),
	}
}

func Setup(verbose bool) {
	DefaultLogger = New(verbose)
}

func (l *Logger) Verbosef(format string, args ...any) {
	if l.verbose {
		l.Printf(format, args...)
	}
}

func (l *Logger) Verboseln(args ...any) {
	if l.verbose {
		l.Println(args...)
	}
}

func (l *Logger) Infof(msg string, args ...any) {
	if l.verbose {
		l.logWithLevel("INFO", msg, args...)
	}
}

func (l *Logger) Debugf(msg string, args ...any) {
	if l.verbose {
		l.logWithLevel("DEBUG", msg, args...)
	}
}

func (l *Logger) Warnf(msg string, args ...any) {
	l.logWithLevel("WARN", msg, args...)
}

func (l *Logger) Errorf(msg string, args ...any) {
	l.logWithLevel("ERROR", msg, args...)
}

func (l *Logger) Success(msg string, args ...any) {
	l.Printf(msg, args...)
}

func (l *Logger) logWithLevel(level, msg string, args ...any) {
	msg = "[" + level + "] " + msg
	l.Printf(msg, args...)
}

func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	return DefaultLogger
}
