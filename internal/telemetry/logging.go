package telemetry

import (
    "log"
    "os"
)

// Logger is a struct that holds the logger instance
type Logger struct {
    *log.Logger
}

// NewLogger initializes a new logger instance
func NewLogger() *Logger {
    // Create a log file
    logFile, err := os.OpenFile("quic_reverse_proxy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("error opening log file: %v", err)
    }

    // Create a new logger
    logger := log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
    return &Logger{logger}
}

// Info logs informational messages
func (l *Logger) Info(msg string) {
    l.Println("INFO: " + msg)
}

// Error logs error messages
func (l *Logger) Error(msg string) {
    l.Println("ERROR: " + msg)
}

// Debug logs debug messages
func (l *Logger) Debug(msg string) {
    l.Println("DEBUG: " + msg)
}