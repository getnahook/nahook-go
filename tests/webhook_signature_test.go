package tests

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// Webhook signature verification tests.
//
// Validates that the Standard Webhooks signing format used by the Nahook API
// can be correctly produced and verified using native crypto.
//
// Signing spec:
//
//	base   = "{msgId}.{timestamp}.{payload}"
//	key    = base64_decode(secret_without_whsec_prefix)
//	sig    = "v1," + base64(HMAC-SHA256(key, base))
//	headers: webhook-id, webhook-timestamp, webhook-signature

const (
	testSecret = "whsec_dGVzdF93ZWJob29rX3NpZ25pbmdfa2V5XzMyYnl0ZXMh"
	msgID      = "msg_test_sig_001"
	timestamp  = "1712345678"
	payload    = `{"order_id":"ord_123","amount":49.99}`
)

func computeSignature(secret, msgId, ts, body string) string {
	rawSecret := secret
	if strings.HasPrefix(rawSecret, "whsec_") {
		rawSecret = rawSecret[6:]
	}
	key, err := base64.StdEncoding.DecodeString(rawSecret)
	if err != nil {
		panic("secret must be valid base64: " + err.Error())
	}

	toSign := fmt.Sprintf("%s.%s.%s", msgId, ts, body)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(toSign))
	digest := mac.Sum(nil)
	return "v1," + base64.StdEncoding.EncodeToString(digest)
}

func TestProducesValidV1Signature(t *testing.T) {
	sig := computeSignature(testSecret, msgID, timestamp, payload)
	matched, _ := regexp.MatchString(`^v1,[A-Za-z0-9+/]+=*$`, sig)
	if !matched {
		t.Fatalf("signature does not match v1 format: %s", sig)
	}
}

func TestDeterministicSameInputsSameSignature(t *testing.T) {
	sig1 := computeSignature(testSecret, msgID, timestamp, payload)
	sig2 := computeSignature(testSecret, msgID, timestamp, payload)
	if sig1 != sig2 {
		t.Fatalf("expected same signature, got %s and %s", sig1, sig2)
	}
}

func TestRejectsTamperedPayload(t *testing.T) {
	original := computeSignature(testSecret, msgID, timestamp, payload)
	tampered := computeSignature(testSecret, msgID, timestamp, `{"order_id":"ord_123","amount":99.99}`)
	if original == tampered {
		t.Fatal("tampered payload should produce a different signature")
	}
}

func TestRejectsWrongSecret(t *testing.T) {
	original := computeSignature(testSecret, msgID, timestamp, payload)
	wrong := computeSignature("whsec_d3Jvbmdfc2VjcmV0", msgID, timestamp, payload)
	if original == wrong {
		t.Fatal("wrong secret should produce a different signature")
	}
}

func TestRejectsTamperedMsgId(t *testing.T) {
	original := computeSignature(testSecret, msgID, timestamp, payload)
	tampered := computeSignature(testSecret, "msg_tampered_id", timestamp, payload)
	if original == tampered {
		t.Fatal("tampered msgId should produce a different signature")
	}
}

func TestRejectsTamperedTimestamp(t *testing.T) {
	original := computeSignature(testSecret, msgID, timestamp, payload)
	tampered := computeSignature(testSecret, msgID, "9999999999", payload)
	if original == tampered {
		t.Fatal("tampered timestamp should produce a different signature")
	}
}

func TestCorrectHeadersStructure(t *testing.T) {
	sig := computeSignature(testSecret, msgID, timestamp, payload)
	headers := map[string]string{
		"content-type":      "application/json",
		"webhook-id":        msgID,
		"webhook-timestamp": timestamp,
		"webhook-signature": sig,
	}

	if !strings.HasPrefix(headers["webhook-id"], "msg_") {
		t.Fatalf("webhook-id should start with msg_, got: %s", headers["webhook-id"])
	}
	if !strings.HasPrefix(headers["webhook-signature"], "v1,") {
		t.Fatalf("webhook-signature should start with v1,, got: %s", headers["webhook-signature"])
	}
	if headers["content-type"] != "application/json" {
		t.Fatalf("content-type should be application/json, got: %s", headers["content-type"])
	}
}

func TestHandlesSecretWithoutPrefix(t *testing.T) {
	rawSecret := testSecret[6:]
	withPrefix := computeSignature(testSecret, msgID, timestamp, payload)
	withoutPrefix := computeSignature(rawSecret, msgID, timestamp, payload)
	if withPrefix != withoutPrefix {
		t.Fatalf("with/without prefix should produce same signature, got %s and %s", withPrefix, withoutPrefix)
	}
}

func TestMatchesKnownCrossLanguageReferenceSignature(t *testing.T) {
	sig := computeSignature(testSecret, msgID, timestamp, payload)
	expected := "v1,VF1JBS4kdSwmE64FeeiWTgszlPCfaop53x8bwzvHizw="
	if sig != expected {
		t.Fatalf("expected %s, got %s", expected, sig)
	}
}

func TestEmptyPayloadProducesValidSignature(t *testing.T) {
	sig := computeSignature(testSecret, msgID, timestamp, "")
	expected := "v1,yNFeVvBSs4aZ/sVHHw1MaUWnN1IGK/Ul/16T8aptSJo="
	if sig != expected {
		t.Fatalf("expected %s, got %s", expected, sig)
	}
}

func TestUnicodePayloadConsistentAcrossLanguages(t *testing.T) {
	unicodePayload := `{"name":"café","price":"€9.99"}`
	sig := computeSignature(testSecret, msgID, timestamp, unicodePayload)
	expected := "v1,GcuGAMV9tELnF2rjay6sA8uo5PDPPlhaFi6gKUg06wQ="
	if sig != expected {
		t.Fatalf("expected %s, got %s", expected, sig)
	}
}
