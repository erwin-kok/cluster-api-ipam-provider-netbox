package logger

import (
	"context"

	"github.com/go-logr/logr"
)

// These are the log levels used by the logger.
// See https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md#what-method-to-use
const (
	logLevelWarn  = 1
	logLevelDebug = 4
	logLevelTrace = 5
)

// Logger is a concrete logger using logr underneath.
type Logger struct {
	callStackHelper func()
	logger          logr.Logger
}

// NewLogger creates a logger with a passed in logr.Logger implementation directly.
func NewLogger(log logr.Logger) *Logger {
	helper, log := log.WithCallStackHelper()
	return &Logger{
		callStackHelper: helper,
		logger:          log,
	}
}

// FromContext retrieves the logr implementation from Context and uses it as underlying logger.
func FromContext(ctx context.Context) *Logger {
	helper, log := logr.FromContextOrDiscard(ctx).WithCallStackHelper()
	return &Logger{
		callStackHelper: helper,
		logger:          log,
	}
}

// Info logs a message at the info level.
func (c *Logger) Info(msg string, keysAndValues ...any) {
	c.callStackHelper()
	c.logger.Info(msg, keysAndValues...)
}

// Debug logs a message at the debug level.
func (c *Logger) Debug(msg string, keysAndValues ...any) {
	c.callStackHelper()
	c.logger.V(logLevelDebug).Info(msg, keysAndValues...)
}

// Warn logs a message at the warn level.
func (c *Logger) Warn(msg string, keysAndValues ...any) {
	c.callStackHelper()
	c.logger.V(logLevelWarn).Info(msg, keysAndValues...)
}

// Trace logs a message at the trace level.
func (c *Logger) Trace(msg string, keysAndValues ...any) {
	c.callStackHelper()
	c.logger.V(logLevelTrace).Info(msg, keysAndValues...)
}

// Error logs a message at the error level.
func (c *Logger) Error(err error, msg string, keysAndValues ...any) {
	c.callStackHelper()
	c.logger.Error(err, msg, keysAndValues...)
}

// GetLogger returns the underlying logr.Logger.
func (c *Logger) GetLogger() logr.Logger {
	return c.logger
}

// WithValues adds some key-value pairs of context to a logger.
func (c *Logger) WithValues(keysAndValues ...any) *Logger {
	return &Logger{
		callStackHelper: c.callStackHelper,
		logger:          c.logger.WithValues(keysAndValues...),
	}
}

// WithName adds a new element to the logger's name.
func (c *Logger) WithName(name string) *Logger {
	return &Logger{
		callStackHelper: c.callStackHelper,
		logger:          c.logger.WithName(name),
	}
}
