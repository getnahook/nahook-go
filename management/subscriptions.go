package management

import (
	"context"

	nahook "github.com/getnahook/nahook-go"
)

// SubscriptionsResource provides operations on event type subscriptions.
type SubscriptionsResource struct {
	http *nahook.HTTPClient
}

// List returns all subscriptions for an endpoint.
func (r *SubscriptionsResource) List(ctx context.Context, workspaceID, endpointID string) (*nahook.ListResult[nahook.Subscription], error) {
	var data []nahook.Subscription
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints/" + nahook.PathEncode(endpointID) + "/subscriptions",
	}, &data)
	if err != nil {
		return nil, err
	}
	return &nahook.ListResult[nahook.Subscription]{Data: data}, nil
}

// Create subscribes an endpoint to one or more event types.
// Returns the number of event types successfully subscribed.
func (r *SubscriptionsResource) Create(ctx context.Context, workspaceID, endpointID string, opts nahook.CreateSubscriptionOptions) (*nahook.SubscribeResult, error) {
	var result nahook.SubscribeResult
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints/" + nahook.PathEncode(endpointID) + "/subscriptions",
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete removes a subscription by event type ID.
func (r *SubscriptionsResource) Delete(ctx context.Context, workspaceID, endpointID, eventTypeID string) error {
	return r.http.Request(ctx, nahook.RequestOptions{
		Method: "DELETE",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints/" + nahook.PathEncode(endpointID) + "/subscriptions/" + nahook.PathEncode(eventTypeID),
	}, nil)
}
