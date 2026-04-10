package management

import (
	"context"

	nahook "github.com/jmatom/nahook-go"
)

// EndpointsResource provides operations on webhook endpoints.
type EndpointsResource struct {
	http *nahook.HTTPClient
}

// List returns all endpoints in a workspace.
func (r *EndpointsResource) List(ctx context.Context, workspaceID string) (*nahook.ListResult[nahook.Endpoint], error) {
	var data []nahook.Endpoint
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints",
	}, &data)
	if err != nil {
		return nil, err
	}
	return &nahook.ListResult[nahook.Endpoint]{Data: data}, nil
}

// Create creates a new endpoint in a workspace.
func (r *EndpointsResource) Create(ctx context.Context, workspaceID string, opts nahook.CreateEndpointOptions) (*nahook.Endpoint, error) {
	var result nahook.Endpoint
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints",
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves an endpoint by ID.
func (r *EndpointsResource) Get(ctx context.Context, workspaceID, id string) (*nahook.Endpoint, error) {
	var result nahook.Endpoint
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints/" + nahook.PathEncode(id),
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates an endpoint by ID.
func (r *EndpointsResource) Update(ctx context.Context, workspaceID, id string, opts nahook.UpdateEndpointOptions) (*nahook.Endpoint, error) {
	var result nahook.Endpoint
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "PATCH",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints/" + nahook.PathEncode(id),
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes an endpoint by ID.
func (r *EndpointsResource) Delete(ctx context.Context, workspaceID, id string) error {
	return r.http.Request(ctx, nahook.RequestOptions{
		Method: "DELETE",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints/" + nahook.PathEncode(id),
	}, nil)
}
