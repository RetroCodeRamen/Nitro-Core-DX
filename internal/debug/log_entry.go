package debug

import (
	"fmt"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelError
	LogLevelWarning
	LogLevelInfo
	LogLevelDebug
	LogLevelTrace
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelNone:
		return "NONE"
	case LogLevelError:
		return "ERROR"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelInfo:
		return "INFO"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelTrace:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

// Component represents the component that generated the log entry
type Component string

const (
	ComponentCPU    Component = "CPU"
	ComponentPPU    Component = "PPU"
	ComponentAPU    Component = "APU"
	ComponentMemory Component = "Memory"
	ComponentInput  Component = "Input"
	ComponentUI     Component = "UI"
	ComponentSystem Component = "System"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	Component Component
	Level     LogLevel
	Message   string
	Data      map[string]interface{} // Optional structured data
}

// Format formats the log entry as a string
func (e *LogEntry) Format() string {
	timestamp := e.Timestamp.Format("15:04:05.000")
	return fmt.Sprintf("[%s] [%s] %s: %s", timestamp, e.Component, e.Level, e.Message)
}

