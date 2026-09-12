package openaiapi

import (
	"errors"
	"fmt"
	"time"
)

var ErrConfiguration = errors.New("openai configuration error")

type ErrorKind string

const (
	ErrorAuthentication ErrorKind = "authentication"
	ErrorQuota          ErrorKind = "quota"
	ErrorRateLimit      ErrorKind = "rate_limit"
	ErrorTransient      ErrorKind = "transient"
	ErrorUnsupported    ErrorKind = "unsupported"
	ErrorRefusal        ErrorKind = "refusal"
	ErrorIncomplete     ErrorKind = "incomplete"
	ErrorOutput         ErrorKind = "output"
)

// ProviderError exposes only safe classification metadata. It deliberately
// omits response bodies because these can include prompt-derived content.
type ProviderError struct {
	Kind       ErrorKind
	StatusCode int
	Code       string
	RetryAfter time.Duration
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "openai provider error"
	}
	if e.Code != "" {
		return fmt.Sprintf("openai provider %s (%d, %s)", e.Kind, e.StatusCode, e.Code)
	}
	return fmt.Sprintf("openai provider %s (%d)", e.Kind, e.StatusCode)
}

func IsFallbackEligible(err error) bool {
	var provider *ProviderError
	return errors.As(err, &provider)
}
