package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// Logger is a custom logger that writes logs to both the terminal and a JSON file.
type Logger struct {
	filePath string
	file     *os.File
}

// NewLogger creates a new Logger instance.
func NewLogger(filePath string) (*Logger, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &Logger{
		filePath: filePath,
		file:     file,
	}, nil
}

// Log logs a message or error to the terminal and the JSON file.
func (l *Logger) Log(message interface{}) {
	// Create a log entry
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   fmt.Sprintf("%v", message),
	}

	// Print to terminal
	fmt.Printf("[%s] : %s\n", entry.Timestamp, entry.Message)

	// Write to JSON file
	if err := l.writeJSONLog(entry); err != nil {
		fmt.Printf("Error writing to log file: %v\n", err)
	}
}

// writeJSONLog writes a log entry to the JSON file.
func (l *Logger) writeJSONLog(entry LogEntry) error {
	encoder := json.NewEncoder(l.file)
	encoder.SetIndent("", "  ") // Pretty print
	if err := encoder.Encode(entry); err != nil {
		return err
	}
	return nil
}

// Close closes the log file.
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
