// Package nahook provides shared types, errors, and an internal HTTP client
// for the Nahook webhook platform SDKs.
package nahook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.nahook.com"
	DefaultTimeout = 30 * time.Second
	DefaultRetries = 0
	sdkVersion     = "0.3.0"
	userAgent      = "nahook-go/" + sdkVersion
	baseDelayMs    = 500
	maxDelayMs     = 10_000
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
	Index          int             `json:"index"`
	DeliveryID     string          `json:"deliveryId,omitempty"`
	IdempotencyKey string          `json:"idempotencyKey,omitempty"`
	EventTypeID    string          `json:"eventTypeId,omitempty"`
	DeliveryIDs    []string        `json:"deliveryIds,omitempty"`
	Status         string          `json:"status,omitempty"`
	Error          *BatchItemError `json:"error,omitempty"`
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
	// MaxEndpoints is the maximum number of endpoints this application may
	// have (disabled endpoints count). nil means unlimited.
	MaxEndpoints *int `json:"maxEndpoints"`
	// ShowEventTypes reports whether the Developer Portal exposes the
	// event-type catalog to this application.
	ShowEventTypes bool   `json:"showEventTypes"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// Subscription represents an event type subscription on an endpoint.
type Subscription struct {
	ID            string `json:"id"`
	EventTypeID   string `json:"eventTypeId"`
	EventTypeName string `json:"eventTypeName"`
	CreatedAt     string `json:"createdAt"`
}

// SubscribeResult is the response from subscribing an endpoint to event types.
type SubscribeResult struct {
	Subscribed int `json:"subscribed"`
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
	// EnvironmentID is optional. Public id (e.g. "env_abc123") of the environment
	// to scope this endpoint. If omitted, the workspace's default environment is used.
	EnvironmentID string `json:"environmentId,omitempty"`
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
	// MaxEndpoints caps how many endpoints this application may have
	// (disabled endpoints count). 0 makes the application read-only.
	// nil (omitted) means unlimited.
	MaxEndpoints *int `json:"maxEndpoints,omitempty"`
	// ShowEventTypes controls whether the Developer Portal exposes the
	// event-type catalog. nil (omitted) defaults to true.
	ShowEventTypes *bool `json:"showEventTypes,omitempty"`
}

// UpdateApplicationOptions configures an application update.
type UpdateApplicationOptions struct {
	Name     *string           `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	// MaxEndpoints is tri-state: leave nil to keep the current cap,
	// IntNull() to clear it (unlimited), or IntValue(n) to set it.
	MaxEndpoints *NullableInt `json:"maxEndpoints,omitempty"`
	// ShowEventTypes is omitted (unchanged) when nil.
	ShowEventTypes *bool `json:"showEventTypes,omitempty"`
}

// NullableInt is a JSON field that marshals as either a number or an explicit
// null. PATCH fields typed *NullableInt are tri-state: a nil pointer is
// omitted from the body entirely (leave unchanged), IntNull() marshals as
// null (clear), and IntValue(n) marshals as n (set).
type NullableInt struct {
	// Value is the number to send; nil marshals as JSON null.
	Value *int
}

// IntValue returns a NullableInt carrying v.
func IntValue(v int) *NullableInt { return &NullableInt{Value: &v} }

// IntNull returns a NullableInt that marshals as explicit JSON null — on
// UpdateApplicationOptions.MaxEndpoints this clears the cap (unlimited).
func IntNull() *NullableInt { return &NullableInt{} }

// MarshalJSON implements json.Marshaler.
func (n NullableInt) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *NullableInt) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

// CreateSubscriptionOptions configures a new subscription (bulk).
type CreateSubscriptionOptions struct {
	EventTypeIDs []string `json:"eventTypeIds"`
}

// CreatePortalSessionOptions configures a new portal session.
type CreatePortalSessionOptions struct {
	Metadata         map[string]string `json:"metadata,omitempty"`
	Role             string            `json:"role,omitempty"`
	ExpiresInMinutes int               `json:"expiresInMinutes,omitempty"`
}

