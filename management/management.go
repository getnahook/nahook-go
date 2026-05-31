// Package management provides the Nahook management API client for
// administering workspaces, endpoints, event types, applications,
// subscriptions, and portal sessions.
package management

import (
	"fmt"
	"net/http"
	"time"

	nahook "github.com/getnahook/nahook-go"
)

// Management is the Nahook management API client.
type Management struct {
	Endpoints      *EndpointsResource
	EventTypes     *EventTypesResource
	Applications   *ApplicationsResource
	Subscriptions  *SubscriptionsResource
	PortalSessions *PortalSessionsResource
	Environments   *EnvironmentsResource
	Deliveries     *DeliveriesResource

	http *nahook.HTTPClient
}

// Option configures the Management client.
type Option func(*options)

type options struct {
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(o *options) { o.baseURL = url }
}

// WithTimeout sets the HTTP request timeout. Ignored when WithHTTPClient is
// also supplied — the caller-owned *http.Client's Timeout governs in that case.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithHTTPClient supplies a caller-owned *http.Client to use for all requests.
// The SDK uses it verbatim and does not mutate it. The caller's HTTPClient.Timeout
// governs request timeouts and is what TimeoutError.TimeoutMs reports.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) { o.httpClient = c }
}

// New creates a new Nahook management API client.
// The token must start with "nhm_".
func New(token string, opts ...Option) (*Management, error) {
	if len(token) < 4 || token[:4] != "nhm_" {
		return nil, fmt.Errorf("nahook: invalid management token: must start with 'nhm_'")
	}

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	httpClient := nahook.NewHTTPClient(nahook.HTTPClientConfig{
		Token:      token,
		BaseURL:    o.baseURL,
		Timeout:    o.timeout,
		Retries:    0, // management client never retries
		HTTPClient: o.httpClient,
	})

	return &Management{
		Endpoints:      &EndpointsResource{http: httpClient},
		EventTypes:     &EventTypesResource{http: httpClient},
		Applications:   &ApplicationsResource{http: httpClient},
		Subscriptions:  &SubscriptionsResource{http: httpClient},
		PortalSessions: &PortalSessionsResource{http: httpClient},
		Environments:   &EnvironmentsResource{http: httpClient},
		Deliveries:     &DeliveriesResource{http: httpClient},

		http: httpClient,
	}, nil
}

// HTTPClient returns the underlying *http.Client used by the SDK. Useful for
// introspection or attaching instrumentation. Mutating the returned client
// affects all subsequent SDK requests.
func (m *Management) HTTPClient() *http.Client {
	return m.http.HTTPClient()
}
