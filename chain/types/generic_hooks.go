package types

import "golang.org/x/exp/slices"

// GenericHookFunc defines a basic function that takes no arguments and returns none, to be used as a hook during
// execution.
type GenericHookFunc func()

// GenericHookFuncs wraps a list of GenericHookFunc items. It provides operations to push new hooks into the top or
// bottom of the list and execute in forward or backward directions.
type GenericHookFuncs []GenericHookFunc

// Execute takes a list of hooks, clones it, clears the original if requested, and executes all hook handlers from the
// clone. This allows for hook handlers to add more hooks to the original list, to be triggered on next Execute.
//
// If the forward flag is provided, hooks are executed from index 0 to the end, otherwise they are executed in reverse.
// If the clear flag is provided, it sets the hooks to empty/nil afterwards on the immediate pointer.
// Copies of the pointer should be considered carefully here.
func (t *GenericHookFuncs) Execute(forward bool, clear bool) {
	// If the hooks aren't set yet, do nothing.
	if t == nil {
		return
	}

	// Create a copy of our array in case it's modified while hooks execute.
	tCopy := slices.Clone(*t)

	// If we're set to clear this, set the hook to nil
	if clear {
		*t = nil
	}

	// Otherwise execute every hook in the order specified.
	// We make a copy so adding to hooks while executing one does not result in an infinite loop.
	if forward {
		for i := 0; i < len(tCopy); i++ {
			(tCopy)[i]()
		}
	} else {
		for i := len(tCopy) - 1; i >= 0; i-- {
			(tCopy)[i]()
		}
	}
}

// Push pushes a provided hook onto the stack (end of the list).
func (t *GenericHookFuncs) Push(f GenericHookFunc) {
	// Push the provided hook onto the stack.
	*t = append(*t, f)
}
