package security

import "testing"

func TestHashPasswordAndCheck(t *testing.T) {
	hash, err := HashPassword("test-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}

	if err := CheckPassword("test-password", hash); err != nil {
		t.Fatalf("CheckPassword() should succeed, got %v", err)
	}

	if err := CheckPassword("wrong-password", hash); err == nil {
		t.Fatal("CheckPassword() should fail for wrong password")
	}
}
