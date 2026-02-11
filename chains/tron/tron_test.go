package tron

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
	if h.rpcURL != "https://api.trongrid.io" {
		t.Errorf("Expected default RPC URL, got %s", h.rpcURL)
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client, WithRPCURL("https://api.shasta.trongrid.io"))

	if h.rpcURL != "https://api.shasta.trongrid.io" {
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
	// Mock Tron API
	tronServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wallet/createtransaction":
			json.NewEncoder(w).Encode(map[string]any{
				"visible":      true,
				"txid":         "abc123def456",
				"raw_data":     map[string]any{"contract": []any{}},
				"raw_data_hex": "0a0200",
			})
		case "/wallet/broadcasttransaction":
			json.NewEncoder(w).Encode(map[string]any{
				"result": true,
				"txid":   "abc123def456",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer tronServer.Close()

	// Mock Privy API
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/wallets/wallet-123":
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "wallet-123",
				"address":    "TJmmqjb1DK9TTZbQXzRQ2AuA94z4jCcPMb",
				"chain_type": "tron",
			})
		case r.Method == "POST" && r.URL.Path == "/v1/wallets/wallet-123/raw_sign":
			json.NewEncoder(w).Encode(map[string]any{
				"method": "raw_sign",
				"data": map[string]any{
					"signature": "0xabcdef1234567890",
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
		WithRPCURL(tronServer.URL),
	)

	txID, err := h.Transfer(context.Background(), "wallet-123", "TF17BgPaZYbz8oxbjhriubPDsA7ArKoLX3", "1000000")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}
	if txID != "abc123def456" {
		t.Errorf("Expected txID abc123def456, got %s", txID)
	}
}

func TestTransfer_InvalidAmount(t *testing.T) {
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "wallet-123",
			"address":    "TJmmqjb1DK9TTZbQXzRQ2AuA94z4jCcPMb",
			"chain_type": "tron",
		})
	}))
	defer privyServer.Close()

	client := privy.NewClient("test-app-id", "test-app-secret",
		privy.WithBaseURL(privyServer.URL+"/v1"))
	h := NewHelper(client)

	_, err := h.Transfer(context.Background(), "wallet-123", "TDest...", "not-a-number")
	if err == nil {
		t.Error("Expected error for invalid amount")
	}
}

func TestTransferTRC20_NotImplemented(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client)

	_, err := h.TransferTRC20(context.Background(), "wallet-id", "TContract...", "TDest...", "1000000")
	if err == nil {
		t.Error("Expected error for unimplemented TransferTRC20")
	}
}
