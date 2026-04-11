// Package nahook provides shared types, errors, and an internal HTTP client
// for the Nahook webhook platform SDKs.
package nahook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL  = "https://api.nahook.com"
	DefaultTimeout  = 30 * time.Second
	DefaultRetries  = 0
	sdkVersion      = "0.1.0"
	userAgent       = "nahook-go/" + sdkVersion
	baseDelayMs     = 500
	maxDelayMs      = 10_000
)

// regionBaseURLs maps the region slug embedded in API keys to base URLs.
var regionBaseURLs = map[string]string{
	"us": "https://us.api.nahook.com",
	"eu": "https://eu.api.nahook.com",
	"ap": "https://ap.api.nahook.com",
}

// ResolveBaseURL extracts the region slug from an nhk_ API key and returns the
// regional base URL. Falls back to DefaultBaseURL for legacy keys.
func ResolveBaseURL(apiKey string) string {
	if len(apiKey) >= 7 && apiKey[:4] == "nhk_" && apiKey[6] == '_' {
		slug := apiKey[4:6]
		if u, ok := regionBaseURLs[slug]; ok {
			return u
		}
	}
	return DefaultBaseURL
}

// ── Error types ─────────────────────────────────────────────────────────────

// APIError represents an error response from the Nahook API.
type APIError struct {
	Status     int
	Code       string
	Message    string
	RetryAfter *int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("nahook: API error %d (%s): %s", e.Status, e.Code, e.Message)
}

// IsRetryable returns true for 5xx and 429 status codes.
func (e *APIError) IsRetryable() bool {
	return e.Status >= 500 || e.Status == 429
}

// IsAuthError returns true for 401 or 403 with code "token_disabled".
func (e *APIError) IsAuthError() bool {
	return e.Status == 401 || (e.Status == 403 && e.Code == "token_disabled")
}

// IsNotFound returns true for 404 status codes.
func (e *APIError) IsNotFound() bool {
	return e.Status == 404
}

// IsRateLimited returns true for 429 status codes.
func (e *APIError) IsRateLimited() bool {
	return e.Status == 429
}

// IsValidationError returns true for 400 status codes.
func (e *APIError) IsValidationError() bool {
	return e.Status == 400
}

// NetworkError represents a network-level failure where no HTTP response was received.
type NetworkError struct {
	Cause error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("nahook: network error: %v", e.Cause)
}

func (e *NetworkError) Unwrap() error {
	return e.Cause
}

// TimeoutError represents a request that exceeded the configured timeout.
type TimeoutError struct {
	TimeoutMs int
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("nahook: request timed out after %dms", e.TimeoutMs)
}

// ── Shared types ────────────────────────────────────────────────────────────

// SendOptions configures a direct send to a specific endpoint.
type SendOptions struct {
	Payload        map[string]interface{} `json:"payload"`
	IdempotencyKey string                 `json:"idempotencyKey,omitempty"`
}

// SendResult is the response from a direct send.
type SendResult struct {
	DeliveryID     string `json:"deliveryId"`
	IdempotencyKey string `json:"idempotencyKey"`
	Status         string `json:"status"`
}

