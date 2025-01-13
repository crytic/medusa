package concolictracer

type concolicStack struct {
	stack []*concolicVariable
}

func newConcolicStack() *concolicStack {
	return &concolicStack{
		stack: make([]*concolicVariable, 0),
	}
}

func (c *concolicStack) pushN(n int) {
	prefix := make([]*concolicVariable, n)
	c.stack = append(prefix, c.stack...)

	// evm stack limit
	if len(c.stack) > 1024 {
		c.stack = c.stack[:1023]
	}
}

func (c *concolicStack) popN(n int) {
	c.stack = c.stack[n:]
}

func (c *concolicStack) pushVariable(v *concolicVariable) {
	c.pushN(1)
	c.stack[0] = v
}

// swapN: n is zero-indexed
func (c *concolicStack) swapN(n int) {
	if n >= len(c.stack) || n <= 0 {
		panic("calling swapN for stack item that does not exist")
	}

	tmp := c.stack[0]
	c.stack[0] = c.stack[n]
	c.stack[n] = tmp
}

// dupeN: n is 1-indexed
func (c *concolicStack) dupeN(n int) {
	if n > len(c.stack) || n < 1 {
		panic("calling dupeN for stack item that does not exist")
	}

	copied := c.stack[n-1].copy()
	c.pushVariable(copied)
}

// getVariable: n is 1-indexed
func (c *concolicStack) getVariable(n int) *concolicVariable {
	if n > len(c.stack) || n < 1 {
		panic("calling get for stack item that does not exist")
	}
	return c.stack[n-1]
}
