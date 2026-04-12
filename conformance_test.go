package nahook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Conformance fixtures ────────────────────────────────────────────────────
//
// These tests load JSON fixtures from fixtures/conformance/ and assert that
// the Go SDK behaviour matches the cross-language contract.
// If fixtures are not present yet, the tests skip gracefully.

func loadFixtures[T any](t *testing.T, name string) []T {
	t.Helper()
	path := filepath.Join("fixtures", "conformance", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture %s not found, skipping: %v", name, err)
	}
	var out []T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to parse fixture %s: %v", name, err)
	}
	return out
}

// ── Error classification conformance ────────────────────────────────────────

type errorFixture struct {
	Status        int    `json:"status"`
	Code          string `json:"code"`
	IsRetryable   bool   `json:"isRetryable"`
	IsAuthError   bool   `json:"isAuthError"`
	IsNotFound    bool   `json:"isNotFound"`
	IsRateLimited bool   `json:"isRateLimited"`
	IsValidation  bool   `json:"isValidationError"`
}

func TestConformance_ErrorClassification(t *testing.T) {
	fixtures := loadFixtures[errorFixture](t, "error-classification.json")
	for i, f := range fixtures {
		err := &APIError{Status: f.Status, Code: f.Code}
		if err.IsRetryable() != f.IsRetryable {
			t.Errorf("fixture[%d] status=%d code=%s: IsRetryable() = %v, want %v", i, f.Status, f.Code, err.IsRetryable(), f.IsRetryable)
		}
		if err.IsAuthError() != f.IsAuthError {
			t.Errorf("fixture[%d] status=%d code=%s: IsAuthError() = %v, want %v", i, f.Status, f.Code, err.IsAuthError(), f.IsAuthError)
		}
		if err.IsNotFound() != f.IsNotFound {
			t.Errorf("fixture[%d] status=%d code=%s: IsNotFound() = %v, want %v", i, f.Status, f.Code, err.IsNotFound(), f.IsNotFound)
		}
		if err.IsRateLimited() != f.IsRateLimited {
			t.Errorf("fixture[%d] status=%d code=%s: IsRateLimited() = %v, want %v", i, f.Status, f.Code, err.IsRateLimited(), f.IsRateLimited)
		}
		if err.IsValidationError() != f.IsValidation {
			t.Errorf("fixture[%d] status=%d code=%s: IsValidationError() = %v, want %v", i, f.Status, f.Code, err.IsValidationError(), f.IsValidation)
		}
	}
}

// ── Region routing conformance ──────────────────────────────────────────────

type regionFixture struct {
	Token    string `json:"token"`
	Expected string `json:"expectedBaseURL"`
}

func TestConformance_RegionRouting(t *testing.T) {
	fixtures := loadFixtures[regionFixture](t, "region-routing.json")
	for i, f := range fixtures {
		got := ResolveBaseURL(f.Token)
		if got != f.Expected {
			t.Errorf("fixture[%d] token=%s: ResolveBaseURL() = %s, want %s", i, f.Token, got, f.Expected)
		}
	}
}

// ── Retry backoff conformance ───────────────────────────────────────────────

type retryFixture struct {
	Attempt      int  `json:"attempt"`
	RetryAfterMs int  `json:"retryAfterMs"`
	ExpectedMs   *int `json:"expectedMs"` // exact match when retryAfterMs > 0
	MaxMs        *int `json:"maxMs"`      // upper bound for jittered delay
}

func TestConformance_RetryBackoff(t *testing.T) {
	fixtures := loadFixtures[retryFixture](t, "retry-backoff.json")
	for i, f := range fixtures {
		d := calculateDelay(f.Attempt, f.RetryAfterMs)
		ms := int(d.Milliseconds())

		if f.ExpectedMs != nil {
			if ms != *f.ExpectedMs {
				t.Errorf("fixture[%d] attempt=%d retryAfterMs=%d: got %dms, want exactly %dms", i, f.Attempt, f.RetryAfterMs, ms, *f.ExpectedMs)
			}
		}
		if f.MaxMs != nil {
			if ms < 0 || ms > *f.MaxMs {
				t.Errorf("fixture[%d] attempt=%d retryAfterMs=%d: got %dms, want [0, %d]ms", i, f.Attempt, f.RetryAfterMs, ms, *f.MaxMs)
			}
		}
	}
}

// ── Signature conformance ───────────────────────────────────────────────────

type signatureFixture struct {
	Secret    string `json:"secret"`
	MsgID     string `json:"msgId"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"`
	Expected  string `json:"expectedSignature"`
}

func TestConformance_Signature(t *testing.T) {
	fixtures := loadFixtures[signatureFixture](t, "signature.json")
	for i, f := range fixtures {
		sig := conformanceSign(f.Secret, f.MsgID, f.Timestamp, f.Payload)
		if sig != f.Expected {
			t.Errorf("fixture[%d]: got %s, want %s", i, sig, f.Expected)
		}
	}
}

// conformanceSign implements the Standard Webhooks signing algorithm inline
// so the conformance test is self-contained within package nahook.
func conformanceSign(secret, msgID, ts, body string) string {
	rawSecret := secret
	if strings.HasPrefix(rawSecret, "whsec_") {
		rawSecret = rawSecret[6:]
	}
	key, err := base64.StdEncoding.DecodeString(rawSecret)
	if err != nil {
		panic("invalid base64 secret: " + err.Error())
	}
	toSign := fmt.Sprintf("%s.%s.%s", msgID, ts, body)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(toSign))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
