package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger is a wrapper around logrus.Logger
type Logger struct {
	*logrus.Logger
}

// Config holds the configuration for the logger
type Config struct {
	Level      string
	Format     string
	Output     io.Writer
	TimeFormat string
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "text",
		Output:     os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}
}

// New creates a new logger with the given configuration
func New(config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.TimeFormat,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: config.TimeFormat,
		})
	}

	// Set output
	logger.SetOutput(config.Output)

	return &Logger{
		Logger: logger,
	}
}

// NewFileLogger creates a new logger that writes to a file
func NewFileLogger(filename string, config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	config.Output = file
	return New(config), nil
}

// NewConsoleLogger creates a new logger that writes to the console
func NewConsoleLogger(config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	config.Output = os.Stdout
	return New(config)
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError adds an error to the logger
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Info logs a message at level Info
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

// Fatal logs a message at level Fatal then the process will exit with status set to 1
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

// Error logs a message at level Error
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}
