package security

import (
	"testing"
	"time"
)

func TestInMemoryTokenBlacklist_RevokeAndCheck(t *testing.T) {
	blacklist := NewInMemoryTokenBlacklist()
	blacklist.Revoke("token-1", time.Now().Add(time.Minute))

	if !blacklist.IsRevoked("token-1") {
		t.Fatal("expected token to be revoked")
	}
}

func TestInMemoryTokenBlacklist_ExpiredTokenIsCleaned(t *testing.T) {
	blacklist := NewInMemoryTokenBlacklist()
	blacklist.Revoke("token-1", time.Now().Add(-time.Minute))

	if blacklist.IsRevoked("token-1") {
		t.Fatal("expected expired token to be cleaned up")
	}
}
