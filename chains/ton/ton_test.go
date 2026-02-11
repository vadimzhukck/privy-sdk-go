package ton

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
	if h.rpcURL != "https://toncenter.com/api/v2" {
		t.Errorf("Expected default RPC URL, got %s", h.rpcURL)
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client, WithRPCURL("https://testnet.toncenter.com/api/v2"))

	if h.rpcURL != "https://testnet.toncenter.com/api/v2" {
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

func TestBuildSigningMessage(t *testing.T) {
	msg := buildSigningMessage(698983191, 1, 1700000000, []byte{0x01, 0x02})
	// 4(walletID) + 4(validUntil) + 4(seqno) + 1(op) + 1(sendMode) + 2(internalMsg) = 16 bytes
	if len(msg) != 16 {
		t.Errorf("Expected signing message length 16, got %d", len(msg))
	}
}

func TestTransfer_WithMockServer(t *testing.T) {
	// Mock TON API
	tonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/getWalletInformation":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"result": map[string]any{
					"wallet":        true,
					"balance":       "1000000000",
					"seqno":         5,
					"account_state": "active",
				},
			})
		case "/sendBoc":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"result": map[string]any{
					"hash": "abc123",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer tonServer.Close()

	// Mock Privy API
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/wallets/wallet-123":
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "wallet-123",
				"address":    "EQDtFpEwcFAEcRe5mLVh2N6C0x-_hJEM7W61_JLnSF74p4q2",
				"chain_type": "ton",
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

	h := NewHelper(client, WithRPCURL(tonServer.URL))

	txHash, err := h.Transfer(context.Background(), "wallet-123", "EQDest...", "1000000000")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}
	if txHash == "" {
		t.Error("Expected non-empty transaction hash")
	}
}
