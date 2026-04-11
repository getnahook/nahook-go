package management_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	nahook "github.com/getnahook/nahook-go"
	"github.com/getnahook/nahook-go/management"
)

func setupClient(t *testing.T) (*management.Management, string) {
	t.Helper()

	apiURL := os.Getenv("NAHOOK_TEST_API_URL")
	token := os.Getenv("NAHOOK_TEST_MGMT_TOKEN")
	workspaceID := os.Getenv("NAHOOK_TEST_WORKSPACE_ID")

	if apiURL == "" || token == "" || workspaceID == "" {
		t.Skip("NAHOOK_TEST_API_URL, NAHOOK_TEST_MGMT_TOKEN, and NAHOOK_TEST_WORKSPACE_ID must be set")
	}

	client, err := management.New(token, management.WithBaseURL(apiURL))
	if err != nil {
		t.Fatalf("failed to create management client: %v", err)
	}

	return client, workspaceID
}

func uniqueName() string {
	return fmt.Sprintf("mgmt.test.%d", time.Now().UnixMilli())
}

func strPtr(s string) *string { return &s }

func TestEventTypesCRUD(t *testing.T) {
	client, wsID := setupClient(t)
	ctx := context.Background()
	name := uniqueName()

	// Create
	created, err := client.EventTypes.Create(ctx, wsID, nahook.CreateEventTypeOptions{
		Name:        name,
		Description: "integration test event type",
	})
	if err != nil {
		t.Fatalf("Create event type: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create event type returned empty ID")
	}
	if created.Name != name {
		t.Fatalf("expected name %q, got %q", name, created.Name)
	}

	// List
	list, err := client.EventTypes.List(ctx, wsID)
	if err != nil {
		t.Fatalf("List event types: %v", err)
	}
	found := false
	for _, et := range list.Data {
		if et.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created event type %s not found in list", created.ID)
	}

	// Get
	got, err := client.EventTypes.Get(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Get event type: %v", err)
	}
	if got.Name != name {
		t.Fatalf("Get returned name %q, expected %q", got.Name, name)
	}

	// Update
	updated, err := client.EventTypes.Update(ctx, wsID, created.ID, nahook.UpdateEventTypeOptions{
		Description: strPtr("updated description"),
	})
	if err != nil {
		t.Fatalf("Update event type: %v", err)
	}
	if updated.Description == nil || *updated.Description != "updated description" {
		t.Fatalf("Update did not apply description")
	}

	// Delete
	err = client.EventTypes.Delete(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Delete event type: %v", err)
	}

	// Verify 404
	_, err = client.EventTypes.Get(ctx, wsID, created.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected 404 after delete, got: %v", err)
	}
}

func TestEndpointsCRUD(t *testing.T) {
	client, wsID := setupClient(t)
	ctx := context.Background()
	desc := uniqueName()

	// Create
	created, err := client.Endpoints.Create(ctx, wsID, nahook.CreateEndpointOptions{
		URL:         "https://httpbin.org/post",
		Description: desc,
	})
	if err != nil {
		t.Fatalf("Create endpoint: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create endpoint returned empty ID")
	}
	if created.URL != "https://httpbin.org/post" {
		t.Fatalf("expected URL https://httpbin.org/post, got %q", created.URL)
	}

	// List
	list, err := client.Endpoints.List(ctx, wsID)
	if err != nil {
		t.Fatalf("List endpoints: %v", err)
	}
	found := false
	for _, ep := range list.Data {
		if ep.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created endpoint %s not found in list", created.ID)
	}

	// Get
	got, err := client.Endpoints.Get(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Get endpoint: %v", err)
	}
	if got.URL != "https://httpbin.org/post" {
		t.Fatalf("Get returned URL %q, expected https://httpbin.org/post", got.URL)
	}

	// Update
	updated, err := client.Endpoints.Update(ctx, wsID, created.ID, nahook.UpdateEndpointOptions{
		Description: strPtr("updated endpoint"),
	})
	if err != nil {
		t.Fatalf("Update endpoint: %v", err)
	}
	if updated.Description == nil || *updated.Description != "updated endpoint" {
		t.Fatalf("Update did not apply description")
	}

	// Delete
	err = client.Endpoints.Delete(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Delete endpoint: %v", err)
	}

	// Verify 404
	_, err = client.Endpoints.Get(ctx, wsID, created.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected 404 after delete, got: %v", err)
	}
}

