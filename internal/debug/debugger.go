package debug

import (
	"fmt"
	"sync"
)

// Breakpoint represents a breakpoint in the debugger
type Breakpoint struct {
	Bank   uint8
	Offset uint16
	Enabled bool
	HitCount int
}

// WatchExpression represents a watch expression to monitor
type WatchExpression struct {
	Expression string
	Value      interface{}
	LastValue  interface{}
}

// Debugger represents the interactive debugger
type Debugger struct {
	// Breakpoints
	breakpoints map[string]*Breakpoint // Key: "bank:offset" format
	breakpointsMu sync.RWMutex

	// Watch expressions
	watches []*WatchExpression
	watchesMu sync.RWMutex

	// Execution state
	paused bool
	stepping bool
	stepCount int
	stateMu sync.RWMutex

	// Call stack (for function call tracking)
	callStack []CallFrame
	stackMu sync.RWMutex

	// Variable tracking (for CoreLX debugging)
	variables map[string]VariableInfo
	variablesMu sync.RWMutex
}

// CallFrame represents a function call frame
type CallFrame struct {
	Bank   uint8
	Offset uint16
	FunctionName string
}

// VariableInfo represents a variable's current state
type VariableInfo struct {
	Name     string
	Type     string
	Value    interface{}
	Location string // "register", "stack", "memory"
	Address  uint32 // Memory address if applicable
}

// NewDebugger creates a new debugger instance
func NewDebugger() *Debugger {
	return &Debugger{
		breakpoints: make(map[string]*Breakpoint),
		watches:     make([]*WatchExpression, 0),
		callStack:   make([]CallFrame, 0),
		variables:   make(map[string]VariableInfo),
		paused:      false,
		stepping:    false,
	}
}

// SetBreakpoint sets a breakpoint at the specified address
func (d *Debugger) SetBreakpoint(bank uint8, offset uint16) string {
	d.breakpointsMu.Lock()
	defer d.breakpointsMu.Unlock()

	key := fmt.Sprintf("%02X:%04X", bank, offset)
	bp := &Breakpoint{
		Bank:     bank,
		Offset:   offset,
		Enabled:  true,
		HitCount: 0,
	}
	d.breakpoints[key] = bp
	return key
}

// RemoveBreakpoint removes a breakpoint
func (d *Debugger) RemoveBreakpoint(key string) bool {
	d.breakpointsMu.Lock()
	defer d.breakpointsMu.Unlock()

	if _, exists := d.breakpoints[key]; exists {
		delete(d.breakpoints, key)
		return true
	}
	return false
}

// GetBreakpoint returns a breakpoint by key
func (d *Debugger) GetBreakpoint(key string) (*Breakpoint, bool) {
	d.breakpointsMu.RLock()
	defer d.breakpointsMu.RUnlock()

	bp, exists := d.breakpoints[key]
	return bp, exists
}

// GetAllBreakpoints returns all breakpoints
func (d *Debugger) GetAllBreakpoints() map[string]*Breakpoint {
	d.breakpointsMu.RLock()
	defer d.breakpointsMu.RUnlock()

	result := make(map[string]*Breakpoint)
	for k, v := range d.breakpoints {
		result[k] = v
	}
	return result
}

// CheckBreakpoint checks if execution should break at the given address
func (d *Debugger) CheckBreakpoint(bank uint8, offset uint16) bool {
	d.breakpointsMu.RLock()
	defer d.breakpointsMu.RUnlock()

	key := fmt.Sprintf("%02X:%04X", bank, offset)
	bp, exists := d.breakpoints[key]
	if exists && bp.Enabled {
		bp.HitCount++
		return true
	}
	return false
}

// EnableBreakpoint enables a breakpoint
func (d *Debugger) EnableBreakpoint(key string) bool {
	d.breakpointsMu.Lock()
	defer d.breakpointsMu.Unlock()

	if bp, exists := d.breakpoints[key]; exists {
		bp.Enabled = true
		return true
	}
	return false
}

// DisableBreakpoint disables a breakpoint
func (d *Debugger) DisableBreakpoint(key string) bool {
	d.breakpointsMu.Lock()
	defer d.breakpointsMu.Unlock()

	if bp, exists := d.breakpoints[key]; exists {
		bp.Enabled = false
		return true
	}
	return false
}

// AddWatch adds a watch expression
func (d *Debugger) AddWatch(expr string) {
	d.watchesMu.Lock()
	defer d.watchesMu.Unlock()

	watch := &WatchExpression{
		Expression: expr,
		Value:      nil,
		LastValue:  nil,
	}
	d.watches = append(d.watches, watch)
}

