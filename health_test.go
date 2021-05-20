package healthcheck_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hoshsadiq/go-healthcheck"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestHealth_CheckHealth(t *testing.T) {
	tests := []struct {
		name          string
		args          []healthcheck.Option
		statusCode    int
		errorMessages map[string]string
	}{
		{
			name:          "returns 200 status if no errors",
			statusCode:    http.StatusOK,
			errorMessages: map[string]string{},
		},
		{
			name: "returns 503 status if errors",
			args: []healthcheck.Option{
				healthcheck.WithChecker("database", healthcheck.CheckerFunc(func(ctx context.Context) error {
					return fmt.Errorf("connection to db timed out")
				})),
				healthcheck.WithChecker("testService", healthcheck.CheckerFunc(func(ctx context.Context) error {
					return fmt.Errorf("connection refused")
				})),
			},
			statusCode: http.StatusServiceUnavailable,
			errorMessages: map[string]string{
				"database":    "connection to db timed out",
				"testService": "connection refused",
			},
		},
		{
			name: "returns 503 status if checkers timeout",
			args: []healthcheck.Option{
				healthcheck.WithTimeout(1 * time.Millisecond),
				healthcheck.WithChecker("database", healthcheck.CheckerFunc(func(ctx context.Context) error {
					time.Sleep(10 * time.Millisecond)
					return nil
				})),
			},
			statusCode: http.StatusServiceUnavailable,
			errorMessages: map[string]string{
				"database": "max check time exceeded",
			},
		},
		{
			name: "returns 200 status if errors are observable",
			args: []healthcheck.Option{
				healthcheck.WithObserver("observableService", healthcheck.CheckerFunc(func(ctx context.Context) error {
					return fmt.Errorf("i fail but it is okay")
				})),
			},
			statusCode: http.StatusOK,
			errorMessages: map[string]string{
				"observableService": "i fail but it is okay",
			},
		},
		{
			name: "returns 503 status if errors with observable fails",
			args: []healthcheck.Option{
				healthcheck.WithObserver("database", healthcheck.CheckerFunc(func(ctx context.Context) error {
					return fmt.Errorf("connection to db timed out")
				})),
				healthcheck.WithChecker("testService", healthcheck.CheckerFunc(func(ctx context.Context) error {
					return fmt.Errorf("connection refused")
				})),
			},
			statusCode: http.StatusServiceUnavailable,
			errorMessages: map[string]string{
				"database":    "connection to db timed out",
				"testService": "connection refused",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, errorMessages := healthcheck.NewService(tt.args...).CheckHealth(context.TODO())
			if code != tt.statusCode {
				t.Errorf("expected code %d, got %d", tt.statusCode, code)
			}
			if !reflect.DeepEqual(errorMessages, tt.errorMessages) {
				t.Errorf("NewHandlerFunc() = %v, want %v", errorMessages, tt.errorMessages)
			}
		})
	}
}

func TestHealth_HandlerFunc(t *testing.T) {
	tests := []struct {
		name       string
		args       []healthcheck.Option
		statusCode int
		response   healthcheck.HealthResponse
	}{
		{
			name:          "returns 200 status if no errors",
			statusCode:    http.StatusOK,
			response: healthcheck.HealthResponse{
				Status: http.StatusText(http.StatusOK),
			},
		},
		{
			name: "returns 503 status if errors",
			args: []healthcheck.Option{
				healthcheck.WithChecker("database", healthcheck.CheckerFunc(func(ctx context.Context) error {
					return fmt.Errorf("connection to db timed out")
				})),
				healthcheck.WithChecker("testService", healthcheck.CheckerFunc(func(ctx context.Context) error {
					return fmt.Errorf("connection refused")
				})),
			},
			statusCode: http.StatusServiceUnavailable,
			response: healthcheck.HealthResponse{
				Status: http.StatusText(http.StatusServiceUnavailable),
				Errors: map[string]string{
					"database":    "connection to db timed out",
					"testService": "connection refused",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "http://localhost/health", nil)
			if err != nil {
				t.Errorf("Failed to create request.")
			}
			healthcheck.NewService(tt.args...).HandlerFunc()(res, req)
			if res.Code != tt.statusCode {
				t.Errorf("expected code %d, got %d", tt.statusCode, res.Code)
			}
			var respBody healthcheck.HealthResponse
			if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
				t.Fatal("failed to parse the body")
			}
			if !reflect.DeepEqual(respBody, tt.response) {
				t.Errorf("NewHandlerFunc() = %v, want %v", respBody, tt.response)
			}
		})
	}
}