func TestApplicationsCRUD(t *testing.T) {
	client, wsID := setupClient(t)
	ctx := context.Background()
	name := uniqueName()

	// Create
	created, err := client.Applications.Create(ctx, wsID, nahook.CreateApplicationOptions{
		Name:     name,
		Metadata: map[string]string{"env": "test"},
	})
	if err != nil {
		t.Fatalf("Create application: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create application returned empty ID")
	}
	if created.Name != name {
		t.Fatalf("expected name %q, got %q", name, created.Name)
	}

	// List
	list, err := client.Applications.List(ctx, wsID, nil)
	if err != nil {
		t.Fatalf("List applications: %v", err)
	}
	found := false
	for _, app := range list.Data {
		if app.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created application %s not found in list", created.ID)
	}

	// Get
	got, err := client.Applications.Get(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Get application: %v", err)
	}
	if got.Name != name {
		t.Fatalf("Get returned name %q, expected %q", got.Name, name)
	}

	// Update
	updatedName := name + ".updated"
	updated, err := client.Applications.Update(ctx, wsID, created.ID, nahook.UpdateApplicationOptions{
		Name: &updatedName,
	})
	if err != nil {
		t.Fatalf("Update application: %v", err)
	}
	if updated.Name != updatedName {
		t.Fatalf("Update did not apply name: got %q, want %q", updated.Name, updatedName)
	}

	// Delete
	err = client.Applications.Delete(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Delete application: %v", err)
	}

	// Verify 404
	_, err = client.Applications.Get(ctx, wsID, created.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected 404 after delete, got: %v", err)
	}
}

func TestSubscriptions(t *testing.T) {
	client, wsID := setupClient(t)
	ctx := context.Background()
	suffix := uniqueName()

	// Create an endpoint to subscribe
	ep, err := client.Endpoints.Create(ctx, wsID, nahook.CreateEndpointOptions{
		URL:         "https://httpbin.org/post",
		Description: "sub-test-" + suffix,
	})
	if err != nil {
		t.Fatalf("Create endpoint for subscription test: %v", err)
	}
	defer func() {
		_ = client.Endpoints.Delete(ctx, wsID, ep.ID)
	}()

	// Create an event type to subscribe to
	et, err := client.EventTypes.Create(ctx, wsID, nahook.CreateEventTypeOptions{
		Name:        suffix,
		Description: "subscription test event type",
	})
	if err != nil {
		t.Fatalf("Create event type for subscription test: %v", err)
	}
	defer func() {
		_ = client.EventTypes.Delete(ctx, wsID, et.ID)
	}()

	// Subscribe (bulk API: eventTypeIds array, returns {subscribed: N})
	subResult, err := client.Subscriptions.Create(ctx, wsID, ep.ID, nahook.CreateSubscriptionOptions{
		EventTypeIDs: []string{et.ID},
	})
	if err != nil {
		t.Fatalf("Create subscription: %v", err)
	}
	if subResult.Subscribed != 1 {
		t.Fatalf("expected subscribed=1, got %d", subResult.Subscribed)
	}

	// List subscriptions
	list, err := client.Subscriptions.List(ctx, wsID, ep.ID)
	if err != nil {
		t.Fatalf("List subscriptions: %v", err)
	}
	found := false
	for _, s := range list.Data {
		if s.EventTypeID == et.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("subscription for event type %s not found in list", et.ID)
	}

	// Unsubscribe (DELETE by event type public ID, returns 204)
	err = client.Subscriptions.Delete(ctx, wsID, ep.ID, et.ID)
	if err != nil {
		t.Fatalf("Delete subscription: %v", err)
	}

	// Verify unsubscribed
	listAfter, err := client.Subscriptions.List(ctx, wsID, ep.ID)
	if err != nil {
		t.Fatalf("List subscriptions after delete: %v", err)
	}
	for _, s := range listAfter.Data {
		if s.EventTypeID == et.ID {
			t.Fatalf("subscription for event type %s still exists after delete", et.ID)
		}
	}
}

func TestAuthError(t *testing.T) {
	apiURL := os.Getenv("NAHOOK_TEST_API_URL")
	wsID := os.Getenv("NAHOOK_TEST_WORKSPACE_ID")
	if apiURL == "" || wsID == "" {
		t.Skip("NAHOOK_TEST_API_URL and NAHOOK_TEST_WORKSPACE_ID must be set")
	}

	badClient, err := management.New("nhm_invalid_token_000", management.WithBaseURL(apiURL))
	if err != nil {
		t.Fatalf("failed to create client with bad token: %v", err)
	}

	_, err = badClient.EventTypes.List(context.Background(), wsID)
	if err == nil {
		t.Fatal("expected auth error with bad token, got nil")
	}

	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 401 {
		t.Fatalf("expected status 401, got %d", apiErr.Status)
	}
}
