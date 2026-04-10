// Package management provides the Nahook management API client for
// administering workspaces, endpoints, event types, applications,
// subscriptions, and portal sessions.
package management

import (
	"fmt"
	"time"

	nahook "github.com/jmatom/nahook-go"
)

// Management is the Nahook management API client.
type Management struct {
	Endpoints      *EndpointsResource
	EventTypes     *EventTypesResource
	Applications   *ApplicationsResource
	Subscriptions  *SubscriptionsResource
	PortalSessions *PortalSessionsResource
}

// Option configures the Management client.
type Option func(*options)

type options struct {
	baseURL string
	timeout time.Duration
}

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(o *options) { o.baseURL = url }
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
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

	http := nahook.NewHTTPClient(nahook.HTTPClientConfig{
		Token:   token,
		BaseURL: o.baseURL,
		Timeout: o.timeout,
		Retries: 0, // management client never retries
	})

	return &Management{
		Endpoints:      &EndpointsResource{http: http},
		EventTypes:     &EventTypesResource{http: http},
		Applications:   &ApplicationsResource{http: http},
		Subscriptions:  &SubscriptionsResource{http: http},
		PortalSessions: &PortalSessionsResource{http: http},
	}, nil
}
