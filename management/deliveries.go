package management

import (
	"context"
	"strconv"

	nahook "github.com/getnahook/nahook-go"
)

// DeliveriesResource provides read-only access to a workspace's webhook
// deliveries. There is no create/update/delete — deliveries are produced
// by the ingestion API and read here for inspection and debugging.
//
// List is scoped to a single endpoint because the regional deliveries table
// is indexed by webhook endpoint; a workspace-wide list is not supported.
// Get and GetAttempts accept a delivery publicId directly.
type DeliveriesResource struct {
	http *nahook.HTTPClient
}

// listDeliveriesResponse is the raw wire shape returned by the List endpoint.
// The SDK renames `deliveries` to `Data` in the public PaginatedResult to
// keep the generic type uniform across resources.
type listDeliveriesResponse struct {
	Deliveries []nahook.Delivery `json:"deliveries"`
	NextCursor *string           `json:"nextCursor"`
}

// List returns a cursor-paginated page of deliveries for an endpoint, newest
// first. The returned NextCursor is an opaque token — pass it verbatim on
// the next call's ListDeliveriesOptions.Cursor to fetch the next page. When
// NextCursor is nil there are no more pages.
func (r *DeliveriesResource) List(ctx context.Context, workspaceID, endpointID string, opts *nahook.ListDeliveriesOptions) (*nahook.PaginatedResult[nahook.Delivery], error) {
	query := make(map[string]string)
	if opts != nil {
		if opts.Limit != nil {
			query["limit"] = strconv.Itoa(*opts.Limit)
		}
		if opts.Cursor != "" {
			query["cursor"] = opts.Cursor
		}
		if opts.Status != "" {
			query["status"] = opts.Status
		}
	}

	var raw listDeliveriesResponse
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/endpoints/" + nahook.PathEncode(endpointID) + "/deliveries",
		Query:  query,
	}, &raw)
	if err != nil {
		return nil, err
	}
	return &nahook.PaginatedResult[nahook.Delivery]{
		Data:       raw.Deliveries,
		NextCursor: raw.NextCursor,
	}, nil
}

// Get retrieves a single delivery's metadata. When opts.IncludePayload is
// true the response also carries a PayloadEnvelope — inspect its Status
// before reading Data, as only "available" envelopes carry payload bytes.
// The other four envelope statuses ("forbidden", "processing", "not_found",
// "error") are not errors: they are returned with HTTP 200.
func (r *DeliveriesResource) Get(ctx context.Context, workspaceID, deliveryID string, opts *nahook.GetDeliveryOptions) (*nahook.DeliveryWithPayload, error) {
	query := make(map[string]string)
	if opts != nil && opts.IncludePayload {
		query["include"] = "payload"
	}

	var result nahook.DeliveryWithPayload
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/deliveries/" + nahook.PathEncode(deliveryID),
		Query:  query,
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAttempts returns the HTTP delivery attempts for a delivery, in
// chronological order (oldest first). Returns an empty slice when no
// attempts have been made yet.
func (r *DeliveriesResource) GetAttempts(ctx context.Context, workspaceID, deliveryID string) ([]nahook.DeliveryAttempt, error) {
	var result []nahook.DeliveryAttempt
	err := r.http.Request(ctx, nahook.RequestOptions{
		Method: "GET",
		Path:   "/management/v1/workspaces/" + nahook.PathEncode(workspaceID) + "/deliveries/" + nahook.PathEncode(deliveryID) + "/attempts",
	}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