// TriggerOptions configures a fan-out trigger by event type.
type TriggerOptions struct {
	Payload  map[string]interface{} `json:"payload"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

// TriggerResult is the response from a trigger.
type TriggerResult struct {
	EventTypeID string   `json:"eventTypeId"`
	DeliveryIDs []string `json:"deliveryIds"`
	Status      string   `json:"status"`
}

// SendBatchItem represents one item in a batch send.
type SendBatchItem struct {
	EndpointID     string                 `json:"endpointId"`
	Payload        map[string]interface{} `json:"payload"`
	IdempotencyKey string                 `json:"idempotencyKey,omitempty"`
}

// TriggerBatchItem represents one item in a batch trigger.
type TriggerBatchItem struct {
	EventType string                 `json:"eventType"`
	Payload   map[string]interface{} `json:"payload"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// BatchResultItem is the result for one item in a batch operation.
type BatchResultItem struct {
	Index          int              `json:"index"`
	DeliveryID     string           `json:"deliveryId,omitempty"`
	IdempotencyKey string           `json:"idempotencyKey,omitempty"`
	EventTypeID    string           `json:"eventTypeId,omitempty"`
	DeliveryIDs    []string         `json:"deliveryIds,omitempty"`
	Status         string           `json:"status,omitempty"`
	Error          *BatchItemError  `json:"error,omitempty"`
}

// BatchItemError is an error for a specific item in a batch.
type BatchItemError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BatchResult is the response from a batch operation.
type BatchResult struct {
	Items []BatchResultItem `json:"items"`
}

// ── Management types ────────────────────────────────────────────────────────

// Endpoint represents a webhook endpoint.
type Endpoint struct {
	ID          string                 `json:"id"`
	URL         string                 `json:"url"`
	Description *string                `json:"description"`
	IsActive    bool                   `json:"isActive"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Secret      string                 `json:"secret,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// EventType represents a registered event type.
type EventType struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"createdAt"`
}

// Application represents a developer application.
type Application struct {
	ID         string            `json:"id"`
	ExternalID *string           `json:"externalId"`
	Name       string            `json:"name"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  string            `json:"createdAt"`
	UpdatedAt  string            `json:"updatedAt"`
}

// Subscription represents an event type subscription on an endpoint.
type Subscription struct {
	ID          string `json:"id"`
	EndpointID  string `json:"endpointId"`
	EventTypeID string `json:"eventTypeId"`
	CreatedAt   string `json:"createdAt"`
}

// PortalSession represents a portal session for developer self-service.
type PortalSession struct {
	URL       string `json:"url"`
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
}

// ListResult wraps a list response.
type ListResult[T any] struct {
	Data []T `json:"data"`
}

// ListOptions configures pagination for list operations.
type ListOptions struct {
	Limit  *int
	Offset *int
}

// CreateEndpointOptions configures a new endpoint.
type CreateEndpointOptions struct {
	URL          string                 `json:"url"`
	Type         string                 `json:"type,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	AuthUsername string                 `json:"authUsername,omitempty"`
	AuthPassword string                 `json:"authPassword,omitempty"`
}

// UpdateEndpointOptions configures an endpoint update.
type UpdateEndpointOptions struct {
	URL         *string           `json:"url,omitempty"`
	Description *string           `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	IsActive    *bool             `json:"isActive,omitempty"`
}

// CreateEventTypeOptions configures a new event type.
type CreateEventTypeOptions struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateEventTypeOptions configures an event type update.
type UpdateEventTypeOptions struct {
	Description *string `json:"description,omitempty"`
}

// CreateApplicationOptions configures a new application.
type CreateApplicationOptions struct {
	Name       string            `json:"name"`
	ExternalID string            `json:"externalId,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// UpdateApplicationOptions configures an application update.
type UpdateApplicationOptions struct {
	Name     *string           `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CreateSubscriptionOptions configures a new subscription.
type CreateSubscriptionOptions struct {
	EventTypeID string `json:"eventTypeId"`
}

// CreatePortalSessionOptions configures a new portal session.
type CreatePortalSessionOptions struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ── Internal HTTP client ────────────────────────────────────────────────────

// HTTPClient is the internal HTTP client shared by the client and management packages.
// Internal: not part of the public API.
type HTTPClient struct {
	token   string
	baseURL string
	timeout time.Duration
	retries int
	http    *http.Client
}

// HTTPClientConfig configures the internal HTTP client.
// Internal: not part of the public API.
type HTTPClientConfig struct {
	Token   string
	BaseURL string
	Timeout time.Duration
	Retries int
}

// NewHTTPClient creates a new internal HTTP client.
// Internal: not part of the public API.
func NewHTTPClient(cfg HTTPClientConfig) *HTTPClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = ResolveBaseURL(cfg.Token)
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	return &HTTPClient{
		token:   cfg.Token,
		baseURL: baseURL,
		timeout: timeout,
		retries: cfg.Retries,
		http:    &http.Client{Timeout: timeout},
	}
}

// RequestOptions describes an HTTP request to the Nahook API.
// Internal: not part of the public API.
type RequestOptions struct {
	Method string
	Path   string
	Body   interface{}
	Query  map[string]string
}

// Request performs an HTTP request and decodes the JSON response into result.
// For DELETE (204) responses, result may be nil.
func (c *HTTPClient) Request(ctx context.Context, opts RequestOptions, result interface{}) error {
	resp, err := c.executeWithRetry(ctx, opts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// RequestWithStatus performs an HTTP request and returns the status code along with the decoded body.
func (c *HTTPClient) RequestWithStatus(ctx context.Context, opts RequestOptions, result interface{}) (int, error) {
	resp, err := c.executeWithRetry(ctx, opts)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func (c *HTTPClient) executeWithRetry(ctx context.Context, opts RequestOptions) (*http.Response, error) {
	reqURL := c.buildURL(opts.Path, opts.Query)

	var bodyBytes []byte
	if opts.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(opts.Body)
		if err != nil {
			return nil, &NetworkError{Cause: fmt.Errorf("failed to marshal request body: %w", err)}
		}
	}

	var lastErr error

	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			var retryAfterMs int
			if apiErr, ok := lastErr.(*APIError); ok && apiErr.RetryAfter != nil {
				retryAfterMs = *apiErr.RetryAfter * 1000
			}
			delay := calculateDelay(attempt-1, retryAfterMs)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, opts.Method, reqURL, bodyReader)
		if err != nil {
			return nil, &NetworkError{Cause: err}
		}

		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", userAgent)
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				lastErr = &TimeoutError{TimeoutMs: int(c.timeout.Milliseconds())}
			} else {
				lastErr = &NetworkError{Cause: err}
			}
			if attempt < c.retries {
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		apiErr := c.parseError(resp)
		resp.Body.Close()

		if attempt < c.retries && apiErr.IsRetryable() {
			lastErr = apiErr
			continue
		}
		return nil, apiErr
	}

	return nil, lastErr
}

func (c *HTTPClient) parseError(resp *http.Response) *APIError {
	var retryAfter *int
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			retryAfter = &secs
		}
	}

	var body struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	code := "unknown"
	message := resp.Status

	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil && body.Error != nil {
		if body.Error.Code != "" {
			code = body.Error.Code
		}
		if body.Error.Message != "" {
			message = body.Error.Message
		}
	}

	return &APIError{
		Status:     resp.StatusCode,
		Code:       code,
		Message:    message,
		RetryAfter: retryAfter,
	}
}

func (c *HTTPClient) buildURL(path string, query map[string]string) string {
	u := c.baseURL + path
	if len(query) > 0 {
		params := url.Values{}
		for k, v := range query {
			if v != "" {
				params.Set(k, v)
			}
		}
		if encoded := params.Encode(); encoded != "" {
			u += "?" + encoded
		}
	}
	return u
}

// calculateDelay computes the retry delay with exponential backoff and full jitter.
func calculateDelay(attempt int, retryAfterMs int) time.Duration {
	if retryAfterMs > 0 {
		return time.Duration(retryAfterMs) * time.Millisecond
	}
	exponential := math.Min(float64(maxDelayMs), float64(baseDelayMs)*math.Pow(2, float64(attempt)))
	jittered := exponential * rand.Float64()
	return time.Duration(jittered) * time.Millisecond
}

// PathEncode encodes a path segment for use in URLs.
// Internal: not part of the public API.
func PathEncode(s string) string {
	return url.PathEscape(s)
}
