package privy

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AuthService handles authentication and token verification.
type AuthService struct {
	client    *Client
	jwksCache *jwksCache
}

// jwksCache caches JWKS keys with expiration.
type jwksCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	ttl       time.Duration
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kty string `json:"kty"` // Key type (RSA)
	Use string `json:"use"` // Key use (sig)
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm (RS256)
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
}

// TokenClaims represents the claims in a Privy access token.
type TokenClaims struct {
	// Standard JWT claims
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"` // Privy user ID (did:privy:...)
	Audience  string `json:"aud"` // Your app ID
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf,omitempty"`
	JWTID     string `json:"jti,omitempty"`

	// Privy-specific claims
	AppID     string `json:"app_id,omitempty"`
	SessionID string `json:"sid,omitempty"`

	// Raw claims for accessing additional data
	Raw map[string]any `json:"-"`
}

// VerifyTokenOptions contains options for token verification.
type VerifyTokenOptions struct {
	// Audience to validate against (defaults to client's app ID)
	Audience string
	// AllowExpired skips expiration check (useful for debugging)
	AllowExpired bool
	// ClockSkew allows for clock differences between servers
	ClockSkew time.Duration
}

var (
	ErrInvalidToken     = errors.New("privy: invalid token")
	ErrTokenExpired     = errors.New("privy: token expired")
	ErrInvalidAudience  = errors.New("privy: invalid audience")
	ErrInvalidIssuer    = errors.New("privy: invalid issuer")
	ErrInvalidSignature = errors.New("privy: invalid signature")
	ErrKeyNotFound      = errors.New("privy: signing key not found")
)

const (
	privyIssuer  = "privy.io"
	jwksCacheTTL = 1 * time.Hour
	jwksEndpoint = "https://auth.privy.io/.well-known/jwks.json"
)

func newAuthService(client *Client) *AuthService {
	return &AuthService{
		client: client,
		jwksCache: &jwksCache{
			keys: make(map[string]*rsa.PublicKey),
			ttl:  jwksCacheTTL,
		},
	}
}

// VerifyToken verifies a Privy access token and returns its claims.
func (s *AuthService) VerifyToken(ctx context.Context, token string) (*TokenClaims, error) {
	return s.VerifyTokenWithOptions(ctx, token, nil)
}

// VerifyTokenWithOptions verifies a Privy access token with custom options.
func (s *AuthService) VerifyTokenWithOptions(ctx context.Context, token string, opts *VerifyTokenOptions) (*TokenClaims, error) {
	if opts == nil {
		opts = &VerifyTokenOptions{}
	}

	// Parse the token
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Decode header
	headerBytes, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid header encoding", ErrInvalidToken)
	}

	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("%w: invalid header", ErrInvalidToken)
	}

	if header.Alg != "RS256" {
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrInvalidToken, header.Alg)
	}

	// Decode payload
	payloadBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid payload encoding", ErrInvalidToken)
	}

	var claims TokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("%w: invalid payload", ErrInvalidToken)
	}

	// Also store raw claims
	var rawClaims map[string]any
	if err := json.Unmarshal(payloadBytes, &rawClaims); err != nil {
		return nil, fmt.Errorf("%w: invalid payload", ErrInvalidToken)
	}
	claims.Raw = rawClaims

	// Verify signature
	if err := s.verifySignature(ctx, parts[0]+"."+parts[1], parts[2], header.Kid); err != nil {
		return nil, err
	}

	// Verify claims
	now := time.Now()
	clockSkew := opts.ClockSkew
	if clockSkew == 0 {
		clockSkew = 30 * time.Second
	}

	// Check expiration
	if !opts.AllowExpired {
		if claims.ExpiresAt > 0 && time.Unix(claims.ExpiresAt, 0).Add(clockSkew).Before(now) {
			return nil, ErrTokenExpired
		}
	}

	// Check not before
	if claims.NotBefore > 0 && time.Unix(claims.NotBefore, 0).Add(-clockSkew).After(now) {
		return nil, fmt.Errorf("%w: token not yet valid", ErrInvalidToken)
	}

	// Check issuer
	if claims.Issuer != privyIssuer && claims.Issuer != "https://"+privyIssuer {
		return nil, ErrInvalidIssuer
	}

	// Check audience
	expectedAudience := opts.Audience
	if expectedAudience == "" {
		expectedAudience = s.client.appID
	}
	if claims.Audience != expectedAudience {
		return nil, ErrInvalidAudience
	}

	return &claims, nil
}

