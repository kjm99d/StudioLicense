package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	levelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}

	levelColors = map[LogLevel]string{
		DEBUG: "\033[36m", // Cyan
		INFO:  "\033[32m", // Green
		WARN:  "\033[33m", // Yellow
		ERROR: "\033[31m", // Red
		FATAL: "\033[35m", // Magenta
	}

	resetColor = "\033[0m"
)

// Logger is a simple structured logger that writes to multiple destinations.
type Logger struct {
	level      LogLevel
	writers    []io.Writer
	mu         sync.Mutex
	useColor   bool
	prefix     string
	showCaller bool
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Config describes how the logger should be initialised.
type Config struct {
	Level      LogLevel
	LogDir     string
	MaxSize    int64 // bytes
	MaxAge     int   // days
	UseColor   bool
	ShowCaller bool
	Prefix     string
}

// Initialize boots the global logger instance if it has not been created yet.
func Initialize(config Config) error {
	var err error
	once.Do(func() {
		defaultLogger = &Logger{
			level:      config.Level,
			writers:    []io.Writer{},
			useColor:   config.UseColor,
			prefix:     config.Prefix,
			showCaller: config.ShowCaller,
		}

		// Append console writer with optional ANSI colours.
		defaultLogger.writers = append(defaultLogger.writers, os.Stdout)

		// Build the log directory when provided.
		if config.LogDir != "" {
			if err = os.MkdirAll(config.LogDir, 0755); err != nil {
				return
			}

			// Open the log file for the current day.
			logFile, fileErr := createLogFile(config.LogDir)
			if fileErr != nil {
				err = fileErr
				return
			}

			defaultLogger.writers = append(defaultLogger.writers, logFile)

			// Spawn rotation worker.
			go rotateLogFiles(config.LogDir, config.MaxSize, config.MaxAge)
		}
	})

	return err
}

// createLogFile creates (or opens) the log file for the current day.
func createLogFile(logDir string) (*os.File, error) {
	timestamp := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("server-%s.log", timestamp))

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// rotateLogFiles periodically rotates and prunes log files.
func rotateLogFiles(logDir string, maxSize int64, maxAge int) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Clean up historic log files.
		files, _ := filepath.Glob(filepath.Join(logDir, "server-*.log"))
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}

			// Remove files older than retention.
			if time.Since(info.ModTime()).Hours() > float64(maxAge*24) {
				os.Remove(file)
				continue
			}

			// Rotate when the file exceeds the maximum size.
			if maxSize > 0 && info.Size() > maxSize {
				// Rename the current file for archiving.
				newName := strings.Replace(file, ".log", fmt.Sprintf("-%d.log", time.Now().Unix()), 1)
				os.Rename(file, newName)
			}
		}
	}
}

// log writes the formatted entry to the underlying logger.
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	// Reset colours for non-console writers.
	caller := ""
	if l.showCaller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			file = filepath.Base(file)
			caller = fmt.Sprintf(" [%s:%d]", file, line)
		}
	}

	// 濡쒓렇 硫붿떆吏 援ъ꽦
	var logMessage string
	for i, writer := range l.writers {
		if i == 0 && l.useColor { // Apply colour only to stdout.
			color := levelColors[level]
			logMessage = fmt.Sprintf("%s%s [%s]%s %s%s%s\n",
				timestamp, caller, levelName, l.prefix, color, message, resetColor)
		} else {
			logMessage = fmt.Sprintf("%s%s [%s]%s %s\n",
				timestamp, caller, levelName, l.prefix, message)
		}

		writer.Write([]byte(logMessage))
	}

	// Exit process on fatal level.
	if level == FATAL {
		os.Exit(1)
	}
}

// Public helper methods for the default logger.
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(DEBUG, format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(INFO, format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(WARN, format, args...)
	} else {
		log.Printf("[WARN] "+format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(ERROR, format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

func Fatal(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.log(FATAL, format, args...)
	} else {
		log.Fatalf("[FATAL] "+format, args...)
	}
}

// WithFields attaches structured fields to the log entry.
func WithFields(fields map[string]interface{}) *LogEntry {
	return &LogEntry{
		fields: fields,
		logger: defaultLogger,
	}
}

// LogEntry represents a structured log entry builder.
type LogEntry struct {
	fields map[string]interface{}
	logger *Logger
}

func (e *LogEntry) Debug(format string, args ...interface{}) {
	e.log(DEBUG, format, args...)
}

func (e *LogEntry) Info(format string, args ...interface{}) {
	e.log(INFO, format, args...)
}

func (e *LogEntry) Warn(format string, args ...interface{}) {
	e.log(WARN, format, args...)
}

func (e *LogEntry) Error(format string, args ...interface{}) {
	e.log(ERROR, format, args...)
}

func (e *LogEntry) Fatal(format string, args ...interface{}) {
	e.log(FATAL, format, args...)
}

func (e *LogEntry) log(level LogLevel, format string, args ...interface{}) {
	if e.logger == nil || level < e.logger.level {
		return
	}

	message := fmt.Sprintf(format, args...)

	// Attach formatted fields to the log message.
	if len(e.fields) > 0 {
		var fieldStrs []string
		for k, v := range e.fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		message = fmt.Sprintf("%s | %s", message, strings.Join(fieldStrs, ", "))
	}

	e.logger.log(level, "%s", message)
}

// Log allows emitting a message with an explicit level via the entry.
func (e *LogEntry) Log(level LogLevel, format string, args ...interface{}) {
	e.log(level, format, args...)
}

// SetLevel updates the global logging level.
func SetLevel(level LogLevel) {
	if defaultLogger != nil {
		defaultLogger.mu.Lock()
		defaultLogger.level = level
		defaultLogger.mu.Unlock()
	}
}

// GetLevel returns the current global logging level.
func GetLevel() LogLevel {
	if defaultLogger != nil {
		return defaultLogger.level
	}
	return INFO
}
