package sui

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
	if h.rpcURL != "https://fullnode.mainnet.sui.io:443" {
		t.Errorf("Expected default RPC URL, got %s", h.rpcURL)
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client, WithRPCURL("https://fullnode.testnet.sui.io:443"))

	if h.rpcURL != "https://fullnode.testnet.sui.io:443" {
		t.Errorf("Expected custom RPC URL, got %s", h.rpcURL)
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
	// Mock Sui RPC
	suiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "suix_getCoins":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]any{
					"data": []map[string]any{
						{"coinObjectId": "0xabc123"},
					},
				},
			})
		case "unsafe_transferSui":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]any{
					"txBytes": "AQAAAA==",
				},
			})
		case "sui_executeTransactionBlock":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]any{
					"digest": "HKE7K8Bk1wYHfGjm2T5X",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer suiServer.Close()

	// Mock Privy API
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/wallets/wallet-123":
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "wallet-123",
				"address":    "0xsender123",
				"chain_type": "sui",
				"public_key": "0xabcdef0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			})
		case r.Method == "POST" && r.URL.Path == "/v1/wallets/wallet-123/raw_sign":
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

	h := NewHelper(client, WithRPCURL(suiServer.URL))

	digest, err := h.Transfer(context.Background(), "wallet-123", "0xrecipient456", "1000000000")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}
	if digest != "HKE7K8Bk1wYHfGjm2T5X" {
		t.Errorf("Expected digest HKE7K8Bk1wYHfGjm2T5X, got %s", digest)
	}
}

func TestTransferObject_NotImplemented(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client)

	_, err := h.TransferObject(context.Background(), "wallet-id", "0xobject...", "0xdest...")
	if err == nil {
		t.Error("Expected error for unimplemented TransferObject")
	}
}
