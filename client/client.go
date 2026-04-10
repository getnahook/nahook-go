// Package client provides the Nahook ingestion client for sending webhooks.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	nahook "github.com/jmatom/nahook-go"
)

// Client is the Nahook ingestion client.
type Client struct {
	http *nahook.HTTPClient
}

// Option configures the Client.
type Option func(*options)

type options struct {
	baseURL string
	timeout time.Duration
	retries int
}

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(o *options) { o.baseURL = url }
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithRetries sets the maximum number of retries for retryable errors.
func WithRetries(n int) Option {
	return func(o *options) { o.retries = n }
}

// New creates a new Nahook ingestion client.
// The apiKey must start with "nhk_".
func New(apiKey string, opts ...Option) (*Client, error) {
	if len(apiKey) < 4 || apiKey[:4] != "nhk_" {
		return nil, fmt.Errorf("nahook: invalid API key: must start with 'nhk_'")
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return &Client{
		http: nahook.NewHTTPClient(nahook.HTTPClientConfig{
			Token:   apiKey,
			BaseURL: o.baseURL,
			Timeout: o.timeout,
			Retries: o.retries,
		}),
	}, nil
}

// Send sends a payload to a specific endpoint.
// If IdempotencyKey is not set in opts, a UUID v4 is generated automatically.
func (c *Client) Send(ctx context.Context, endpointID string, opts nahook.SendOptions) (*nahook.SendResult, error) {
	if opts.IdempotencyKey == "" {
		opts.IdempotencyKey = uuid.New().String()
	}

	body := map[string]interface{}{
		"payload":        opts.Payload,
		"idempotencyKey": opts.IdempotencyKey,
	}

	var result nahook.SendResult
	err := c.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/api/ingest/" + nahook.PathEncode(endpointID),
		Body:   body,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Trigger fans out a payload by event type to all subscribed endpoints.
func (c *Client) Trigger(ctx context.Context, eventType string, opts nahook.TriggerOptions) (*nahook.TriggerResult, error) {
	body := map[string]interface{}{
		"payload": opts.Payload,
	}
	if len(opts.Metadata) > 0 {
		body["metadata"] = opts.Metadata
	}

	var result nahook.TriggerResult
	err := c.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/api/ingest/event/" + nahook.PathEncode(eventType),
		Body:   body,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SendBatch sends payloads to multiple specific endpoints in a single request (max 20 items).
func (c *Client) SendBatch(ctx context.Context, items []nahook.SendBatchItem) (*nahook.BatchResult, error) {
	body := map[string]interface{}{
		"items": items,
	}

	var result nahook.BatchResult
	_, err := c.http.RequestWithStatus(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/api/ingest/batch",
		Body:   body,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// TriggerBatch fans out payloads by event types in a single request (max 20 items).
func (c *Client) TriggerBatch(ctx context.Context, items []nahook.TriggerBatchItem) (*nahook.BatchResult, error) {
	body := map[string]interface{}{
		"items": items,
	}

	var result nahook.BatchResult
	_, err := c.http.RequestWithStatus(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/api/ingest/event/batch",
		Body:   body,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