// Environment represents a workspace environment.
type Environment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IsDefault bool   `json:"isDefault"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// CreateEnvironmentOptions configures a new environment.
type CreateEnvironmentOptions struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// UpdateEnvironmentOptions configures an environment update.
type UpdateEnvironmentOptions struct {
	Name *string `json:"name,omitempty"`
}

// EventTypeVisibility represents the visibility of an event type in an environment.
type EventTypeVisibility struct {
	EventTypeID   string `json:"eventTypeId"`
	EventTypeName string `json:"eventTypeName"`
	Published     bool   `json:"published"`
}

// SetVisibilityOptions configures event type visibility in an environment.
type SetVisibilityOptions struct {
	Published bool `json:"published"`
}

// PaginatedResult is a generic cursor-paginated read result. NextCursor is an
// opaque, server-encrypted token — pass it back verbatim on the next request,
// do not decode or modify it. Nil when there are no more pages.
type PaginatedResult[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"nextCursor"`
}

// DeliveryStatus enumerates the lifecycle states of a webhook delivery.
// Possible values: "pending", "delivering", "delivered", "scheduled_retry",
// "failed", "dead_letter".
type DeliveryStatus = string

// Delivery represents a webhook delivery's metadata (no payload body).
type Delivery struct {
	ID             string  `json:"id"`
	IdempotencyKey string  `json:"idempotencyKey"`
	EndpointID     string  `json:"endpointId"`
	Status         string  `json:"status"`
	TotalAttempts  int     `json:"totalAttempts"`
	FirstAttemptAt *string `json:"firstAttemptAt"`
	DeliveredAt    *string `json:"deliveredAt"`
	NextRetryAt    *string `json:"nextRetryAt"`
	HasPayload     bool    `json:"hasPayload"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// PayloadEnvelope is a flat tagged union describing the access state of a
// stored delivery payload. The Status field discriminates which other fields
// are populated:
//
//   - "available": Data and ContentType are set.
//   - "forbidden": workspace plan does not include payload storage.
//   - "processing": delivery still in flight, payload not yet written.
//   - "not_found": terminal delivery without a stored payload.
//   - "error": transient infrastructure failure reading the payload.
//
// Only "available" is a successful read; all four other statuses are
// returned with HTTP 200 — do not treat them as errors.
type PayloadEnvelope struct {
	Status      string          `json:"status"`
	Data        json.RawMessage `json:"data,omitempty"`
	ContentType string          `json:"contentType,omitempty"`
}

// DeliveryWithPayload is the response shape from get() — Delivery metadata,
// plus an optional payload envelope when the request included ?include=payload.
type DeliveryWithPayload struct {
	Delivery
	Payload *PayloadEnvelope `json:"payload,omitempty"`
}

// DeliveryAttempt represents one HTTP delivery attempt against an endpoint.
// Status is an opaque worker-emitted string (e.g. "success", "failed") — do
// not model it as an enum, the set may evolve.
type DeliveryAttempt struct {
	ID                 string  `json:"id"`
	AttemptNumber      int     `json:"attemptNumber"`
	Status             string  `json:"status"`
	ResponseStatusCode *int    `json:"responseStatusCode"`
	ResponseTimeMs     *int    `json:"responseTimeMs"`
	ErrorMessage       *string `json:"errorMessage"`
	CreatedAt          string  `json:"createdAt"`
}

// ListDeliveriesOptions configures a deliveries list query. All fields are
// optional. Cursor is an opaque token from a previous PaginatedResult's
// NextCursor — pass it through verbatim.
type ListDeliveriesOptions struct {
	Limit  *int
	Cursor string
	Status string
}

// GetDeliveryOptions configures a single delivery fetch. Set IncludePayload
// to true to request the payload envelope alongside metadata.
type GetDeliveryOptions struct {
	IncludePayload bool
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
	// ownsHTTPClient tracks whether the SDK constructed the *http.Client itself
	// (vs. receiving a caller-owned one via HTTPClientConfig.HTTPClient). Drives
	// Close() — we only drain the connection pool when we own the client.
	ownsHTTPClient bool
}

// HTTPClientConfig configures the internal HTTP client.
// Internal: not part of the public API.
type HTTPClientConfig struct {
	Token   string
	BaseURL string
	Timeout time.Duration
	Retries int
	// HTTPClient, when non-nil, is used verbatim and not mutated. The caller's
	// HTTPClient.Timeout governs request timeouts and is what TimeoutError.TimeoutMs
	// reports. When nil, the SDK builds a *http.Client with a tuned *http.Transport
	// (HTTP/2, TCP keep-alive, MaxIdleConnsPerHost = 50).
	HTTPClient *http.Client
}

// NewHTTPClient creates a new internal HTTP client.
// Internal: not part of the public API.
func NewHTTPClient(cfg HTTPClientConfig) *HTTPClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = ResolveBaseURL(cfg.Token)
	}
	baseURL = strings.TrimRight(baseURL, "/")

	var (
		httpClient     *http.Client
		timeout        time.Duration
		ownsHTTPClient bool
	)
	if cfg.HTTPClient != nil {
		// Caller-owned *http.Client: use verbatim. Caller's Timeout governs
		// request timeouts and is what TimeoutError.TimeoutMs reports.
		httpClient = cfg.HTTPClient
		timeout = cfg.HTTPClient.Timeout
		ownsHTTPClient = false
	} else {
		timeout = cfg.Timeout
		if timeout == 0 {
			timeout = DefaultTimeout
		}
		httpClient = buildDefaultHTTPClient(timeout)
		ownsHTTPClient = true
	}

	return &HTTPClient{
		token:          cfg.Token,
		baseURL:        baseURL,
		timeout:        timeout,
		retries:        cfg.Retries,
		http:           httpClient,
		ownsHTTPClient: ownsHTTPClient,
	}
}

// Close drains the SDK-owned *http.Transport's idle connection pool. Useful for
// clean test teardown, graceful shutdown, or explicit reset before recycling
// long-lived clients. Idempotent. No-op when a caller-owned *http.Client was
// supplied via HTTPClientConfig.HTTPClient — the caller owns that transport's
// lifecycle.
func (c *HTTPClient) Close() {
	if !c.ownsHTTPClient {
		return
	}
	c.http.CloseIdleConnections()
}

// HTTPClient returns the underlying *http.Client. Useful for callers who want
// to introspect or attach instrumentation. Mutating the returned client affects
// all subsequent SDK requests.
func (c *HTTPClient) HTTPClient() *http.Client {
	return c.http
}

// buildDefaultHTTPClient constructs the SDK's default *http.Client backed by a
// tuned *http.Transport. The pool sizing (MaxIdleConnsPerHost = 50) is sized
// for moderate fan-out — go's net/http default of 2 idle conns per host churns
// connections aggressively during bursts. ForceAttemptHTTP2 + 30s TCP keep-alive
// are made explicit so the contract is visible at a glance.
func buildDefaultHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
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
			// Three distinct timeout sources collapse to TimeoutError:
			//   1. caller cancelled or their ctx deadline expired (ctx.Err() != nil)
			//   2. http.Client.Timeout fired with the RoundTripper returning a
			//      context.DeadlineExceeded under the *url.Error wrapper
			//   3. http.Client.Timeout fired and net/http wrapped its internal
			//      *httpError (doesn't unwrap to DeadlineExceeded, but does
			//      implement net.Error.Timeout() returning true). Which of (2)
			//      vs (3) shows up depends on the Go version + timing — check
			//      both via net.Error.Timeout(), which both shapes implement.
			// Anything else is a transport-level failure → NetworkError.
			if ctx.Err() != nil || isTimeoutErr(err) {
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

// isTimeoutErr reports whether err represents a timeout. Walks the wrap chain
// looking for either context.DeadlineExceeded (the simple case) or anything
// implementing the net.Error.Timeout() interface (the http.Client.Timeout
// case where Go wraps an internal *httpError).
func isTimeoutErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
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
