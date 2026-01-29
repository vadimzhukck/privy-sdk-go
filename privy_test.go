package privy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("app-id", "app-secret")

	if client.appID != "app-id" {
		t.Errorf("expected appID to be 'app-id', got '%s'", client.appID)
	}

	if client.appSecret != "app-secret" {
		t.Errorf("expected appSecret to be 'app-secret', got '%s'", client.appSecret)
	}

	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL to be '%s', got '%s'", DefaultBaseURL, client.baseURL)
	}

	if client.authURL != DefaultAuthBaseURL {
		t.Errorf("expected authURL to be '%s', got '%s'", DefaultAuthBaseURL, client.authURL)
	}
}

func TestNewClientWithOptions(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 60 * time.Second}

	client := NewClient(
		"app-id",
		"app-secret",
		WithBaseURL("https://custom.api.privy.io/v1"),
		WithAuthURL("https://custom.auth.privy.io/api/v1"),
		WithHTTPClient(customHTTPClient),
	)

	if client.baseURL != "https://custom.api.privy.io/v1" {
		t.Errorf("expected custom baseURL, got '%s'", client.baseURL)
	}

	if client.authURL != "https://custom.auth.privy.io/api/v1" {
		t.Errorf("expected custom authURL, got '%s'", client.authURL)
	}

	if client.httpClient != customHTTPClient {
		t.Error("expected custom HTTP client")
	}
}

func TestBasicAuth(t *testing.T) {
	client := NewClient("my-app-id", "my-app-secret")
	auth := client.basicAuth()

	// Basic auth should be "Basic base64(my-app-id:my-app-secret)"
	expected := "Basic bXktYXBwLWlkOm15LWFwcC1zZWNyZXQ="
	if auth != expected {
		t.Errorf("expected auth to be '%s', got '%s'", expected, auth)
	}
}

func TestUsersCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/users" {
			t.Errorf("expected path '/v1/users', got '%s'", r.URL.Path)
		}

		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}

		if r.Header.Get("privy-app-id") != "test-app-id" {
			t.Errorf("expected privy-app-id header to be 'test-app-id', got '%s'", r.Header.Get("privy-app-id"))
		}

		response := User{
			ID:        "did:privy:test123",
			CreatedAt: time.Now().UnixMilli(),
			LinkedAccounts: []LinkedAccount{
				{
					Type:    LinkedAccountTypeEmail,
					Address: "[email protected]",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-app-id", "test-secret", WithBaseURL(server.URL+"/v1"))

	user, err := client.Users().Create(context.Background(), &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{
				Type:    LinkedAccountTypeEmail,
				Address: "[email protected]",
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != "did:privy:test123" {
		t.Errorf("expected user ID 'did:privy:test123', got '%s'", user.ID)
	}

	if len(user.LinkedAccounts) != 1 {
		t.Fatalf("expected 1 linked account, got %d", len(user.LinkedAccounts))
	}

	if user.LinkedAccounts[0].Type != LinkedAccountTypeEmail {
		t.Errorf("expected linked account type 'email', got '%s'", user.LinkedAccounts[0].Type)
	}
}

func TestWalletsCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/wallets" {
			t.Errorf("expected path '/v1/wallets', got '%s'", r.URL.Path)
		}

		var req CreateWalletRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.ChainType != ChainTypeEthereum {
			t.Errorf("expected chain type 'ethereum', got '%s'", req.ChainType)
		}

		response := Wallet{
			ID:        "wallet-test-123",
			Address:   "0x1234567890abcdef1234567890abcdef12345678",
			ChainType: ChainTypeEthereum,
			CreatedAt: time.Now().UnixMilli(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-app-id", "test-secret", WithBaseURL(server.URL+"/v1"))

	wallet, err := client.Wallets().Create(context.Background(), &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wallet.ID != "wallet-test-123" {
		t.Errorf("expected wallet ID 'wallet-test-123', got '%s'", wallet.ID)
	}

	if wallet.ChainType != ChainTypeEthereum {
		t.Errorf("expected chain type 'ethereum', got '%s'", wallet.ChainType)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "User not found",
			"code":    "user_not_found",
		})
	}))
	defer server.Close()

	client := NewClient("test-app-id", "test-secret", WithAuthURL(server.URL+"/api/v1"))

	_, err := client.Users().Get(context.Background(), "invalid-id")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status code 404, got %d", apiErr.StatusCode)
	}

	if apiErr.Message != "User not found" {
		t.Errorf("expected message 'User not found', got '%s'", apiErr.Message)
	}
}

