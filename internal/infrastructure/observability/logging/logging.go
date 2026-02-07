package logging

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggingConfig struct {
	LogLevel       string
	LogFormat      string
	LogFile        string
	LokiURL        string
	ServiceName    string
	ServiceVersion string
	Environment    string
}

func NewLogger(config LoggingConfig) (*zap.Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(config.LogLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.MessageKey = "message"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"
	encoderConfig.StacktraceKey = "stacktrace"

	// Add service information
	encoderConfig.FunctionKey = "function"
	encoderConfig.NameKey = "logger"

	// Choose encoder based on format
	var encoder zapcore.Encoder
	if config.LogFormat == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create cores
	var cores []zapcore.Core

	// Console core
	consoleCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
	cores = append(cores, consoleCore)

	// File core (if log file is specified)
	if config.LogFile != "" {
		fileCore := zapcore.NewCore(
			encoder,
			zapcore.AddSync(&lumberjack.Logger{
				Filename:   config.LogFile,
				MaxSize:    100, // megabytes
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}),
			level,
		)
		cores = append(cores, fileCore)
	}

	// Create core
	core := zapcore.NewTee(cores...)

	// Create logger
	logger := zap.New(core,
		zap.AddCaller(),
		// zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1),
	)

	// Add service context
	fields := []zap.Field{
		zap.String("service", config.ServiceName),
		zap.String("version", config.ServiceVersion),
		zap.String("environment", config.Environment),
		zap.String("hostname", getHostname()),
	}


	logger = logger.With(fields...)

	return logger, nil
}

// Helper function to get hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// Helper function to create structured logger with additional fields
func WithFields(logger *zap.Logger, fields map[string]interface{}) *zap.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return logger.With(zapFields...)
}

// Helper function to log with trace context
func LogWithTrace(logger *zap.Logger, traceID, spanID string, message string, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	)
	logger.Info(message, allFields...)
}

// Helper function to create request logger
func NewRequestLogger(logger *zap.Logger, method, path, userID string) *zap.Logger {
	return logger.With(
		zap.String("method", method),
		zap.String("path", path),
		zap.String("user_id", userID),
		zap.Time("request_time", time.Now()),
	)
}
