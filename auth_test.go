package privy

import (
	"testing"
	"time"
)

func TestBase64URLDecode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard base64url",
			input:    "SGVsbG8gV29ybGQ",
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:     "with padding",
			input:    "SGVsbG8=",
			expected: "Hello",
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := base64URLDecode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("base64URLDecode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("base64URLDecode() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestTokenClaims_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		expected  bool
	}{
		{
			name:      "not expired",
			expiresAt: time.Now().Add(1 * time.Hour).Unix(),
			expected:  false,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-1 * time.Hour).Unix(),
			expected:  true,
		},
		{
			name:      "zero value",
			expiresAt: 0,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &TokenClaims{ExpiresAt: tt.expiresAt}
			if got := claims.IsExpired(); got != tt.expected {
				t.Errorf("TokenClaims.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTokenClaims_UserID(t *testing.T) {
	claims := &TokenClaims{Subject: "did:privy:abc123"}
	if got := claims.UserID(); got != "did:privy:abc123" {
		t.Errorf("TokenClaims.UserID() = %v, want %v", got, "did:privy:abc123")
	}
}

func TestTokenClaims_GetClaim(t *testing.T) {
	claims := &TokenClaims{
		Raw: map[string]any{
			"custom_field": "custom_value",
			"number_field": float64(123),
		},
	}

	// Test existing claim
	val, ok := claims.GetClaim("custom_field")
	if !ok {
		t.Error("GetClaim() should return true for existing claim")
	}
	if val != "custom_value" {
		t.Errorf("GetClaim() = %v, want %v", val, "custom_value")
	}

	// Test non-existing claim
	_, ok = claims.GetClaim("non_existing")
	if ok {
		t.Error("GetClaim() should return false for non-existing claim")
	}

	// Test nil Raw
	nilClaims := &TokenClaims{}
	_, ok = nilClaims.GetClaim("any")
	if ok {
		t.Error("GetClaim() should return false for nil Raw")
	}
}

func TestTokenClaims_ExpiresIn(t *testing.T) {
	// Test with future expiration
	futureExpires := time.Now().Add(1 * time.Hour).Unix()
	claims := &TokenClaims{ExpiresAt: futureExpires}
	duration := claims.ExpiresIn()

	// Should be roughly 1 hour (with some tolerance)
	if duration < 59*time.Minute || duration > 61*time.Minute {
		t.Errorf("ExpiresIn() = %v, expected roughly 1 hour", duration)
	}

	// Test with zero expiration
	zeroClaims := &TokenClaims{ExpiresAt: 0}
	if zeroClaims.ExpiresIn() != 0 {
		t.Errorf("ExpiresIn() should return 0 for zero expiration")
	}
}

func TestJWKToRSAPublicKey(t *testing.T) {
	// Test with valid JWK
	jwk := &JWK{
		Kty: "RSA",
		N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
		E:   "AQAB",
	}

	pubKey, err := jwkToRSAPublicKey(jwk)
	if err != nil {
		t.Fatalf("jwkToRSAPublicKey() error = %v", err)
	}

	if pubKey.E != 65537 {
		t.Errorf("Expected exponent 65537, got %d", pubKey.E)
	}

	if pubKey.N == nil {
		t.Error("Expected non-nil modulus")
	}
}

func TestNewAuthService(t *testing.T) {
	client := NewClient("app-id", "app-secret")
	auth := client.Auth()

	if auth == nil {
		t.Fatal("Auth() should not return nil")
	}

	if auth.client != client {
		t.Error("Auth service should have reference to client")
	}

	if auth.jwksCache == nil {
		t.Error("Auth service should have initialized JWKS cache")
	}
}

func TestVerifyToken_InvalidFormat(t *testing.T) {
	client := NewClient("app-id", "app-secret")
	auth := client.Auth()

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"single part", "header"},
		{"two parts", "header.payload"},
		{"four parts", "a.b.c.d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.VerifyToken(nil, tt.token)
			if err == nil {
				t.Error("Expected error for invalid token format")
			}
		})
	}
}
