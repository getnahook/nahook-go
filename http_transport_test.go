package nahook_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
		Transport: &slowRoundTripper{},
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
	// Regression guard. The SDK's *http.Client field is set in the constructor
	// and never re-assigned. No public API today can mutate it post-construction —
	// this test exists to catch a future change that breaks that invariant.
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
	httpClientBefore := c.HTTPClient()

	for i := 0; i < 5; i++ {
		_, err := c.Send(context.Background(), "ep_abc", nahook.SendOptions{
			Payload: map[string]interface{}{"i": i},
		})
		if err != nil {
			t.Fatalf("send %d failed: %v", i, err)
		}
	}

	if c.HTTPClient() != httpClientBefore {
		t.Error("SDK reconstructed its *http.Client across calls — should reuse the same instance")
	}
}

// ── net.Error.Timeout() classification (Go-version-independent) ───────────

func TestTimeoutErrorClassification_FromNetErrorTimeoutInterface(t *testing.T) {
	// Pin the classification path: a RoundTripper that returns an error
	// implementing net.Error.Timeout() == true MUST surface as TimeoutError.
	// Regression guard against Go-version-dependent http.Client error wrapping:
	// in some versions Client.Timeout produces a *url.Error wrapping
	// context.DeadlineExceeded (errors.Is catches it), in others an unexported
	// *httpError that only implements net.Error (errors.Is does NOT catch it).
	custom := &http.Client{
		Transport: &timeoutRoundTripper{},
	}
	c, err := client.New("nhk_us_test",
		client.WithBaseURL("https://test.nahook.com"),
		client.WithHTTPClient(custom),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Send(context.Background(), "ep_abc", nahook.SendOptions{
		Payload: map[string]interface{}{"x": 1},
	})

	if _, ok := err.(*nahook.TimeoutError); !ok {
		t.Fatalf("expected *nahook.TimeoutError, got %T: %v", err, err)
	}
}

// ── Close() lifecycle ──────────────────────────────────────────────────────

func TestClose_OnDefaultClient_DoesNotPanic(t *testing.T) {
	c := nahook.NewHTTPClient(nahook.HTTPClientConfig{Token: "nhk_us_test"})
	// Should not panic; calling twice should also be safe — Go's
	// http.Transport.CloseIdleConnections is naturally idempotent.
	c.Close()
	c.Close()
}

func TestClose_OnBYOClient_DoesNotTouchCallerTransport(t *testing.T) {
	// The caller-owned *http.Client has lifecycle owned by the caller.
	// SDK Close() must NOT trigger CloseIdleConnections on the caller's transport.
	spy := &closeIdleSpyRoundTripper{}
	custom := &http.Client{Transport: spy}

	c := nahook.NewHTTPClient(nahook.HTTPClientConfig{
		Token:      "nhk_us_test",
		HTTPClient: custom,
	})

	c.Close()

	if spy.closeCount != 0 {
		t.Errorf("expected caller-owned transport untouched, got CloseIdleConnections call count = %d", spy.closeCount)
	}
}

func TestClient_Close_DelegatesToHTTPClient(t *testing.T) {
	c, err := client.New("nhk_us_test")
	if err != nil {
		t.Fatal(err)
	}
	// Smoke test: no panic on default-built client.
	c.Close()
}

func TestClient_Close_IsIdempotent(t *testing.T) {
	// Public-API-level idempotency guarantee: defer + explicit Close,
	// or two defers in different scopes, must not panic.
	c, err := client.New("nhk_us_test")
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
	c.Close()
}

func TestManagement_Close_DelegatesToHTTPClient(t *testing.T) {
	m, err := management.New("nhm_test")
	if err != nil {
		t.Fatal(err)
	}
	m.Close()
}

func TestClient_Close_OnBYOClient_DoesNotTouchCallerTransport(t *testing.T) {
	spy := &closeIdleSpyRoundTripper{}
	custom := &http.Client{Transport: spy}

	c, err := client.New("nhk_us_test", client.WithHTTPClient(custom))
	if err != nil {
		t.Fatal(err)
	}

	c.Close()

	if spy.closeCount != 0 {
		t.Errorf("client.Close() touched caller's transport: CloseIdleConnections count = %d", spy.closeCount)
	}
}

// ── Round-tripper test doubles ─────────────────────────────────────────────

// slowRoundTripper blocks until the request context is cancelled — used for
// timeout-precedence tests. Has no success branch: the test paths driving it
// always cancel via http.Client.Timeout before any response would be returned.
type slowRoundTripper struct{}

func (s *slowRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()
	return nil, r.Context().Err()
}

// timeoutRoundTripper returns an error that satisfies the net.Error interface
// with Timeout() == true — mirrors the shape Go's http.Client produces when
// Client.Timeout fires. Used to pin the SDK's timeout-classification path.
type timeoutRoundTripper struct{}

func (t *timeoutRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, &netTimeoutError{}
}

type netTimeoutError struct{}

func (e *netTimeoutError) Error() string   { return "i/o timeout" }
func (e *netTimeoutError) Timeout() bool   { return true }
func (e *netTimeoutError) Temporary() bool { return true }

// closeIdleSpyRoundTripper implements CloseIdleConnections so it satisfies the
// http.Client.CloseIdleConnections dispatch path. Used to verify the SDK's
// Close() does NOT propagate to a caller-owned *http.Client's transport.
type closeIdleSpyRoundTripper struct {
	closeCount int
}

func (s *closeIdleSpyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{}`)),
		Request:    r,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}, nil
}

func (s *closeIdleSpyRoundTripper) CloseIdleConnections() {
	s.closeCount++
}

// countingRoundTripper returns a minimal 202-accepted ingest response and
// increments count on every RoundTrip call. The response body is real JSON
// the SDK can decode, so the test exercises the full happy path.
type countingRoundTripper struct {
	count int
}

func (c *countingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	c.count++
	body := `{"deliveryId":"del_1","idempotencyKey":"k","status":"accepted"}`
	return &http.Response{
		StatusCode:    http.StatusAccepted,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       r,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}, nil
}
