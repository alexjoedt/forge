//nolint:revive // package name "log" intentionally used for ergonomic import.
package log

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Logger provides context-aware leveled logging.
type Logger struct {
	output  io.Writer
	verbose bool
}

type contextKey string

const loggerKey contextKey = "logger"

//nolint:gochecknoglobals
var DefaultLogger = New(false)

// New creates a new Logger writing to stdout.
func New(verbose bool) *Logger {
	return &Logger{
		output:  os.Stdout,
		verbose: verbose,
	}
}

// Setup configures the DefaultLogger with the given verbosity without
// reassigning the global logger instance.
func Setup(verbose bool) {
	DefaultLogger.verbose = verbose
}

// Verbosef logs a formatted message when verbose mode is enabled.
func (l *Logger) Verbosef(format string, args ...any) {
	if l.verbose {
		l.Printf(format, args...)
	}
}

// Verboseln logs a line when verbose mode is enabled.
func (l *Logger) Verboseln(args ...any) {
	if l.verbose {
		l.Println(args...)
	}
}

// Infof logs an info-level message when verbose mode is enabled.
func (l *Logger) Infof(msg string, args ...any) {
	if l.verbose {
		l.logWithLevel("INFO", msg, args...)
	}
}

// Debugf logs a debug-level message when verbose mode is enabled.
func (l *Logger) Debugf(msg string, args ...any) {
	if l.verbose {
		l.logWithLevel("DEBUG", msg, args...)
	}
}

// Warnf logs a warning-level message.
func (l *Logger) Warnf(msg string, args ...any) {
	l.logWithLevel("WARN", msg, args...)
}

// Errorf logs an error-level message.
func (l *Logger) Errorf(msg string, args ...any) {
	l.logWithLevel("ERROR", msg, args...)
}

// Success logs a success message.
func (l *Logger) Success(msg string, args ...any) {
	l.Printf(msg, args...)
}

// Println writes args to output followed by a newline.
func (l *Logger) Println(args ...any) {
	fmt.Fprintln(l.output, args...)
}

// Printf writes a formatted string to output.
// A newline is appended if the formatted string does not end with one.
func (l *Logger) Printf(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s += "\n"
	}
	fmt.Fprint(l.output, s)
}

func (l *Logger) logWithLevel(level, msg string, args ...any) {
	msg = "[" + level + "] " + msg
	l.Printf(msg, args...)
}

// WithLogger returns a new context carrying the given logger.
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger stored in ctx, or returns DefaultLogger.
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	return DefaultLogger
}
