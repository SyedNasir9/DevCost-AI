package logger

import (
	"go.uber.org/zap"
)

// Logger wraps zap logger with additional functionality
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new structured logger
func NewLogger(config *zap.Config) (*Logger, error) {
	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{Logger: zapLogger}, nil
}

// WithRequestID adds request ID to the logger
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{Logger: l.Logger.With(zap.String("request_id", requestID))}
}

// WithUserID adds user ID to the logger
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{Logger: l.Logger.With(zap.String("user_id", userID))}
}

// WithComponent adds component name to the logger
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{Logger: l.Logger.With(zap.String("component", component))}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
