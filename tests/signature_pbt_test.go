package tests

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Property-based tests for webhook signature computation using pgregory.net/rapid.
//
// These tests verify algebraic properties of HMAC-SHA256 signatures
// without relying on specific known values.

// validBase64Secret generates a random valid whsec_ prefixed secret.
func validBase64Secret(t *rapid.T) string {
	// Generate 16-48 random bytes, base64-encode them, prefix with whsec_
	n := rapid.IntRange(16, 48).Draw(t, "keyLen")
	raw := make([]byte, n)
	for i := range raw {
		raw[i] = byte(rapid.IntRange(0, 255).Draw(t, "keyByte"))
	}
	return "whsec_" + base64.StdEncoding.EncodeToString(raw)
}

// signPBT is the signing function used in property tests (mirrors computeSignature).
func signPBT(secret, msgID, ts, body string) string {
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

// Property 1: sign then verify roundtrip — signature produced by sign
// must match when recomputed with the same inputs.
func TestPBT_SignVerifyRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		secret := validBase64Secret(t)
		msgID := "msg_" + rapid.StringMatching(`[a-zA-Z0-9]{8,20}`).Draw(t, "msgID")
		ts := fmt.Sprintf("%d", rapid.Int64Range(1000000000, 9999999999).Draw(t, "ts"))
		payload := rapid.String().Draw(t, "payload")

		sig1 := signPBT(secret, msgID, ts, payload)
		sig2 := signPBT(secret, msgID, ts, payload)

		// Verify the signature is valid format
		if !strings.HasPrefix(sig1, "v1,") {
			t.Fatalf("signature missing v1, prefix: %s", sig1)
		}

		// Verify roundtrip: recomputing with same inputs yields same sig
		if sig1 != sig2 {
			t.Fatalf("roundtrip failed: %s != %s", sig1, sig2)
		}
	})
}

// Property 2: tampered payload produces different signature.
func TestPBT_TamperedPayloadFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		secret := validBase64Secret(t)
		msgID := "msg_" + rapid.StringMatching(`[a-zA-Z0-9]{8,20}`).Draw(t, "msgID")
		ts := fmt.Sprintf("%d", rapid.Int64Range(1000000000, 9999999999).Draw(t, "ts"))
		payload1 := rapid.String().Draw(t, "payload1")
		payload2 := rapid.String().Draw(t, "payload2")

		// Only assert different sigs when payloads actually differ
		if payload1 == payload2 {
			return
		}

		sig1 := signPBT(secret, msgID, ts, payload1)
		sig2 := signPBT(secret, msgID, ts, payload2)

		if sig1 == sig2 {
			t.Fatalf("different payloads produced same signature: %q vs %q", payload1, payload2)
		}
	})
}

// Property 3: wrong secret produces different signature.
func TestPBT_WrongSecretFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		secret1 := validBase64Secret(t)
		secret2 := validBase64Secret(t)
		msgID := "msg_" + rapid.StringMatching(`[a-zA-Z0-9]{8,20}`).Draw(t, "msgID")
		ts := fmt.Sprintf("%d", rapid.Int64Range(1000000000, 9999999999).Draw(t, "ts"))
		payload := rapid.String().Draw(t, "payload")

		// Only assert when secrets differ
		if secret1 == secret2 {
			return
		}

		sig1 := signPBT(secret1, msgID, ts, payload)
		sig2 := signPBT(secret2, msgID, ts, payload)

		if sig1 == sig2 {
			t.Fatalf("different secrets produced same signature")
		}
	})
}

// Property 4: deterministic — same inputs always produce identical output.
func TestPBT_Deterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		secret := validBase64Secret(t)
		msgID := "msg_" + rapid.StringMatching(`[a-zA-Z0-9]{8,20}`).Draw(t, "msgID")
		ts := fmt.Sprintf("%d", rapid.Int64Range(1000000000, 9999999999).Draw(t, "ts"))
		payload := rapid.String().Draw(t, "payload")

		results := make([]string, 5)
		for i := range results {
			results[i] = signPBT(secret, msgID, ts, payload)
		}

		for i := 1; i < len(results); i++ {
			if results[i] != results[0] {
				t.Fatalf("non-deterministic: iteration %d produced %s, expected %s", i, results[i], results[0])
			}
		}
	})
}
