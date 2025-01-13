package concolictracer

import "github.com/mitchellh/go-z3"

type concolicVariable struct {
	variable *z3.AST
}

func (c *concolicVariable) copy() *concolicVariable {
	return &concolicVariable{
		variable: c.variable,
	}
}