func TestUserCreatedAtTime(t *testing.T) {
	now := time.Now()
	user := &User{
		ID:        "did:privy:test",
		CreatedAt: now.UnixMilli(),
	}

	createdAt := user.CreatedAtTime()

	// Allow some tolerance for millisecond precision
	if createdAt.Sub(now) > time.Millisecond || now.Sub(createdAt) > time.Millisecond {
		t.Errorf("expected CreatedAtTime to return %v, got %v", now, createdAt)
	}
}

func TestWalletCreatedAtTime(t *testing.T) {
	now := time.Now()
	wallet := &Wallet{
		ID:        "wallet-test",
		CreatedAt: now.UnixMilli(),
	}

	createdAt := wallet.CreatedAtTime()

	if createdAt.Sub(now) > time.Millisecond || now.Sub(createdAt) > time.Millisecond {
		t.Errorf("expected CreatedAtTime to return %v, got %v", now, createdAt)
	}
}

func TestChainTypes(t *testing.T) {
	chainTypes := []ChainType{
		ChainTypeEthereum,
		ChainTypeSolana,
		ChainTypeStellar,
		ChainTypeCosmos,
		ChainTypeSui,
		ChainTypeTron,
		ChainTypeBitcoinSegwit,
		ChainTypeNear,
		ChainTypeTon,
		ChainTypeStarknet,
		ChainTypeAptos,
	}

	expected := []string{
		"ethereum",
		"solana",
		"stellar",
		"cosmos",
		"sui",
		"tron",
		"bitcoin-segwit",
		"near",
		"ton",
		"starknet",
		"aptos",
	}

	for i, ct := range chainTypes {
		if string(ct) != expected[i] {
			t.Errorf("expected chain type '%s', got '%s'", expected[i], ct)
		}
	}
}

func TestLinkedAccountTypes(t *testing.T) {
	types := []LinkedAccountType{
		LinkedAccountTypeEmail,
		LinkedAccountTypePhone,
		LinkedAccountTypeWallet,
		LinkedAccountTypeGoogle,
		LinkedAccountTypeTwitter,
		LinkedAccountTypeDiscord,
		LinkedAccountTypeGithub,
		LinkedAccountTypeFarcaster,
		LinkedAccountTypeTelegram,
	}

	for _, lat := range types {
		if lat == "" {
			t.Error("linked account type should not be empty")
		}
	}
}

func TestEthereumSignMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var req RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Method != "personal_sign" {
			t.Errorf("expected method 'personal_sign', got '%s'", req.Method)
		}

		if req.ChainType != "ethereum" {
			t.Errorf("expected chain type 'ethereum', got '%s'", req.ChainType)
		}

		response := SignatureResponse{
			Method: "personal_sign",
		}
		response.Data.Signature = "0xsignature..."
		response.Data.Encoding = "hex"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-app-id", "test-secret", WithBaseURL(server.URL+"/v1"))

	resp, err := client.Wallets().Ethereum().SignMessage(
		context.Background(),
		"wallet-123",
		"Hello, World!",
		"utf-8",
		"",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Method != "personal_sign" {
		t.Errorf("expected method 'personal_sign', got '%s'", resp.Method)
	}

	if resp.Data.Signature != "0xsignature..." {
		t.Errorf("expected signature '0xsignature...', got '%s'", resp.Data.Signature)
	}
}
