package nahook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{Status: 404, Code: "not_found", Message: "Endpoint not found"}
	expected := "nahook: API error 404 (not_found): Endpoint not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	tests := []struct {
		status   int
		expected bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{409, false},
		{413, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
	}
	for _, tt := range tests {
		err := &APIError{Status: tt.status}
		if err.IsRetryable() != tt.expected {
			t.Errorf("status %d: expected IsRetryable() = %v", tt.status, tt.expected)
		}
	}
}

func TestAPIError_IsAuthError(t *testing.T) {
	tests := []struct {
		status   int
		code     string
		expected bool
	}{
		{401, "unauthorized", true},
		{403, "token_disabled", true},
		{403, "forbidden", false},
		{404, "not_found", false},
	}
	for _, tt := range tests {
		err := &APIError{Status: tt.status, Code: tt.code}
		if err.IsAuthError() != tt.expected {
			t.Errorf("status %d, code %s: expected IsAuthError() = %v", tt.status, tt.code, tt.expected)
		}
	}
}

func TestAPIError_IsNotFound(t *testing.T) {
	err := &APIError{Status: 404}
	if !err.IsNotFound() {
		t.Error("expected IsNotFound() to be true for 404")
	}
	err2 := &APIError{Status: 400}
	if err2.IsNotFound() {
		t.Error("expected IsNotFound() to be false for 400")
	}
}

func TestAPIError_IsRateLimited(t *testing.T) {
	err := &APIError{Status: 429}
	if !err.IsRateLimited() {
		t.Error("expected IsRateLimited() to be true for 429")
	}
}

func TestAPIError_IsValidationError(t *testing.T) {
	err := &APIError{Status: 400}
	if !err.IsValidationError() {
		t.Error("expected IsValidationError() to be true for 400")
	}
}

func TestNetworkError(t *testing.T) {
	cause := &http.ProtocolError{ErrorString: "test error"}
	err := &NetworkError{Cause: cause}
	if err.Error() != "nahook: network error: test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestTimeoutError(t *testing.T) {
	err := &TimeoutError{TimeoutMs: 30000}
	if err.Error() != "nahook: request timed out after 30000ms" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestHTTPClient_Request_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("unexpected accept: %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "nahook-go/0.1.0" {
			t.Errorf("unexpected user-agent: %s", r.Header.Get("User-Agent"))
		}
		json.NewEncoder(w).Encode(map[string]string{"key": "value"})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]string
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestHTTPClient_Request_NoContentType_OnGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "" {
			t.Error("GET should not have Content-Type")
		}
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]string
	c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
}

func TestHTTPClient_Request_ContentType_OnPOST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("POST with body should have Content-Type application/json")
		}
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]string
	c.Request(context.Background(), RequestOptions{
		Method: "POST",
		Path:   "/test",
		Body:   map[string]string{"key": "value"},
	}, &result)
}

func TestHTTPClient_Request_204NoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	err := c.Request(context.Background(), RequestOptions{Method: "DELETE", Path: "/test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_Request_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "rate_limited",
				"message": "Slow down",
			},
		})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test_token", BaseURL: srv.URL})
	var result map[string]string
	err := c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 429 {
		t.Errorf("expected 429, got %d", apiErr.Status)
	}
	if apiErr.Code != "rate_limited" {
		t.Errorf("expected code rate_limited, got %s", apiErr.Code)
	}
	if apiErr.RetryAfter == nil || *apiErr.RetryAfter != 60 {
		t.Errorf("expected RetryAfter 60, got %v", apiErr.RetryAfter)
	}
}

func TestHTTPClient_DefaultBaseURL(t *testing.T) {
	c := NewHTTPClient(HTTPClientConfig{Token: "test"})
	// We can't easily test the default URL directly, but we verify it was set
	// by checking the client was created without error
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestHTTPClient_TrailingSlashStripped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/test" {
			t.Errorf("expected /test, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test", BaseURL: srv.URL + "/"})
	var result map[string]string
	c.Request(context.Background(), RequestOptions{Method: "GET", Path: "/test"}, &result)
}

func TestHTTPClient_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "20" {
			t.Errorf("expected offset=20, got %s", r.URL.Query().Get("offset"))
		}
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	c := NewHTTPClient(HTTPClientConfig{Token: "test", BaseURL: srv.URL})
	var result map[string]string
	c.Request(context.Background(), RequestOptions{
		Method: "GET",
		Path:   "/test",
		Query:  map[string]string{"limit": "10", "offset": "20"},
	}, &result)
}

func TestCalculateDelay(t *testing.T) {
	// With retryAfterMs > 0, should return that value
	d := calculateDelay(0, 5000)
	if d.Milliseconds() != 5000 {
		t.Errorf("expected 5000ms, got %dms", d.Milliseconds())
	}

	// Without retryAfter, should return jittered exponential backoff
	d = calculateDelay(0, 0)
	if d.Milliseconds() > 500 {
		t.Errorf("attempt 0 delay should be <= 500ms, got %dms", d.Milliseconds())
	}

	d = calculateDelay(5, 0)
	if d.Milliseconds() > 10000 {
		t.Errorf("delay should be capped at 10000ms, got %dms", d.Milliseconds())
	}
}

func TestPathEncode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ep_123", "ep_123"},
		{"hello world", "hello%20world"},
		{"a/b", "a%2Fb"},
	}
	for _, tt := range tests {
		result := PathEncode(tt.input)
		if result != tt.expected {
			t.Errorf("PathEncode(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