// GetJWKS fetches the JWKS from Privy's endpoint.
func (s *AuthService) GetJWKS(ctx context.Context) (*JWKS, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", jwksEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("privy: failed to create JWKS request: %w", err)
	}

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("privy: failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("privy: JWKS request failed with status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("privy: failed to decode JWKS: %w", err)
	}

	return &jwks, nil
}

// RefreshJWKS forces a refresh of the cached JWKS keys.
func (s *AuthService) RefreshJWKS(ctx context.Context) error {
	jwks, err := s.GetJWKS(ctx)
	if err != nil {
		return err
	}

	s.jwksCache.mu.Lock()
	defer s.jwksCache.mu.Unlock()

	s.jwksCache.keys = make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}
		pubKey, err := jwkToRSAPublicKey(&key)
		if err != nil {
			continue
		}
		s.jwksCache.keys[key.Kid] = pubKey
	}
	s.jwksCache.expiresAt = time.Now().Add(s.jwksCache.ttl)

	return nil
}

// verifySignature verifies the JWT signature using JWKS.
func (s *AuthService) verifySignature(ctx context.Context, message, signature, kid string) error {
	// Get the public key
	pubKey, err := s.getPublicKey(ctx, kid)
	if err != nil {
		return err
	}

	// Decode signature
	sigBytes, err := base64URLDecode(signature)
	if err != nil {
		return fmt.Errorf("%w: invalid signature encoding", ErrInvalidSignature)
	}

	// Hash the message with SHA256
	hash := sha256.Sum256([]byte(message))

	// Verify using RSA PKCS1v15 with SHA256
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes); err != nil {
		return ErrInvalidSignature
	}

	return nil
}

// getPublicKey retrieves a public key from cache or fetches from JWKS.
func (s *AuthService) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Check cache
	s.jwksCache.mu.RLock()
	if time.Now().Before(s.jwksCache.expiresAt) {
		if key, ok := s.jwksCache.keys[kid]; ok {
			s.jwksCache.mu.RUnlock()
			return key, nil
		}
	}
	s.jwksCache.mu.RUnlock()

	// Refresh cache
	if err := s.RefreshJWKS(ctx); err != nil {
		return nil, err
	}

	// Try again
	s.jwksCache.mu.RLock()
	defer s.jwksCache.mu.RUnlock()

	if key, ok := s.jwksCache.keys[kid]; ok {
		return key, nil
	}

	return nil, ErrKeyNotFound
}

// jwkToRSAPublicKey converts a JWK to an RSA public key.
func jwkToRSAPublicKey(jwk *JWK) (*rsa.PublicKey, error) {
	// Decode modulus
	nBytes, err := base64URLDecode(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode exponent
	eBytes, err := base64URLDecode(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert exponent bytes to int
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// base64URLDecode decodes a base64url encoded string.
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

// UserID returns the Privy user ID from the token claims.
func (c *TokenClaims) UserID() string {
	return c.Subject
}

// IsExpired checks if the token is expired.
func (c *TokenClaims) IsExpired() bool {
	if c.ExpiresAt == 0 {
		return false
	}
	return time.Unix(c.ExpiresAt, 0).Before(time.Now())
}

// ExpiresIn returns the duration until the token expires.
func (c *TokenClaims) ExpiresIn() time.Duration {
	if c.ExpiresAt == 0 {
		return 0
	}
	return time.Until(time.Unix(c.ExpiresAt, 0))
}

// GetClaim returns a custom claim by name.
func (c *TokenClaims) GetClaim(name string) (any, bool) {
	if c.Raw == nil {
		return nil, false
	}
	v, ok := c.Raw[name]
	return v, ok
}
