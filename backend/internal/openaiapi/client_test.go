package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStructuredResponseSendsSafeResponsesContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["store"] != false || body["model"] != "gpt-test" {
			t.Fatalf("body = %#v", body)
		}
		text := body["text"].(map[string]any)["format"].(map[string]any)
		if text["type"] != "json_schema" || text["strict"] != true {
			t.Fatalf("format = %#v", text)
		}
		if tools := body["tools"].([]any); len(tools) != 1 || tools[0].(map[string]any)["type"] != "web_search" {
			t.Fatalf("tools = %#v", tools)
		}
		_, _ = w.Write([]byte(`{"id":"resp_1","model":"gpt-test","status":"completed","output":[{"type":"web_search_call"},{"type":"message","content":[{"type":"output_text","text":"{\"ok\":true}"}]}]}`))
	}))
	defer server.Close()
	client, err := NewHTTPClient(Config{APIKey: "test-key", Model: "gpt-test", BaseURL: server.URL + "/v1", Timeout: time.Second, MaxRetries: 1}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	raw, metadata, err := client.StructuredResponse(context.Background(), Request{SchemaName: "test", Schema: json.RawMessage(`{"type":"object"}`), WebSearch: true})
	if err != nil || string(raw) != `{"ok":true}` || metadata.ResponseID != "resp_1" {
		t.Fatalf("raw=%s metadata=%+v err=%v", raw, metadata, err)
	}
}

func TestStructuredResponseClassifiesAndDoesNotRetryQuota(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"insufficient_quota"}}`))
	}))
	defer server.Close()
	client, err := NewHTTPClient(Config{APIKey: "key", Model: "model", BaseURL: server.URL, Timeout: time.Second, MaxRetries: 1}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.StructuredResponse(context.Background(), Request{SchemaName: "test", Schema: json.RawMessage(`{}`)})
	var provider *ProviderError
	if !IsFallbackEligible(err) || !strings.Contains(err.Error(), "quota") || !asProvider(err, &provider) || provider.Kind != ErrorQuota || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestStructuredResponseDoesNotFallbackCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	client, err := NewHTTPClient(Config{APIKey: "key", Model: "model", BaseURL: server.URL, Timeout: time.Second}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = client.StructuredResponse(ctx, Request{SchemaName: "test", Schema: json.RawMessage(`{}`)})
	if err != context.Canceled {
		t.Fatalf("err = %v", err)
	}
}

func TestStructuredResponsePropagatesCancellationDuringBodyRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := NewHTTPClient(Config{APIKey: "key", Model: "model", BaseURL: "https://example.test", Timeout: time.Second}, &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: cancelingReadCloser{cancel: cancel}}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.StructuredResponse(ctx, Request{SchemaName: "test", Schema: json.RawMessage(`{}`)})
	if !errors.Is(err, context.Canceled) || IsFallbackEligible(err) {
		t.Fatalf("err = %v, want non-fallback context.Canceled", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type cancelingReadCloser struct{ cancel context.CancelFunc }

func (body cancelingReadCloser) Read([]byte) (int, error) {
	body.cancel()
	return 0, errors.New("simulated body read failure")
}

func (cancelingReadCloser) Close() error { return nil }

func asProvider(err error, target **ProviderError) bool { return errors.As(err, target) }

func TestStructuredResponseFallsBackOnProviderTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"id":"resp","model":"model","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{}"}]}]}`))
	}))
	defer server.Close()
	client, err := NewHTTPClient(Config{APIKey: "key", Model: "model", BaseURL: server.URL, Timeout: 20 * time.Millisecond}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.StructuredResponse(context.Background(), Request{SchemaName: "test", Schema: json.RawMessage(`{}`)})
	var provider *ProviderError
	if !IsFallbackEligible(err) || !asProvider(err, &provider) || provider.Kind != ErrorTransient {
		t.Fatalf("provider timeout err=%v want fallback-eligible transient", err)
	}
}
