package management

import (
	"context"

	nahook "github.com/getnahook/nahook-go"
)

// EnvironmentsResource provides operations on workspace environments.
type EnvironmentsResource struct {
	http *nahook.HTTPClient
}

func environmentsPath(workspaceID string) string {
	return "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/environments"
}

func environmentPath(workspaceID, id string) string {
	return environmentsPath(workspaceID) + "/" + nahook.PathEncode(id)
}

// List returns all environments in a workspace.
func (r *EnvironmentsResource) List(ctx context.Context, workspaceID string) (*nahook.ListResult[nahook.Environment], error) {
	var data []nahook.Environment
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   environmentsPath(workspaceID),
	}, &data)
	if err != nil {
		return nil, err
	}
	return &nahook.ListResult[nahook.Environment]{Data: data}, nil
}

// Create creates a new environment in a workspace.
func (r *EnvironmentsResource) Create(ctx context.Context, workspaceID string, opts nahook.CreateEnvironmentOptions) (*nahook.Environment, error) {
	var result nahook.Environment
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   environmentsPath(workspaceID),
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves an environment by ID.
func (r *EnvironmentsResource) Get(ctx context.Context, workspaceID, id string) (*nahook.Environment, error) {
	var result nahook.Environment
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   environmentPath(workspaceID, id),
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates an environment by ID.
func (r *EnvironmentsResource) Update(ctx context.Context, workspaceID, id string, opts nahook.UpdateEnvironmentOptions) (*nahook.Environment, error) {
	var result nahook.Environment
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "PATCH",
		Path:   environmentPath(workspaceID, id),
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes an environment by ID.
func (r *EnvironmentsResource) Delete(ctx context.Context, workspaceID, id string) error {
	return r.http.Request(ctx, nahook.RequestOptions{
		Method: "DELETE",
		Path:   environmentPath(workspaceID, id),
	}, nil)
}

// ListEventTypeVisibility returns the visibility of all event types in an environment.
func (r *EnvironmentsResource) ListEventTypeVisibility(ctx context.Context, workspaceID, envID string) (*nahook.ListResult[nahook.EventTypeVisibility], error) {
	var data []nahook.EventTypeVisibility
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   environmentPath(workspaceID, envID) + "/event-types",
	}, &data)
	if err != nil {
		return nil, err
	}
	return &nahook.ListResult[nahook.EventTypeVisibility]{Data: data}, nil
}

// SetEventTypeVisibility sets the visibility of an event type in an environment.
func (r *EnvironmentsResource) SetEventTypeVisibility(ctx context.Context, workspaceID, envID, eventTypeID string, opts nahook.SetVisibilityOptions) (*nahook.EventTypeVisibility, error) {
	var result nahook.EventTypeVisibility
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "PUT",
		Path:   environmentPath(workspaceID, envID) + "/event-types/" + nahook.PathEncode(eventTypeID) + "/visibility",
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
