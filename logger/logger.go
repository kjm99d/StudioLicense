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

// LogLevel 로그 레벨
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

// Logger 구조화된 로거
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

// Config 로거 설정
type Config struct {
	Level      LogLevel
	LogDir     string
	MaxSize    int64 // bytes
	MaxAge     int   // days
	UseColor   bool
	ShowCaller bool
	Prefix     string
}

// Initialize 로거 초기화
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

		// 콘솔 출력 추가
		defaultLogger.writers = append(defaultLogger.writers, os.Stdout)

		// 로그 디렉토리 생성
		if config.LogDir != "" {
			if err = os.MkdirAll(config.LogDir, 0755); err != nil {
				return
			}

			// 로그 파일 생성
			logFile, fileErr := createLogFile(config.LogDir)
			if fileErr != nil {
				err = fileErr
				return
			}

			defaultLogger.writers = append(defaultLogger.writers, logFile)

			// 로그 rotation 고루틴 시작
			go rotateLogFiles(config.LogDir, config.MaxSize, config.MaxAge)
		}
	})

	return err
}

// createLogFile 로그 파일 생성
func createLogFile(logDir string) (*os.File, error) {
	timestamp := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("server-%s.log", timestamp))

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// rotateLogFiles 로그 파일 rotation
func rotateLogFiles(logDir string, maxSize int64, maxAge int) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// 오래된 로그 파일 삭제
		files, _ := filepath.Glob(filepath.Join(logDir, "server-*.log"))
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}

			// 파일 나이 체크
			if time.Since(info.ModTime()).Hours() > float64(maxAge*24) {
				os.Remove(file)
				continue
			}

			// 파일 크기 체크
			if maxSize > 0 && info.Size() > maxSize {
				// 파일 이름 변경 (압축 또는 아카이브)
				newName := strings.Replace(file, ".log", fmt.Sprintf("-%d.log", time.Now().Unix()), 1)
				os.Rename(file, newName)
			}
		}
	}
}

// log 내부 로깅 함수
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	// 호출자 정보
	caller := ""
	if l.showCaller {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			file = filepath.Base(file)
			caller = fmt.Sprintf(" [%s:%d]", file, line)
		}
	}

	// 로그 메시지 구성
	var logMessage string
	for i, writer := range l.writers {
		if i == 0 && l.useColor { // 콘솔에만 컬러 적용
			color := levelColors[level]
			logMessage = fmt.Sprintf("%s%s [%s]%s %s%s%s\n",
				timestamp, caller, levelName, l.prefix, color, message, resetColor)
		} else {
			logMessage = fmt.Sprintf("%s%s [%s]%s %s\n",
				timestamp, caller, levelName, l.prefix, message)
		}

		writer.Write([]byte(logMessage))
	}

	// FATAL 레벨은 프로그램 종료
	if level == FATAL {
		os.Exit(1)
	}
}

// Public 로깅 함수들
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

// WithFields 구조화된 로깅을 위한 필드 추가
func WithFields(fields map[string]interface{}) *LogEntry {
	return &LogEntry{
		fields: fields,
		logger: defaultLogger,
	}
}

// LogEntry 구조화된 로그 엔트리
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

	// 필드를 메시지에 추가
	if len(e.fields) > 0 {
		var fieldStrs []string
		for k, v := range e.fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		message = fmt.Sprintf("%s | %s", message, strings.Join(fieldStrs, ", "))
	}

	e.logger.log(level, message)
}

// Log public 로그 메서드 (외부 호출용)
func (e *LogEntry) Log(level LogLevel, format string, args ...interface{}) {
	e.log(level, format, args...)
}

// SetLevel 로그 레벨 변경
func SetLevel(level LogLevel) {
	if defaultLogger != nil {
		defaultLogger.mu.Lock()
		defaultLogger.level = level
		defaultLogger.mu.Unlock()
	}
}

// GetLevel 현재 로그 레벨 반환
func GetLevel() LogLevel {
	if defaultLogger != nil {
		return defaultLogger.level
	}
	return INFO
}
