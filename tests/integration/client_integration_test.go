package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	nahook "github.com/getnahook/nahook-go"
	"github.com/getnahook/nahook-go/client"
)

// env helpers

func requiredEnv(t *testing.T) (apiURL, apiKey, disabledKey, endpointID, eventType string) {
	t.Helper()
	apiURL = os.Getenv("NAHOOK_TEST_API_URL")
	apiKey = os.Getenv("NAHOOK_TEST_API_KEY")
	disabledKey = os.Getenv("NAHOOK_TEST_DISABLED_API_KEY")
	endpointID = os.Getenv("NAHOOK_TEST_ENDPOINT_ID")
	eventType = os.Getenv("NAHOOK_TEST_EVENT_TYPE")

	if apiURL == "" || apiKey == "" || disabledKey == "" || endpointID == "" || eventType == "" {
		t.Skip("integration test env not set")
	}
	return
}

func newClient(t *testing.T, apiURL, apiKey string) *client.Client {
	t.Helper()
	c, err := client.New(apiKey, client.WithBaseURL(apiURL))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

func testPayload() map[string]interface{} {
	return map[string]interface{}{
		"test":      true,
		"timestamp": time.Now().UnixMilli(),
	}
}

func uniqueKey() string {
	return fmt.Sprintf("idem_%d", time.Now().UnixNano())
}

// ── Send tests ──────────────────────────────────────────────────────────────

func TestSend_HappyPath(t *testing.T) {
	apiURL, apiKey, _, endpointID, _ := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	result, err := c.Send(ctx, endpointID, nahook.SendOptions{
		Payload: testPayload(),
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if result.Status != "accepted" {
		t.Errorf("expected status 'accepted', got %q", result.Status)
	}
	if !strings.HasPrefix(result.DeliveryID, "del_") {
		t.Errorf("expected DeliveryID to start with 'del_', got %q", result.DeliveryID)
	}
	if result.IdempotencyKey == "" {
		t.Error("expected IdempotencyKey to be non-empty")
	}
}

func TestSend_IdempotencyDedup(t *testing.T) {
	apiURL, apiKey, _, endpointID, _ := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	key := uniqueKey()
	payload := testPayload()

	r1, err := c.Send(ctx, endpointID, nahook.SendOptions{
		Payload:        payload,
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("first Send failed: %v", err)
	}

	r2, err := c.Send(ctx, endpointID, nahook.SendOptions{
		Payload:        payload,
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("second Send failed: %v", err)
	}

	if r1.DeliveryID != r2.DeliveryID {
		t.Errorf("expected same DeliveryID for duplicate key, got %q and %q", r1.DeliveryID, r2.DeliveryID)
	}
}

func TestSend_SeparateKeys(t *testing.T) {
	apiURL, apiKey, _, endpointID, _ := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	r1, err := c.Send(ctx, endpointID, nahook.SendOptions{
		Payload:        testPayload(),
		IdempotencyKey: uniqueKey(),
	})
	if err != nil {
		t.Fatalf("first Send failed: %v", err)
	}

	r2, err := c.Send(ctx, endpointID, nahook.SendOptions{
		Payload:        testPayload(),
		IdempotencyKey: uniqueKey(),
	})
	if err != nil {
		t.Fatalf("second Send failed: %v", err)
	}

	if r1.DeliveryID == r2.DeliveryID {
		t.Errorf("expected different DeliveryIDs for separate keys, both got %q", r1.DeliveryID)
	}
}

// ── Trigger tests ───────────────────────────────────────────────────────────

func TestTrigger_FanOut(t *testing.T) {
	apiURL, apiKey, _, _, eventType := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	result, err := c.Trigger(ctx, eventType, nahook.TriggerOptions{
		Payload: testPayload(),
	})
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	if result.Status != "accepted" {
		t.Errorf("expected status 'accepted', got %q", result.Status)
	}
	if !strings.HasPrefix(result.EventTypeID, "evt_") {
		t.Errorf("expected EventTypeID to start with 'evt_', got %q", result.EventTypeID)
	}
	if len(result.DeliveryIDs) < 1 {
		t.Error("expected at least 1 DeliveryID in fan-out")
	}
}

func TestTrigger_Unsubscribed(t *testing.T) {
	apiURL, apiKey, _, _, _ := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	result, err := c.Trigger(ctx, "unsubscribed.event.that.does.not.exist", nahook.TriggerOptions{
		Payload: testPayload(),
	})
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	if len(result.DeliveryIDs) != 0 {
		t.Errorf("expected 0 DeliveryIDs for unsubscribed event, got %d", len(result.DeliveryIDs))
	}
}

// ── Batch tests ─────────────────────────────────────────────────────────────

func TestSendBatch(t *testing.T) {
	apiURL, apiKey, _, endpointID, _ := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	items := []nahook.SendBatchItem{
		{EndpointID: endpointID, Payload: testPayload()},
		{EndpointID: endpointID, Payload: testPayload()},
	}

	result, err := c.SendBatch(ctx, items)
	if err != nil {
		t.Fatalf("SendBatch failed: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 batch items, got %d", len(result.Items))
	}
	for i, item := range result.Items {
		if item.Status != "accepted" {
			t.Errorf("item[%d]: expected status 'accepted', got %q", i, item.Status)
		}
		if !strings.HasPrefix(item.DeliveryID, "del_") {
			t.Errorf("item[%d]: expected DeliveryID to start with 'del_', got %q", i, item.DeliveryID)
		}
	}
}

func TestTriggerBatch(t *testing.T) {
	apiURL, apiKey, _, _, eventType := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	items := []nahook.TriggerBatchItem{
		{EventType: eventType, Payload: testPayload()},
		{EventType: eventType, Payload: testPayload()},
	}

	result, err := c.TriggerBatch(ctx, items)
	if err != nil {
		t.Fatalf("TriggerBatch failed: %v", err)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 batch items, got %d", len(result.Items))
	}
	for i, item := range result.Items {
		if item.Status != "accepted" {
			t.Errorf("item[%d]: expected status 'accepted', got %q", i, item.Status)
		}
		if !strings.HasPrefix(item.EventTypeID, "evt_") {
			t.Errorf("item[%d]: expected EventTypeID to start with 'evt_', got %q", i, item.EventTypeID)
		}
	}
}

// ── Error tests ─────────────────────────────────────────────────────────────

func TestError_401_InvalidKey(t *testing.T) {
	apiURL, _, _, endpointID, _ := requiredEnv(t)
	c := newClient(t, apiURL, "nhk_us_garbage_key_that_does_not_exist")
	ctx := context.Background()

	_, err := c.Send(ctx, endpointID, nahook.SendOptions{
		Payload: testPayload(),
	})
	if err == nil {
		t.Fatal("expected error for invalid API key, got nil")
	}

	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *nahook.APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 401 {
		t.Errorf("expected status 401, got %d", apiErr.Status)
	}
	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError() to be true")
	}
	if apiErr.IsRetryable() {
		t.Error("expected IsRetryable() to be false")
	}
}

func TestError_403_DisabledKey(t *testing.T) {
	apiURL, _, disabledKey, endpointID, _ := requiredEnv(t)
	c := newClient(t, apiURL, disabledKey)
	ctx := context.Background()

	_, err := c.Send(ctx, endpointID, nahook.SendOptions{
		Payload: testPayload(),
	})
	if err == nil {
		t.Fatal("expected error for disabled API key, got nil")
	}

	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *nahook.APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 403 {
		t.Errorf("expected status 403, got %d", apiErr.Status)
	}
	if apiErr.Code != "token_disabled" {
		t.Errorf("expected code 'token_disabled', got %q", apiErr.Code)
	}
	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError() to be true")
	}
}

func TestError_404_MissingEndpoint(t *testing.T) {
	apiURL, apiKey, _, _, _ := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	_, err := c.Send(ctx, "ep_nonexistent_endpoint_id_000", nahook.SendOptions{
		Payload: testPayload(),
	})
	if err == nil {
		t.Fatal("expected error for missing endpoint, got nil")
	}

	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *nahook.APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 404 {
		t.Errorf("expected status 404, got %d", apiErr.Status)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

func TestError_400_InvalidEventType(t *testing.T) {
	apiURL, apiKey, _, _, _ := requiredEnv(t)
	c := newClient(t, apiURL, apiKey)
	ctx := context.Background()

	_, err := c.Trigger(ctx, "INVALID NAME!!", nahook.TriggerOptions{
		Payload: testPayload(),
	})
	if err == nil {
		t.Fatal("expected error for invalid event type, got nil")
	}

	var apiErr *nahook.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *nahook.APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 400 {
		t.Errorf("expected status 400, got %d", apiErr.Status)
	}
	if !apiErr.IsValidationError() {
		t.Error("expected IsValidationError() to be true")
	}
}
