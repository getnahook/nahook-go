package management

import (
	"context"

	nahook "github.com/jmatom/nahook-go"
)

// EventTypesResource provides operations on event types.
type EventTypesResource struct {
	http *nahook.HTTPClient
}

// List returns all event types in a workspace.
func (r *EventTypesResource) List(ctx context.Context, workspaceID string) (*nahook.ListResult[nahook.EventType], error) {
	var data []nahook.EventType
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/event-types",
	}, &data)
	if err != nil {
		return nil, err
	}
	return &nahook.ListResult[nahook.EventType]{Data: data}, nil
}

// Create creates a new event type in a workspace.
func (r *EventTypesResource) Create(ctx context.Context, workspaceID string, opts nahook.CreateEventTypeOptions) (*nahook.EventType, error) {
	var result nahook.EventType
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/event-types",
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves an event type by ID.
func (r *EventTypesResource) Get(ctx context.Context, workspaceID, id string) (*nahook.EventType, error) {
	var result nahook.EventType
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/event-types/" + nahook.PathEncode(id),
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates an event type by ID.
func (r *EventTypesResource) Update(ctx context.Context, workspaceID, id string, opts nahook.UpdateEventTypeOptions) (*nahook.EventType, error) {
	var result nahook.EventType
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "PATCH",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/event-types/" + nahook.PathEncode(id),
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes an event type by ID.
func (r *EventTypesResource) Delete(ctx context.Context, workspaceID, id string) error {
	return r.http.Request(ctx, nahook.RequestOptions{
		Method: "DELETE",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/event-types/" + nahook.PathEncode(id),
	}, nil)
}
