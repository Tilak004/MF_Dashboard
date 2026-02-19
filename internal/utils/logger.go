package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
	logFile     *os.File
)

// InitLogger initializes the logging system
func InitLogger() error {
	// Create logs directory if it doesn't exist
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with current date
	logFileName := filepath.Join(logsDir, fmt.Sprintf("app_%s.log", time.Now().Format("2006-01-02")))
	var err error
	logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create multi-writer to write to both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Initialize loggers
	InfoLogger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	InfoLogger.Println("Logger initialized successfully")
	return nil
}

// CloseLogger closes the log file
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// LogRequest logs HTTP request details
func LogRequest(method, path, remoteAddr string, statusCode int, duration time.Duration) {
	InfoLogger.Printf("[%s] %s from %s - Status: %d - Duration: %v",
		method, path, remoteAddr, statusCode, duration)
}

// LogError logs error with context
func LogError(context string, err error) {
	ErrorLogger.Printf("[%s] Error: %v", context, err)
}

// LogDebug logs debug information (only in development)
func LogDebug(context string, message string) {
	if os.Getenv("DEBUG") == "true" {
		DebugLogger.Printf("[%s] %s", context, message)
	}
}

// LogInfo logs general information
func LogInfo(context string, message string) {
	InfoLogger.Printf("[%s] %s", context, message)
}
