package common

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel representa el nivel de severidad de un log
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// Logger es una estructura simple para logging estructurado
type Logger struct {
	component string
	level     LogLevel
	logger    *log.Logger
}

// NewLogger crea una nueva instancia de Logger para un componente
func NewLogger(component string, level LogLevel) *Logger {
	return &Logger{
		component: component,
		level:     level,
		logger:    log.New(os.Stdout, "", 0),
	}
}

// log es el método interno que formatea y escribe los logs
func (l *Logger) log(level LogLevel, message string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	
	timestamp := time.Now().Format(time.RFC3339)
	formattedMsg := fmt.Sprintf(message, args...)
	logLine := fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, level, l.component, formattedMsg)
	l.logger.Println(logLine)
}

// shouldLog determina si un mensaje debe ser loggeado según el nivel configurado
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
	}
	return levels[level] >= levels[l.level]
}

// Debug registra un mensaje de nivel DEBUG
func (l *Logger) Debug(message string, args ...interface{}) {
	l.log(LogLevelDebug, message, args...)
}

// Info registra un mensaje de nivel INFO
func (l *Logger) Info(message string, args ...interface{}) {
	l.log(LogLevelInfo, message, args...)
}

// Warn registra un mensaje de nivel WARN
func (l *Logger) Warn(message string, args ...interface{}) {
	l.log(LogLevelWarn, message, args...)
}

// Error registra un mensaje de nivel ERROR
func (l *Logger) Error(message string, args ...interface{}) {
	l.log(LogLevelError, message, args...)
}
