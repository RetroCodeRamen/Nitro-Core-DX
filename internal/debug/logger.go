package debug

import (
	"fmt"
	"sync"
	"time"
)

// Logger represents the centralized logging system
type Logger struct {
	// Circular buffer for log entries
	entries    []LogEntry
	entriesMu  sync.RWMutex
	maxEntries int
	writeIndex int
	entryCount int

	// Component enable/disable flags
	componentEnabled map[Component]bool
	componentMu      sync.RWMutex

	// Minimum log level (entries below this level are filtered)
	minLevel LogLevel
	levelMu  sync.RWMutex

	// Channel for thread-safe logging
	logChan chan LogEntry

	// Shutdown channel
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewLogger creates a new logger instance
func NewLogger(maxEntries int) *Logger {
	if maxEntries < 100 {
		maxEntries = 100 // Minimum buffer size
	}

	logger := &Logger{
		entries:          make([]LogEntry, maxEntries),
		maxEntries:       maxEntries,
		writeIndex:       0,
		entryCount:       0,
		componentEnabled: make(map[Component]bool),
		minLevel:         LogLevelInfo, // Default to Info level
		logChan:          make(chan LogEntry, 1000), // Buffered channel
		shutdown:         make(chan struct{}),
	}

	// Disable all components by default (logging is opt-in)
	logger.componentEnabled[ComponentCPU] = false
	logger.componentEnabled[ComponentPPU] = false
	logger.componentEnabled[ComponentAPU] = false
	logger.componentEnabled[ComponentMemory] = false
	logger.componentEnabled[ComponentInput] = false
	logger.componentEnabled[ComponentUI] = false
	logger.componentEnabled[ComponentSystem] = false

	// Start log processing goroutine
	logger.wg.Add(1)
	go logger.processLogs()

	return logger
}

// processLogs processes log entries from the channel
func (l *Logger) processLogs() {
	defer l.wg.Done()

	for {
		select {
		case entry := <-l.logChan:
			l.addEntry(entry)
		case <-l.shutdown:
			// Drain remaining logs
			for {
				select {
				case entry := <-l.logChan:
					l.addEntry(entry)
				default:
					return
				}
			}
		}
	}
}

// addEntry adds a log entry to the circular buffer
func (l *Logger) addEntry(entry LogEntry) {
	l.entriesMu.Lock()
	defer l.entriesMu.Unlock()

	// Add entry at current write index
	l.entries[l.writeIndex] = entry
	l.writeIndex = (l.writeIndex + 1) % l.maxEntries

	// Update entry count (don't exceed maxEntries)
	if l.entryCount < l.maxEntries {
		l.entryCount++
	}
}

// Log logs a message with the specified component and level
func (l *Logger) Log(component Component, level LogLevel, message string, data map[string]interface{}) {
	// Check if component is enabled
	l.componentMu.RLock()
	enabled := l.componentEnabled[component]
	l.componentMu.RUnlock()

	if !enabled {
		return
	}

	// Check if level is high enough
	l.levelMu.RLock()
	minLevel := l.minLevel
	l.levelMu.RUnlock()

	if level < minLevel {
		return
	}

	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now(),
		Component: component,
		Level:     level,
		Message:   message,
		Data:      data,
	}

	// Send to channel (non-blocking if channel is full)
	select {
	case l.logChan <- entry:
	default:
		// Channel is full, drop entry (or could use a larger buffer)
		// For now, we'll drop it to prevent blocking
	}
}

// Logf logs a formatted message
func (l *Logger) Logf(component Component, level LogLevel, format string, args ...interface{}) {
	l.Log(component, level, fmt.Sprintf(format, args...), nil)
}

// Convenience methods for each component
func (l *Logger) LogCPU(level LogLevel, message string, data map[string]interface{}) {
	l.Log(ComponentCPU, level, message, data)
}

func (l *Logger) LogPPU(level LogLevel, message string, data map[string]interface{}) {
	l.Log(ComponentPPU, level, message, data)
}

