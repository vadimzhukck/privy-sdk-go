package cosmos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

func TestNewHelper(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client)
	if h == nil {
		t.Fatal("Expected non-nil helper")
	}
	if h.rpcURL != "https://rest.cosmos.directory/cosmoshub" {
		t.Errorf("Expected default RPC URL, got %s", h.rpcURL)
	}
	if h.chainID != "cosmoshub-4" {
		t.Errorf("Expected default chain ID, got %s", h.chainID)
	}
	if h.denom != "uatom" {
		t.Errorf("Expected default denom uatom, got %s", h.denom)
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client,
		WithRPCURL("https://custom-rpc.example.com"),
		WithChainID("osmosis-1"),
		WithDenom("uosmo"),
		WithGasLimit(300000),
		WithFeeAmount("10000"),
	)

	if h.rpcURL != "https://custom-rpc.example.com" {
		t.Errorf("Expected custom RPC URL, got %s", h.rpcURL)
	}
	if h.chainID != "osmosis-1" {
		t.Errorf("Expected custom chain ID, got %s", h.chainID)
	}
	if h.denom != "uosmo" {
		t.Errorf("Expected custom denom, got %s", h.denom)
	}
	if h.gasLimit != 300000 {
		t.Errorf("Expected gas limit 300000, got %d", h.gasLimit)
	}
	if h.feeAmount != "10000" {
		t.Errorf("Expected fee amount 10000, got %s", h.feeAmount)
	}
}

func TestNewHelper_WithHTTPClient(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	customHTTP := &http.Client{}

	h := NewHelper(client, WithHTTPClient(customHTTP))

	if h.httpClient != customHTTP {
		t.Error("Expected custom HTTP client to be set")
	}
}

func TestTransfer_WithMockServer(t *testing.T) {
	// Mock Cosmos REST API
	cosmosServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/cosmos/auth/v1beta1/accounts/cosmos1sender123":
			json.NewEncoder(w).Encode(map[string]any{
				"account": map[string]any{
					"account_number": "12345",
					"sequence":       "7",
				},
			})
		case r.Method == "POST" && r.URL.Path == "/cosmos/tx/v1beta1/txs":
			json.NewEncoder(w).Encode(map[string]any{
				"tx_response": map[string]any{
					"txhash":  "ABCDEF1234567890",
					"code":    0,
					"raw_log": "",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cosmosServer.Close()

	// Mock Privy API
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/wallets/wallet-123":
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "wallet-123",
				"address":    "cosmos1sender123",
				"chain_type": "cosmos",
				"public_key": "0x0200010203040506070809101112131415161718192021222324252627282930313233",
			})
		case r.Method == "POST" && r.URL.Path == "/v1/wallets/wallet-123/raw_sign":
			// 64-byte secp256k1 signature (R+S, no recovery)
			json.NewEncoder(w).Encode(map[string]any{
				"method": "raw_sign",
				"data": map[string]any{
					"signature": "0xaabbccddeeff00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899",
					"encoding":  "hex",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer privyServer.Close()

	client := privy.NewClient("test-app-id", "test-app-secret",
		privy.WithBaseURL(privyServer.URL+"/v1"))

	h := NewHelper(client,
		WithRPCURL(cosmosServer.URL),
	)

	txHash, err := h.Transfer(context.Background(), "wallet-123", "cosmos1recipient456", "1000000")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}
	if txHash != "ABCDEF1234567890" {
		t.Errorf("Expected txHash ABCDEF1234567890, got %s", txHash)
	}
}

func TestDelegate_NotImplemented(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client)

	_, err := h.Delegate(context.Background(), "wallet-id", "cosmosvaloper1...", "1000000")
	if err == nil {
		t.Error("Expected error for unimplemented Delegate")
	}
}

func TestUndelegate_NotImplemented(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client)

	_, err := h.Undelegate(context.Background(), "wallet-id", "cosmosvaloper1...", "1000000")
	if err == nil {
		t.Error("Expected error for unimplemented Undelegate")
	}
}
