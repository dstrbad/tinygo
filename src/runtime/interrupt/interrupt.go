// Package interrupt provides access to hardware interrupts. It provides a way
// to define interrupts and to enable/disable them.
package interrupt

// Interrupt provides direct access to hardware interrupts. You can configure
// this interrupt through this interface.
//
// Do not use the zero value of an Interrupt object. Instead, call New to obtain
// an interrupt handle.
type Interrupt struct {
	// Make this number unexported so it cannot be set directly. This provides
	// some encapsulation.
	num int
}

// New is a compiler intrinsic that creates a new Interrupt object. You may call
// it only once, and must pass constant parameters to it. That means that the
// interrupt ID must be a Go constant and that the handler must be a simple
// function: closures are not supported.
func New(id int, handler func(Interrupt)) Interrupt

type handle struct {
	handler func(Interrupt)
	Interrupt
}
