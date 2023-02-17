package types

// GenericHookFunc defines a basic function that takes no arguments and returns none, to be used as a hook during
// execution.
type GenericHookFunc func()

// GenericHookFuncs wraps a list of GenericHookFunc items. It provides operations to push new hooks into the top or
// bottom of the list and execute in forward or backward directions.
type GenericHookFuncs []GenericHookFunc

// Execute takes each hook in the list and executes it in the given order, indicating whether they should be cleared.
// If the forward flag is provided, hooks are executed from index 0 to the end, otherwise they are executed in reverse.
// If the clear flag is provided, it sets the hooks to empty/nil afterwards on the immediate pointer.
// Copies of the pointer should be considered carefully here.
func (t *GenericHookFuncs) Execute(forward bool, clear bool) {
	// If the hooks aren't set yet, do nothing.
	if t == nil {
		return
	}

	// Otherwise execute every hook in the order specified.
	if forward {
		for i := 0; i < len(*t); i++ {
			(*t)[i]()
		}
	} else {
		for i := len(*t) - 1; i >= 0; i-- {
			(*t)[i]()
		}
	}

	// If we're set to clear this, set the hook to nil
	if clear {
		*t = nil
	}
}

// Push pushes a provided hook onto the stack (end of the list).
func (t *GenericHookFuncs) Push(f GenericHookFunc) {
	// Push the provided hook onto the stack.
	*t = append(*t, f)
}
