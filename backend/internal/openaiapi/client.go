package openaiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Metadata struct {
	ResponseID string
	Model      string
}

type Request struct {
	Instructions string
	Input        string
	SchemaName   string
	Schema       json.RawMessage
	WebSearch    bool
}

type Client interface {
	StructuredResponse(context.Context, Request) ([]byte, Metadata, error)
}

// HTTPClient is a narrow Responses API adapter. The rest of the application
// never sees SDK or provider response types, making provider replacement and
// fixture testing straightforward.
type HTTPClient struct {
	config Config
	http   *http.Client
}

func NewHTTPClient(config Config, client *http.Client) (*HTTPClient, error) {
	config, err := config.Normalized()
	if err != nil {
		return nil, err
	}
	if !config.LiveReady() {
		return nil, fmt.Errorf("%w: API key and model are required", ErrConfiguration)
	}
	if client == nil {
		client = &http.Client{}
	}
	return &HTTPClient{config: config, http: client}, nil
}

func (c *HTTPClient) StructuredResponse(ctx context.Context, request Request) ([]byte, Metadata, error) {
	if c == nil {
		return nil, Metadata{}, fmt.Errorf("%w: nil client", ErrConfiguration)
	}
	if len(request.Schema) == 0 || strings.TrimSpace(request.SchemaName) == "" {
		return nil, Metadata{}, fmt.Errorf("%w: structured schema is required", ErrConfiguration)
	}
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	for attempt := 0; ; attempt++ {
		output, metadata, retryAfter, err := c.do(ctx, parent, request)
		if err == nil || attempt >= c.config.MaxRetries || !retryable(err) {
			return output, metadata, err
		}
		if retryAfter <= 0 {
			retryAfter = 100 * time.Millisecond
		}
		if deadline, ok := ctx.Deadline(); ok && time.Now().Add(retryAfter).After(deadline) {
			return nil, Metadata{}, err
		}
		timer := time.NewTimer(retryAfter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, Metadata{}, timeoutOrCancel(parent)
		case <-timer.C:
		}
	}
}

func retryable(err error) bool {
	var provider *ProviderError
	return errors.As(err, &provider) && (provider.Kind == ErrorTransient || provider.Kind == ErrorRateLimit)
}

// timeoutOrCancel distinguishes a caller cancellation, which must propagate so
// fallback never masks it, from an internal per-request provider timeout (the
// caller context is still live), which is a transient, fallback-eligible error.
func timeoutOrCancel(parent context.Context) error {
	if parent.Err() != nil {
		return parent.Err()
	}
	return &ProviderError{Kind: ErrorTransient}
}

func (c *HTTPClient) do(ctx, parent context.Context, request Request) ([]byte, Metadata, time.Duration, error) {
	body := map[string]any{
		"model":             c.config.Model,
		"instructions":      request.Instructions,
		"input":             request.Input,
		"store":             false,
		"max_output_tokens": c.config.MaxOutputTokens,
		"text": map[string]any{"format": map[string]any{
			"type": "json_schema", "name": request.SchemaName, "schema": json.RawMessage(request.Schema), "strict": true,
		}},
	}
	if request.WebSearch {
		body["tools"] = []map[string]string{{"type": "web_search"}}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, Metadata{}, 0, fmt.Errorf("%w: encode request", ErrConfiguration)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/responses", bytes.NewReader(raw))
	if err != nil {
		return nil, Metadata{}, 0, fmt.Errorf("%w: create request", ErrConfiguration)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(httpRequest)
	if err != nil {
		return nil, Metadata{}, 0, timeoutOrCancel(parent)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 2<<20)
	data, readErr := io.ReadAll(limited)
	if readErr != nil {
		return nil, Metadata{}, 0, &ProviderError{Kind: ErrorTransient, StatusCode: response.StatusCode}
	}
	retryAfter := parseRetryAfter(response.Header.Get("Retry-After"))
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, Metadata{}, retryAfter, classifyError(response.StatusCode, data, retryAfter)
	}
	text, metadata, err := decodeResponse(data)
	return text, metadata, retryAfter, err
}

func parseRetryAfter(raw string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func classifyError(status int, data []byte, retryAfter time.Duration) error {
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(data, &payload)
	kind := ErrorUnsupported
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		kind = ErrorAuthentication
	case status == http.StatusTooManyRequests && (strings.Contains(payload.Error.Code, "quota") || strings.Contains(payload.Error.Code, "billing") || strings.Contains(payload.Error.Code, "insufficient")):
		kind = ErrorQuota
	case status == http.StatusTooManyRequests:
		kind = ErrorRateLimit
	case status >= 500:
		kind = ErrorTransient
	}
	return &ProviderError{Kind: kind, StatusCode: status, Code: payload.Error.Code, RetryAfter: retryAfter}
}

func decodeResponse(data []byte) ([]byte, Metadata, error) {
	var response struct {
		ID     string `json:"id"`
		Model  string `json:"model"`
		Status string `json:"status"`
		Error  *struct {
			Code string `json:"code"`
		} `json:"error"`
		Output []struct {
			Type    string `json:"type"`
			Refusal string `json:"refusal"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, Metadata{}, &ProviderError{Kind: ErrorOutput}
	}
	metadata := Metadata{ResponseID: response.ID, Model: response.Model}
	if response.Error != nil {
		return nil, metadata, &ProviderError{Kind: ErrorOutput, Code: response.Error.Code}
	}
	if response.Status != "completed" {
		return nil, metadata, &ProviderError{Kind: ErrorIncomplete}
	}
	for _, item := range response.Output {
		if item.Refusal != "" {
			return nil, metadata, &ProviderError{Kind: ErrorRefusal}
		}
		for _, content := range item.Content {
			if content.Refusal != "" {
				return nil, metadata, &ProviderError{Kind: ErrorRefusal}
			}
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				return []byte(content.Text), metadata, nil
			}
		}
	}
	return nil, metadata, &ProviderError{Kind: ErrorOutput}
}

var _ Client = (*HTTPClient)(nil)
