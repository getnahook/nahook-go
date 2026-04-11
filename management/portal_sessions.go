package management

import (
	"context"

	nahook "github.com/getnahook/nahook-go"
)

// PortalSessionsResource provides operations on developer portal sessions.
type PortalSessionsResource struct {
	http *nahook.HTTPClient
}

// Create creates a new portal session for an application.
func (r *PortalSessionsResource) Create(ctx context.Context, workspaceID, appID string, opts *nahook.CreatePortalSessionOptions) (*nahook.PortalSession, error) {
	var body interface{}
	if opts != nil {
		body = opts
	} else {
		body = map[string]interface{}{}
	}

	var result nahook.PortalSession
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "POST",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/applications/" + nahook.PathEncode(appID) + "/portal",
		Body:   body,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
