package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	ts := NewTokenService("test-secret", 15*time.Minute, time.Hour)
	uid := uuid.New()

	tok, exp, err := ts.IssueAccessToken(uid, "admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !exp.After(time.Now()) {
		t.Fatalf("expiry should be in the future")
	}
	claims, err := ts.ParseAccessToken(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != uid.String() {
		t.Errorf("subject = %q, want %q", claims.Subject, uid.String())
	}
	if claims.Role != "admin" {
		t.Errorf("role = %q, want admin", claims.Role)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	a := NewTokenService("secret-a", time.Minute, time.Hour)
	b := NewTokenService("secret-b", time.Minute, time.Hour)
	tok, _, _ := a.IssueAccessToken(uuid.New(), "customer")
	if _, err := b.ParseAccessToken(tok); err == nil {
		t.Fatalf("expected parse to fail with a different secret")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	ts := NewTokenService("secret", -time.Minute, time.Hour) // already expired
	tok, _, _ := ts.IssueAccessToken(uuid.New(), "customer")
	if _, err := ts.ParseAccessToken(tok); err == nil {
		t.Fatalf("expected expired token to be rejected")
	}
}

func TestRefreshTokenHashing(t *testing.T) {
	raw, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("new refresh: %v", err)
	}
	if raw == hash {
		t.Fatalf("raw token must not equal its hash")
	}
	if HashToken(raw) != hash {
		t.Fatalf("HashToken must be deterministic and match")
	}
}

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("supersecret", 10)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "supersecret") {
		t.Errorf("correct password should verify")
	}
	if CheckPassword(hash, "wrong") {
		t.Errorf("wrong password must not verify")
	}
}
