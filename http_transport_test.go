package nahook_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	nahook "github.com/getnahook/nahook-go"
	"github.com/getnahook/nahook-go/client"
	"github.com/getnahook/nahook-go/management"
)

// ── Pass 1: default transport config ───────────────────────────────────────

func TestDefaultHTTPClientUsesCustomTransport(t *testing.T) {
	c := nahook.NewHTTPClient(nahook.HTTPClientConfig{Token: "nhk_us_test"})
	httpClient := c.HTTPClient()

	tr, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected default Transport to be *http.Transport, got %T", httpClient.Transport)
	}

	if got, want := tr.MaxIdleConnsPerHost, 50; got != want {
		t.Errorf("MaxIdleConnsPerHost: got %d, want %d", got, want)
	}
	if got, want := tr.MaxIdleConns, 100; got != want {
		t.Errorf("MaxIdleConns: got %d, want %d", got, want)
	}
	if got, want := tr.IdleConnTimeout, 90*time.Second; got != want {
		t.Errorf("IdleConnTimeout: got %v, want %v", got, want)
	}
	if !tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2: got false, want true")
	}
}

func TestDefaultHTTPClientTimeoutMatchesConfig(t *testing.T) {
	c := nahook.NewHTTPClient(nahook.HTTPClientConfig{
		Token:   "nhk_us_test",
		Timeout: 5 * time.Second,
	})
	if got, want := c.HTTPClient().Timeout, 5*time.Second; got != want {
		t.Errorf("Timeout: got %v, want %v", got, want)
	}
}

// ── Pass 2: BYO http.Client ────────────────────────────────────────────────

func TestHTTPClientConfig_HTTPClient_UsedVerbatim(t *testing.T) {
	custom := &http.Client{Timeout: 7 * time.Second}
	c := nahook.NewHTTPClient(nahook.HTTPClientConfig{
		Token:      "nhk_us_test",
		HTTPClient: custom,
	})

	if c.HTTPClient() != custom {
		t.Fatal("SDK should use the supplied *http.Client verbatim (pointer equality)")
	}
}

func TestHTTPClientConfig_HTTPClient_TimeoutDrivesTimeoutError(t *testing.T) {
	// Caller's *http.Client.Timeout governs request timeouts AND is what
	// TimeoutError.TimeoutMs reports.
	custom := &http.Client{
		Timeout:   50 * time.Millisecond,
		Transport: &slowRoundTripper{delay: 500 * time.Millisecond},
	}
	c, err := client.New("nhk_us_test", client.WithHTTPClient(custom))
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Send(context.Background(), "ep_abc", nahook.SendOptions{
		Payload: map[string]interface{}{"x": 1},
	})

	te, ok := err.(*nahook.TimeoutError)
	if !ok {
		t.Fatalf("expected *nahook.TimeoutError, got %T: %v", err, err)
	}
	if te.TimeoutMs != 50 {
		t.Errorf("TimeoutError.TimeoutMs: got %d, want 50", te.TimeoutMs)
	}
}

func TestClient_WithHTTPClient_FunnelsThroughCustom(t *testing.T) {
	counter := &countingRoundTripper{}
	custom := &http.Client{Transport: counter}

	c, err := client.New("nhk_us_test",
		client.WithBaseURL("https://test.nahook.com"),
		client.WithHTTPClient(custom),
	)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		_, _ = c.Send(context.Background(), "ep_abc", nahook.SendOptions{
			Payload: map[string]interface{}{"i": i},
		})
	}

	if counter.count != 3 {
		t.Errorf("expected 3 requests to custom transport, got %d", counter.count)
	}
}

func TestManagement_WithHTTPClient_FunnelsThroughCustom(t *testing.T) {
	counter := &countingRoundTripper{}
	custom := &http.Client{Transport: counter}

	m, err := management.New("nhm_test",
		management.WithBaseURL("https://test.nahook.com"),
		management.WithHTTPClient(custom),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = m.Endpoints.List(context.Background(), "ws_abc")

	if counter.count != 1 {
		t.Errorf("expected 1 request to custom transport, got %d", counter.count)
	}
}

// ── HTTP client is not reconstructed per request ──────────────────────────

func TestSDKDoesNotReconstructHTTPClientPerRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintln(w, `{"deliveryId":"del_1","idempotencyKey":"k","status":"accepted"}`)
	}))
	defer server.Close()

	c, err := client.New("nhk_us_test", client.WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	// Snapshot internal *http.Client before any calls.
	httpClientBefore := nahookHTTPClient(c)

	for i := 0; i < 5; i++ {
		_, err := c.Send(context.Background(), "ep_abc", nahook.SendOptions{
			Payload: map[string]interface{}{"i": i},
		})
		if err != nil {
			t.Fatalf("send %d failed: %v", i, err)
		}
	}

	if nahookHTTPClient(c) != httpClientBefore {
		t.Error("SDK reconstructed its *http.Client across calls — should reuse the same instance")
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

// nahookHTTPClient reaches into the Client to fetch the internal *http.Client
// it's using. Uses the public HTTPClient() inspector on client.Client.
func nahookHTTPClient(c *client.Client) *http.Client {
	return c.HTTPClient()
}

type slowRoundTripper struct {
	delay time.Duration
}

func (s *slowRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	select {
	case <-time.After(s.delay):
		// Build a minimal accepted response if we ever got here.
		body, _ := json.Marshal(map[string]interface{}{
			"deliveryId": "del_x", "idempotencyKey": "k", "status": "accepted",
		})
		resp := &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       http.NoBody,
		}
		_ = body
		return resp, nil
	case <-r.Context().Done():
		return nil, r.Context().Err()
	}
}

type countingRoundTripper struct {
	count int
}

func (c *countingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	c.count++
	// Return a generic 200 with minimal JSON; caller paths tolerate decoding noise.
	body := `{"deliveryId":"del_1","idempotencyKey":"k","status":"accepted","data":[]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       http.NoBody,
		Request:    r,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1, ProtoMinor: 1,
		ContentLength: int64(len(body)),
	}, nil
}
