package healthcheck

import (
	"context"
	"errors"
	"time"
)

// Option adds optional parameter for the Healthcheck
type Option func(*health)

// WithChecker adds a status checker that needs to be added as part of healthcheck. i.e database, cache or any external dependency
func WithChecker(name string, s Checker) Option {
	return func(h *health) {
		h.checkers[name] = &timeoutChecker{s}
	}
}

// WithObserver adds a status checker but it does not fail the entire status.
func WithObserver(name string, s Checker) Option {
	return func(h *health) {
		h.observers[name] = &timeoutChecker{s}
	}
}

// WithTimeout configures the global timeout for all individual checkers.
func WithTimeout(timeout time.Duration) Option {
	return func(h *health) {
		h.timeout = timeout
	}
}

type timeoutChecker struct {
	checker Checker
}

func (t *timeoutChecker) Check(ctx context.Context) error {
	checkerChan := make(chan error)
	go func() {
		checkerChan <- t.checker.Check(ctx)
	}()
	select {
	case err := <-checkerChan:
		return err
	case <-ctx.Done():
		return errors.New("max check time exceeded")
	}
}
