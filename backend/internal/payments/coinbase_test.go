package payments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestCoinbaseWebhookVerification(t *testing.T) {
	c := newCoinbaseClient("api-key", "shared-secret")
	body := []byte(`{"event":{"id":"evt_1","type":"charge:confirmed","data":{"code":"ABC123"}}}`)

	valid := sign("shared-secret", body)
	if !c.verifyWebhook(body, valid) {
		t.Errorf("valid signature should verify")
	}
	if c.verifyWebhook(body, "deadbeef") {
		t.Errorf("garbage signature must not verify")
	}
	if c.verifyWebhook(body, sign("wrong-secret", body)) {
		t.Errorf("signature under the wrong secret must not verify")
	}
	tampered := append([]byte{}, body...)
	tampered[10] ^= 0xFF
	if c.verifyWebhook(tampered, valid) {
		t.Errorf("tampered body must not verify against the original signature")
	}
}

func TestParseCoinbaseEvent(t *testing.T) {
	body := []byte(`{"event":{"id":"evt_9","type":"charge:resolved","data":{"code":"XYZ","metadata":{"order_id":"o-1"}}}}`)
	e, err := parseCoinbaseEvent(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if e.Event.ID != "evt_9" || e.Event.Type != "charge:resolved" || e.Event.Data.Code != "XYZ" {
		t.Errorf("parsed event mismatch: %+v", e.Event)
	}
	if e.Event.Data.Metadata["order_id"] != "o-1" {
		t.Errorf("metadata order_id not parsed")
	}
}
