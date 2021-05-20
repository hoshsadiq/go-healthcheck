package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Service interface {
	CheckHealth(ctx context.Context) (httpError int, errorMsgs map[string]string)
	Handler() http.Handler
	HandlerFunc() http.HandlerFunc
}

type response struct {
	Status string            `json:"status,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
}

type health struct {
	checkers  map[string]Checker
	observers map[string]Checker
	timeout   time.Duration
}

// NewService returns a new Service instance
func NewService(opts ...Option) Service {
	h := &health{
		checkers:  make(map[string]Checker),
		observers: make(map[string]Checker),
		timeout:   30 * time.Second,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Handler returns an http.Handler
func (h *health) Handler() http.Handler {
	return h
}

// HandlerFunc returns an http.HandlerFunc to mount the API implementation at a specific route
func (h *health) HandlerFunc() http.HandlerFunc {
	return h.ServeHTTP
}

func (h *health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code, errorMsgs := h.CheckHealth(r.Context())

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response{
		Status: http.StatusText(code),
		Errors: errorMsgs,
	})
}

// CheckHealth does the heavy lifting of checking all the services based on the configured Checker functions
func (h *health) CheckHealth(ctx context.Context) (int, map[string]string) {
	nCheckers := len(h.checkers) + len(h.observers)

	code := http.StatusOK
	errorMsgs := make(map[string]string, nCheckers)

	cancel := func() {}
	if h.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
	}
	defer cancel()

	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(nCheckers)

	for key, checker := range h.checkers {
		go func(key string, checker Checker, ctx context.Context) {
			if err := checker.Check(ctx); err != nil {
				mutex.Lock()
				errorMsgs[key] = err.Error()
				code = http.StatusServiceUnavailable
				mutex.Unlock()
			}
			wg.Done()
		}(key, checker, ctx)
	}
	for key, observer := range h.observers {
		go func(key string, observer Checker, ctx context.Context) {
			if err := observer.Check(ctx); err != nil {
				mutex.Lock()
				errorMsgs[key] = err.Error()
				mutex.Unlock()
			}
			wg.Done()
		}(key, observer, ctx)
	}

	wg.Wait()

	return code, errorMsgs
}
