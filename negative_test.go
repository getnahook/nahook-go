package nahook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── Negative / resilience tests ─────────────────────────────────────────────
//
// These tests verify the HTTP client handles malformed, unexpected, and
// degraded server responses gracefully — no panics, correct error types.

// NEG-01: Malformed JSON response returns an error, not a panic.
func TestNegative_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]interface{}
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// NEG-02: Empty body on 200 returns an error when a result is expected.
func TestNegative_EmptyBodyOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write nothing — empty body
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]interface{}
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err == nil {
		t.Fatal("expected error for empty body on 200 when result is expected, got nil")
	}
}

// NEG-03: 5xx with HTML body returns APIError with correct status.
func TestNegative_5xxWithHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`<html><body><h1>502 Bad Gateway</h1></body></html>`))
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]interface{}
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err == nil {
		t.Fatal("expected error for 502 response")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 502 {
		t.Errorf("expected status 502, got %d", apiErr.Status)
	}
}

// NEG-04: Empty body 500 returns APIError with status 500.
func TestNegative_EmptyBody500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		// No body at all
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]interface{}
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 500 {
		t.Errorf("expected status 500, got %d", apiErr.Status)
	}
}

// NEG-05: Unknown/extra fields in JSON response are ignored — no error.
func TestNegative_UnknownFieldsSucceed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"deliveryId":     "del_abc",
			"idempotencyKey": "key_123",
			"status":         "accepted",
			"unknownField":   "should be ignored",
			"extraNested":    map[string]string{"a": "b"},
		})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result SendResult
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v — unknown fields should be ignored", err)
	}
	if result.DeliveryID != "del_abc" {
		t.Errorf("expected deliveryId del_abc, got %s", result.DeliveryID)
	}
}

// NEG-06: Missing expected fields in JSON response succeed with zero values.
func TestNegative_MissingFieldsSucceed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Only return partial fields — deliveryId is missing
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "accepted",
		})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result SendResult
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v — missing fields should use zero values", err)
	}
	if result.DeliveryID != "" {
		t.Errorf("expected empty deliveryId for missing field, got %s", result.DeliveryID)
	}
	if result.Status != "accepted" {
		t.Errorf("expected status accepted, got %s", result.Status)
	}
}
