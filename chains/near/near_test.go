package near

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
	if h.rpcURL != "https://rpc.mainnet.near.org" {
		t.Errorf("Expected default RPC URL, got %s", h.rpcURL)
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client, WithRPCURL("https://rpc.testnet.near.org"))

	if h.rpcURL != "https://rpc.testnet.near.org" {
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

func TestBase58EncodeDecode(t *testing.T) {
	tests := []struct {
		decoded []byte
		encoded string
	}{
		{[]byte{0, 0, 1}, "112"},
		{[]byte{0x01, 0x02, 0x03}, "Ldp"},
	}

	for _, tt := range tests {
		enc := base58Encode(tt.decoded)
		if enc != tt.encoded {
			t.Errorf("base58Encode(%v) = %q, want %q", tt.decoded, enc, tt.encoded)
		}
		dec, err := base58Decode(tt.encoded)
		if err != nil {
			t.Fatalf("base58Decode(%q): %v", tt.encoded, err)
		}
		if len(dec) != len(tt.decoded) {
			t.Errorf("base58Decode(%q) length = %d, want %d", tt.encoded, len(dec), len(tt.decoded))
		}
	}
}

func TestTransfer_WithMockServer(t *testing.T) {
	// Mock NEAR RPC
	nearServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "query":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      "privy",
				"result": map[string]any{
					"nonce":      42,
					"block_hash": "11111111111111111111111111111111",
				},
			})
		case "block":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      "privy",
				"result": map[string]any{
					"header": map[string]any{
						"hash": "11111111111111111111111111111111",
					},
				},
			})
		case "broadcast_tx_commit":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      "privy",
				"result": map[string]any{
					"transaction": map[string]any{
						"hash": "9FbE4bFHQZ2D1e5GwHZ3bE4gQzP7xXMwHY6DEjq1Lhi4",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer nearServer.Close()

	// Mock Privy API
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/wallets/wallet-123":
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "wallet-123",
				"address":    "test.near",
				"chain_type": "near",
				"public_key": "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			})
		case r.Method == "POST" && r.URL.Path == "/v1/wallets/wallet-123/raw_sign":
			// Return a 64-byte Ed25519 signature
			json.NewEncoder(w).Encode(map[string]any{
				"method": "raw_sign",
				"data": map[string]any{
					"signature": "0x" + "aa" + "bb" + "cc" + "dd" + "ee" + "ff" + "00" + "11" + "22" + "33" + "44" + "55" + "66" + "77" + "88" + "99" + "aa" + "bb" + "cc" + "dd" + "ee" + "ff" + "00" + "11" + "22" + "33" + "44" + "55" + "66" + "77" + "88" + "99" + "aa" + "bb" + "cc" + "dd" + "ee" + "ff" + "00" + "11" + "22" + "33" + "44" + "55" + "66" + "77" + "88" + "99" + "aa" + "bb" + "cc" + "dd" + "ee" + "ff" + "00" + "11" + "22" + "33" + "44" + "55" + "66" + "77" + "88" + "99",
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

	h := NewHelper(client, WithRPCURL(nearServer.URL))

	txHash, err := h.Transfer(context.Background(), "wallet-123", "recipient.near", "1000000000000000000000000")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}
	if txHash == "" {
		t.Error("Expected non-empty transaction hash")
	}
}
