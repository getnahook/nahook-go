package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	nahook "github.com/getnahook/nahook-go"
)

func TestNew_InvalidAPIKey(t *testing.T) {
	_, err := New("bad_key")
	if err == nil {
		t.Fatal("expected error for invalid API key")
	}
	if err.Error() != "nahook: invalid API key: must start with 'nhk_'" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestNew_ValidAPIKey(t *testing.T) {
	c, err := New("nhk_test123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/ingest/ep_123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer nhk_test123" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("unexpected accept header: %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "nahook-go/0.1.0" {
			t.Errorf("unexpected user-agent: %s", r.Header.Get("User-Agent"))
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["idempotencyKey"] == nil || body["idempotencyKey"] == "" {
			t.Error("expected idempotencyKey to be auto-generated")
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deliveryId":     "del_abc",
			"idempotencyKey": body["idempotencyKey"],
			"status":         "accepted",
		})
	}))
	defer srv.Close()

	c, _ := New("nhk_test123", WithBaseURL(srv.URL))
	result, err := c.Send(context.Background(), "ep_123", nahook.SendOptions{
		Payload: map[string]interface{}{"test": true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DeliveryID != "del_abc" {
		t.Errorf("expected deliveryId del_abc, got %s", result.DeliveryID)
	}
	if result.Status != "accepted" {
		t.Errorf("expected status accepted, got %s", result.Status)
	}
	if result.IdempotencyKey == "" {
		t.Error("expected idempotencyKey to be set")
	}
}

func TestSend_CustomIdempotencyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["idempotencyKey"] != "my-key-123" {
			t.Errorf("expected custom idempotency key, got %v", body["idempotencyKey"])
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deliveryId":     "del_abc",
			"idempotencyKey": "my-key-123",
			"status":         "accepted",
		})
	}))
	defer srv.Close()

	c, _ := New("nhk_test123", WithBaseURL(srv.URL))
	result, err := c.Send(context.Background(), "ep_123", nahook.SendOptions{
		Payload:        map[string]interface{}{"test": true},
		IdempotencyKey: "my-key-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IdempotencyKey != "my-key-123" {
		t.Errorf("expected idempotencyKey my-key-123, got %s", result.IdempotencyKey)
	}
}

func TestTrigger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/ingest/event/order.paid" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["payload"] == nil {
			t.Error("expected payload in body")
		}
		if _, hasKey := body["idempotencyKey"]; hasKey {
			t.Error("trigger should NOT include idempotencyKey in body")
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"eventTypeId": "evt_abc",
			"deliveryIds": []string{"del_1"},
			"status":      "accepted",
		})
	}))
	defer srv.Close()

	c, _ := New("nhk_test123", WithBaseURL(srv.URL))
	result, err := c.Trigger(context.Background(), "order.paid", nahook.TriggerOptions{
		Payload: map[string]interface{}{"orderId": "123"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventTypeID != "evt_abc" {
		t.Errorf("expected eventTypeId evt_abc, got %s", result.EventTypeID)
	}
	if len(result.DeliveryIDs) != 1 || result.DeliveryIDs[0] != "del_1" {
		t.Errorf("unexpected deliveryIds: %v", result.DeliveryIDs)
	}
}

func TestTrigger_WithMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		meta, ok := body["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("expected metadata in body")
		}
		if meta["region"] != "us-east-1" {
			t.Errorf("unexpected metadata: %v", meta)
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"eventTypeId": "evt_abc",
			"deliveryIds": []string{},
			"status":      "accepted",
		})
	}))
	defer srv.Close()

	c, _ := New("nhk_test123", WithBaseURL(srv.URL))
	_, err := c.Trigger(context.Background(), "order.paid", nahook.TriggerOptions{
		Payload:  map[string]interface{}{"orderId": "123"},
		Metadata: map[string]string{"region": "us-east-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/ingest/batch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		items, ok := body["items"].([]interface{})
		if !ok {
			t.Fatal("expected body to have 'items' array wrapper")
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item in batch, got %d", len(items))
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"index": 0, "deliveryId": "del_abc", "status": "accepted"},
			},
		})
	}))
	defer srv.Close()

	c, _ := New("nhk_test123", WithBaseURL(srv.URL))
	result, err := c.SendBatch(context.Background(), []nahook.SendBatchItem{
		{EndpointID: "ep_123", Payload: map[string]interface{}{"test": true}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Status != "accepted" {
		t.Errorf("expected status accepted, got %s", result.Items[0].Status)
	}
}

func TestTriggerBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/ingest/event/batch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		items, ok := body["items"].([]interface{})
		if !ok {
			t.Fatal("expected body to have 'items' array wrapper")
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item in batch, got %d", len(items))
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"index": 0, "eventTypeId": "evt_abc", "deliveryIds": []string{}, "status": "accepted"},
			},
		})
	}))
	defer srv.Close()

	c, _ := New("nhk_test123", WithBaseURL(srv.URL))
	result, err := c.TriggerBatch(context.Background(), []nahook.TriggerBatchItem{
		{EventType: "order.paid", Payload: map[string]interface{}{"orderId": "123"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestSend_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "not_found",
				"message": "Endpoint not found",
			},
		})
	}))
	defer srv.Close()

	c, _ := New("nhk_test123", WithBaseURL(srv.URL))
	_, err := c.Send(context.Background(), "ep_missing", nahook.SendOptions{
		Payload: map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*nahook.APIError)
	if !ok {
		t.Fatalf("expected *nahook.APIError, got %T", err)
	}
	if apiErr.Status != 404 {
		t.Errorf("expected status 404, got %d", apiErr.Status)
	}
	if apiErr.Code != "not_found" {
		t.Errorf("expected code not_found, got %s", apiErr.Code)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

func TestSend_NoContentTypeOnGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "" {
			t.Error("GET request should not have Content-Type header")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer srv.Close()

	// Use the HTTP client directly to test GET without Content-Type
	httpClient := nahook.NewHTTPClient(nahook.HTTPClientConfig{
		Token:   "nhk_test123",
		BaseURL: srv.URL,
	})
	var result []interface{}
	httpClient.Request(context.Background(), nahook.RequestOptions{
		Method: "GET",
		Path:   "/test",
	}, &result)
}
