package security

import "testing"

func TestJWTGenerateAndParse(t *testing.T) {
	manager := NewJWTManager("unit-test-secret", 10)

	token, err := manager.GenerateToken("user-1", "u@test.com", "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := manager.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.UserID != "user-1" || claims.Email != "u@test.com" || claims.Role != "user" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}
