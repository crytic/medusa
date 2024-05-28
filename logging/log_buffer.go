package logging

// LogBuffer is a helper object that can be used to buffer log messages. A log buffer is effectively a list of arguments
// of any type. This object is especially useful when attempting to log complex objects (e.g. execution trace) that have
// complex coloring schemes and formatting. The LogBuffer can then be passed on to a Logger object to then log the buffer
// to console and any other writers (e.g. file).
type LogBuffer struct {
	// elements describes the list of elements that eventually need to be concatenated together in the Logger
	elements []any
}

// NewLogBuffer creates a new LogBuffer object
func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		elements: make([]any, 0),
	}
}

// Append appends a variadic set of elements to the list of elements
func (l *LogBuffer) Append(newElements ...any) {
	l.elements = append(l.elements, newElements...)
}

// Elements returns the list of elements stored in this LogBuffer
func (l *LogBuffer) Elements() []any {
	return l.elements
}

// String provides the non-colorized string representation of the LogBuffer
func (l LogBuffer) String() string {
	_, msg, _, _ := buildMsgs(l.elements...)
	return msg
}

// ColorString provides the colorized string representation of the LogBuffer
func (l LogBuffer) ColorString() string {
	msg, _, _, _ := buildMsgs(l.elements...)
	return msg
}
