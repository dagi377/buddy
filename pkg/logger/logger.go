package logger

import (
	"github.com/sirupsen/logrus"
	"github.com/ai-agent-framework/pkg/interfaces"
)

// LogrusLogger implements the Logger interface using logrus
type LogrusLogger struct {
	*logrus.Entry
}

// NewLogrusLogger creates a new logrus-based logger
func NewLogrusLogger(level string) interfaces.Logger {
	logger := logrus.New()
	
	// Set log level
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})

	return &LogrusLogger{
		Entry: logrus.NewEntry(logger),
	}
}

// WithField adds a field to the logger
func (l *LogrusLogger) WithField(key string, value interface{}) interfaces.Logger {
	return &LogrusLogger{
		Entry: l.Entry.WithField(key, value),
	}
}

// WithFields adds multiple fields to the logger
func (l *LogrusLogger) WithFields(fields map[string]interface{}) interfaces.Logger {
	return &LogrusLogger{
		Entry: l.Entry.WithFields(logrus.Fields(fields)),
	}
}

// Debug logs a debug message
func (l *LogrusLogger) Debug(args ...interface{}) {
	l.Entry.Debug(args...)
}

// Info logs an info message
func (l *LogrusLogger) Info(args ...interface{}) {
	l.Entry.Info(args...)
}

// Warn logs a warning message
func (l *LogrusLogger) Warn(args ...interface{}) {
	l.Entry.Warn(args...)
}

// Error logs an error message
func (l *LogrusLogger) Error(args ...interface{}) {
	l.Entry.Error(args...)
}
