package ports

import (
	logger "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
)

// Logger provides fast, leveled, structured logging. All methods are safe
// for concurrent use.
type LoggerService interface {
	// Log logs a message at the specified level. The message includes any fields
	// passed at the log site
	Log(lvl logger.Level, msg string, fields ...logger.Field)

	// Debug logs a message at DebugLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Debug(msg string, fields ...logger.Field)

	// Info logs a message at InfoLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Info(msg string, fields ...logger.Field)

	// Warn logs a message at WarnLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Warn(msg string, fields ...logger.Field)

	// Error logs a message at ErrorLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.

	Error(msg string, fields ...logger.Field)
	// Panic logs a message at PanicLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	//
	// The logger then panics, even if logging at PanicLevel is disabled.
	Panic(msg string, fields ...logger.Field)

	// Fatal logs a message at FatalLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	//
	// The logger then calls os.Exit(1), even if logging at FatalLevel is
	// disabled.
	Fatal(msg string, fields ...logger.Field)
}
