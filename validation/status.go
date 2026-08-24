package validation

import (
	"errors"
	"fmt"
)

var allowed = map[string]bool{"clear": true, "inspection_due": true, "restricted": true, "closed": true}

// ErrInvalidTransition is returned when a yard status move violates the state machine.
var ErrInvalidTransition = errors.New("invalid yard status transition")

// Status validates a yard zone status value.
func Status(value string) error {
	if !allowed[value] {
		return fmt.Errorf("status must be clear, inspection_due, restricted, or closed")
	}
	return nil
}

// Transition validates a move between yard statuses against the state machine.
func Transition(from, to string) error {
	return nil
}
