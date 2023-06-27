package logging

// LogBuffer is a helper object that can be used to buffer log messages. A log buffer is effectively a list of arguments
// of any type. This object is especially useful when attempting to log complex objects (e.g. execution trace) that have
// complex coloring schemes and formatting. The LogBuffer can then be passed on to a Logger object to then log the buffer
// to console and any other writers (e.g. file).
type LogBuffer struct {
	// args describes the list of arguments that eventually need to be concatenated together in the Logger
	args []any
}

// NewLogBuffer creates a new LogBuffer object
func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		args: make([]any, 0),
	}
}

// Append appends a variadic set of arguments to the list of arguments
func (l *LogBuffer) Append(newArgs ...any) {
	l.args = append(l.args, newArgs...)
}

// Args returns the list of arguments stored in this LogBuffer
func (l *LogBuffer) Args() []any {
	return l.args
}

// String provides the non-colorized string representation of the LogBuffer
func (l LogBuffer) String() string {
	_, msg, _ := buildMsgs(l.args)
	return msg
}