// RemoveWatch removes a watch expression
func (d *Debugger) RemoveWatch(index int) bool {
	d.watchesMu.Lock()
	defer d.watchesMu.Unlock()

	if index >= 0 && index < len(d.watches) {
		d.watches = append(d.watches[:index], d.watches[index+1:]...)
		return true
	}
	return false
}

// GetWatches returns all watch expressions
func (d *Debugger) GetWatches() []*WatchExpression {
	d.watchesMu.RLock()
	defer d.watchesMu.RUnlock()

	result := make([]*WatchExpression, len(d.watches))
	copy(result, d.watches)
	return result
}

// Pause pauses execution
func (d *Debugger) Pause() {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	d.paused = true
	d.stepping = false
}

// Resume resumes execution
func (d *Debugger) Resume() {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	d.paused = false
	d.stepping = false
}

// Step sets single-step mode
func (d *Debugger) Step(count int) {
	d.stateMu.Lock()
	defer d.stateMu.Unlock()
	d.stepping = true
	d.stepCount = count
	d.paused = false
}

// IsPaused returns whether execution is paused
func (d *Debugger) IsPaused() bool {
	d.stateMu.RLock()
	defer d.stateMu.RUnlock()
	return d.paused
}

// ShouldBreak checks if execution should break (breakpoint hit or stepping)
func (d *Debugger) ShouldBreak(bank uint8, offset uint16) bool {
	// Check if stepping
	d.stateMu.RLock()
	stepping := d.stepping
	stepCount := d.stepCount
	d.stateMu.RUnlock()

	if stepping {
		if stepCount > 0 {
			d.stateMu.Lock()
			d.stepCount--
			if d.stepCount <= 0 {
				d.stepping = false
				d.paused = true
			}
			d.stateMu.Unlock()
			return true
		}
	}

	// Check breakpoints
	return d.CheckBreakpoint(bank, offset)
}

// PushCallFrame pushes a function call frame onto the stack
func (d *Debugger) PushCallFrame(bank uint8, offset uint16, functionName string) {
	d.stackMu.Lock()
	defer d.stackMu.Unlock()

	frame := CallFrame{
		Bank:         bank,
		Offset:       offset,
		FunctionName: functionName,
	}
	d.callStack = append(d.callStack, frame)
}

// PopCallFrame pops a function call frame from the stack
func (d *Debugger) PopCallFrame() *CallFrame {
	d.stackMu.Lock()
	defer d.stackMu.Unlock()

	if len(d.callStack) == 0 {
		return nil
	}

	frame := d.callStack[len(d.callStack)-1]
	d.callStack = d.callStack[:len(d.callStack)-1]
	return &frame
}

// GetCallStack returns the current call stack
func (d *Debugger) GetCallStack() []CallFrame {
	d.stackMu.RLock()
	defer d.stackMu.RUnlock()

	result := make([]CallFrame, len(d.callStack))
	copy(result, d.callStack)
	return result
}

// SetVariable tracks a variable's state
func (d *Debugger) SetVariable(name string, info VariableInfo) {
	d.variablesMu.Lock()
	defer d.variablesMu.Unlock()
	d.variables[name] = info
}

// GetVariable returns a variable's state
func (d *Debugger) GetVariable(name string) (VariableInfo, bool) {
	d.variablesMu.RLock()
	defer d.variablesMu.RUnlock()
	info, exists := d.variables[name]
	return info, exists
}

// GetAllVariables returns all tracked variables
func (d *Debugger) GetAllVariables() map[string]VariableInfo {
	d.variablesMu.RLock()
	defer d.variablesMu.RUnlock()

	result := make(map[string]VariableInfo)
	for k, v := range d.variables {
		result[k] = v
	}
	return result
}

// ClearVariables clears all tracked variables
func (d *Debugger) ClearVariables() {
	d.variablesMu.Lock()
	defer d.variablesMu.Unlock()
	d.variables = make(map[string]VariableInfo)
}

// ClearBreakpoints clears all breakpoints
func (d *Debugger) ClearBreakpoints() {
	d.breakpointsMu.Lock()
	defer d.breakpointsMu.Unlock()
	d.breakpoints = make(map[string]*Breakpoint)
}

// ClearWatches clears all watch expressions
func (d *Debugger) ClearWatches() {
	d.watchesMu.Lock()
	defer d.watchesMu.Unlock()
	d.watches = make([]*WatchExpression, 0)
}
