package management

import (
	"context"
	"strconv"

	nahook "github.com/getnahook/nahook-go"
)

// ApplicationsResource provides operations on applications.
type ApplicationsResource struct {
	http *nahook.HTTPClient
}

// List returns applications in a workspace with optional pagination.
func (r *ApplicationsResource) List(ctx context.Context, workspaceID string, opts *nahook.ListOptions) (*nahook.ListResult[nahook.Application], error) {
	query := make(map[string]string)
	if opts != nil {
		if opts.Limit != nil {
			query["limit"] = strconv.Itoa(*opts.Limit)
		}
		if opts.Offset != nil {
			query["offset"] = strconv.Itoa(*opts.Offset)
		}
	}

	var data []nahook.Application
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications",
		Query:  query,
	}, &data)
	if err != nil {
		return nil, err
	}
	return &nahook.ListResult[nahook.Application]{Data: data}, nil
}

// Create creates a new application in a workspace.
func (r *ApplicationsResource) Create(ctx context.Context, workspaceID string, opts nahook.CreateApplicationOptions) (*nahook.Application, error) {
	var result nahook.Application
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications",
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves an application by ID.
func (r *ApplicationsResource) Get(ctx context.Context, workspaceID, id string) (*nahook.Application, error) {
	var result nahook.Application
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications/" + nahook.PathEncode(id),
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates an application by ID.
func (r *ApplicationsResource) Update(ctx context.Context, workspaceID, id string, opts nahook.UpdateApplicationOptions) (*nahook.Application, error) {
	var result nahook.Application
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "PATCH",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications/" + nahook.PathEncode(id),
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes an application by ID.
func (r *ApplicationsResource) Delete(ctx context.Context, workspaceID, id string) error {
	return r.http.Request(ctx, nahook.RequestOptions{
		Method: "DELETE",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications/" + nahook.PathEncode(id),
	}, nil)
}

// ListEndpoints returns all endpoints belonging to an application.
func (r *ApplicationsResource) ListEndpoints(ctx context.Context, workspaceID, appID string) (*nahook.ListResult[nahook.Endpoint], error) {
	var data []nahook.Endpoint
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications/" + nahook.PathEncode(appID) + "/endpoints",
	}, &data)
	if err != nil {
		return nil, err
	}
	return &nahook.ListResult[nahook.Endpoint]{Data: data}, nil
}

// CreateEndpoint creates a new endpoint under an application.
func (r *ApplicationsResource) CreateEndpoint(ctx context.Context, workspaceID, appID string, opts nahook.CreateEndpointOptions) (*nahook.Endpoint, error) {
	var result nahook.Endpoint
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications/" + nahook.PathEncode(appID) + "/endpoints",
		Body:   opts,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
