package auth_test

import (
	"testing"
	"time"

	"github.com/chris-konkol/triage/internal/auth"
)

func TestGenerate_Parse_RoundTrip(t *testing.T) {
	token, err := auth.Generate("uid1", "alice", "admin", "secret")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	claims, err := auth.Parse(token, "secret")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.UserID != "uid1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "uid1")
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want %q", claims.Username, "alice")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
}

func TestGenerate_ExpiresInFuture(t *testing.T) {
	token, _ := auth.Generate("uid1", "alice", "admin", "secret")
	claims, err := auth.Parse(token, "secret")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Error("expected token to expire in the future")
	}
}

func TestParse_WrongSecret(t *testing.T) {
	token, _ := auth.Generate("uid1", "alice", "admin", "correct-secret")
	_, err := auth.Parse(token, "wrong-secret")
	if err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestParse_Malformed(t *testing.T) {
	_, err := auth.Parse("not.a.valid.token", "secret")
	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestParse_Empty(t *testing.T) {
	_, err := auth.Parse("", "secret")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}
