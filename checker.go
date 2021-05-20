package healthcheck

import "context"

// Checker checks the status of the dependency and returns error.
// In case the dependency is working as expected, return nil.
type Checker interface {
	Check(ctx context.Context) error
}

// CheckerFunc is a convenience type to create functions that implement the Checker interface.
type CheckerFunc func(ctx context.Context) error

// Check Implements the Checker interface to allow for any func() error method
// to be passed as a Checker
func (c CheckerFunc) Check(ctx context.Context) error {
	return c(ctx)
}
