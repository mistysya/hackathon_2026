package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRouterReturnsJSONNotFoundWithRequestID(t *testing.T) {
	router := NewRouter(testLogger())
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	requestID := response.Header().Get(requestIDHeader)
	if requestID == "" {
		t.Fatal("response is missing request ID")
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "not_found" || body.Error.RequestID != requestID {
		t.Fatalf("unexpected error response: %+v", body)
	}
}

func TestRouterRecoversPanics(t *testing.T) {
	router := NewRouter(testLogger(), registrarFunc(func(router chi.Router) {
		router.Get("/panic", func(http.ResponseWriter, *http.Request) {
			panic("boom")
		})
	}))
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	var body errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "internal_error" || body.Error.RequestID == "" {
		t.Fatalf("unexpected error response: %+v", body)
	}
}

func TestLimitBody(t *testing.T) {
	router := NewRouter(testLogger(), registrarFunc(func(router chi.Router) {
		router.Post("/body", func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if err != nil {
				WriteError(w, r, http.StatusBadRequest, "invalid_body", "request body is too large")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}))
	request := httptest.NewRequest(http.MethodPost, "/body", bytes.NewReader(make([]byte, DefaultMaxBodyBytes+1)))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

type registrarFunc func(chi.Router)

func (fn registrarFunc) RegisterRoutes(router chi.Router) {
	fn(router)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