func (l *Logger) LogAPU(level LogLevel, message string, data map[string]interface{}) {
	l.Log(ComponentAPU, level, message, data)
}

func (l *Logger) LogMemory(level LogLevel, message string, data map[string]interface{}) {
	l.Log(ComponentMemory, level, message, data)
}

func (l *Logger) LogInput(level LogLevel, message string, data map[string]interface{}) {
	l.Log(ComponentInput, level, message, data)
}

func (l *Logger) LogUI(level LogLevel, message string, data map[string]interface{}) {
	l.Log(ComponentUI, level, message, data)
}

func (l *Logger) LogSystem(level LogLevel, message string, data map[string]interface{}) {
	l.Log(ComponentSystem, level, message, data)
}

// Convenience methods with formatted strings
func (l *Logger) LogCPUf(level LogLevel, format string, args ...interface{}) {
	l.Logf(ComponentCPU, level, format, args...)
}

func (l *Logger) LogPPUf(level LogLevel, format string, args ...interface{}) {
	l.Logf(ComponentPPU, level, format, args...)
}

func (l *Logger) LogAPUf(level LogLevel, format string, args ...interface{}) {
	l.Logf(ComponentAPU, level, format, args...)
}

func (l *Logger) LogMemoryf(level LogLevel, format string, args ...interface{}) {
	l.Logf(ComponentMemory, level, format, args...)
}

func (l *Logger) LogInputf(level LogLevel, format string, args ...interface{}) {
	l.Logf(ComponentInput, level, format, args...)
}

func (l *Logger) LogUIf(level LogLevel, format string, args ...interface{}) {
	l.Logf(ComponentUI, level, format, args...)
}

func (l *Logger) LogSystemf(level LogLevel, format string, args ...interface{}) {
	l.Logf(ComponentSystem, level, format, args...)
}

// GetEntries returns a copy of all log entries (oldest first)
func (l *Logger) GetEntries() []LogEntry {
	l.entriesMu.RLock()
	defer l.entriesMu.RUnlock()

	if l.entryCount == 0 {
		return []LogEntry{}
	}

	entries := make([]LogEntry, l.entryCount)

	if l.entryCount < l.maxEntries {
		// Buffer not full yet, return entries from 0 to entryCount
		copy(entries, l.entries[:l.entryCount])
	} else {
		// Buffer is full, return entries starting from writeIndex (oldest)
		// and wrapping around
		for i := 0; i < l.entryCount; i++ {
			idx := (l.writeIndex + i) % l.maxEntries
			entries[i] = l.entries[idx]
		}
	}

	return entries
}

// GetRecentEntries returns the most recent N entries
func (l *Logger) GetRecentEntries(count int) []LogEntry {
	allEntries := l.GetEntries()
	if count >= len(allEntries) {
		return allEntries
	}
	return allEntries[len(allEntries)-count:]
}

// Clear clears all log entries
func (l *Logger) Clear() {
	l.entriesMu.Lock()
	defer l.entriesMu.Unlock()

	l.entryCount = 0
	l.writeIndex = 0
}

// SetComponentEnabled enables or disables logging for a component
func (l *Logger) SetComponentEnabled(component Component, enabled bool) {
	l.componentMu.Lock()
	defer l.componentMu.Unlock()
	l.componentEnabled[component] = enabled
}

// IsComponentEnabled returns whether a component is enabled
func (l *Logger) IsComponentEnabled(component Component) bool {
	l.componentMu.RLock()
	defer l.componentMu.RUnlock()
	return l.componentEnabled[component]
}

// SetMinLevel sets the minimum log level
func (l *Logger) SetMinLevel(level LogLevel) {
	l.levelMu.Lock()
	defer l.levelMu.Unlock()
	l.minLevel = level
}

// GetMinLevel returns the minimum log level
func (l *Logger) GetMinLevel() LogLevel {
	l.levelMu.RLock()
	defer l.levelMu.RUnlock()
	return l.minLevel
}

// Shutdown shuts down the logger and waits for all logs to be processed
func (l *Logger) Shutdown() {
	close(l.shutdown)
	l.wg.Wait()
}

