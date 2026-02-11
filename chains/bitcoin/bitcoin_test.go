package bitcoin

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
	if h.network != "mainnet" {
		t.Errorf("Expected default network mainnet, got %s", h.network)
	}
	if h.feeRate != 10 {
		t.Errorf("Expected default fee rate 10, got %d", h.feeRate)
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client,
		WithExplorerURL("https://blockstream.info/testnet/api"),
		WithNetwork("testnet"),
		WithFeeRate(20),
	)

	if h.explorerURL != "https://blockstream.info/testnet/api" {
		t.Errorf("Expected custom explorer URL, got %s", h.explorerURL)
	}
	if h.network != "testnet" {
		t.Errorf("Expected testnet network, got %s", h.network)
	}
	if h.feeRate != 20 {
		t.Errorf("Expected fee rate 20, got %d", h.feeRate)
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

func TestDerEncodeSignature(t *testing.T) {
	// Test with known R and S values
	r := make([]byte, 32)
	s := make([]byte, 32)
	r[0] = 0x01
	s[0] = 0x02

	der := derEncodeSignature(r, s)

	// Should start with 0x30 (SEQUENCE)
	if der[0] != 0x30 {
		t.Errorf("Expected DER sequence tag 0x30, got 0x%02x", der[0])
	}
	// Should contain two INTEGER elements (0x02)
	if der[2] != 0x02 {
		t.Errorf("Expected first INTEGER tag 0x02, got 0x%02x", der[2])
	}
}

func TestCanonicalizeInt(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{"strip leading zeros", []byte{0, 0, 1}, []byte{1}},
		{"high bit set", []byte{0x80}, []byte{0, 0x80}},
		{"already canonical", []byte{0x42}, []byte{0x42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalizeInt(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
		})
	}
}

func TestSelectUTXOs(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client, WithFeeRate(10))

	utxos := []UTXO{
		{TxID: "aaa", Vout: 0, Value: 50000},
		{TxID: "bbb", Vout: 1, Value: 100000},
	}

	selected, total, fee, err := h.selectUTXOs(utxos, 30000)
	if err != nil {
		t.Fatalf("selectUTXOs failed: %v", err)
	}
	if len(selected) == 0 {
		t.Error("Expected at least one selected UTXO")
	}
	if total < 30000+fee {
		t.Errorf("Total input %d should cover amount + fee %d", total, 30000+fee)
	}
}

func TestSelectUTXOs_InsufficientFunds(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client)

	utxos := []UTXO{
		{TxID: "aaa", Vout: 0, Value: 100},
	}

	_, _, _, err := h.selectUTXOs(utxos, 100000)
	if err == nil {
		t.Error("Expected error for insufficient funds")
	}
}

func TestTransfer_WithMockServer(t *testing.T) {
	// The compressed public key 0x0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798
	// corresponds to P2WPKH address bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4.
	walletAddress := "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"

	// Mock Block Explorer API
	explorerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/address/"+walletAddress+"/utxo":
			json.NewEncoder(w).Encode([]UTXO{
				{TxID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Vout: 0, Value: 100000},
			})
		case r.Method == "POST" && r.URL.Path == "/tx":
			w.Write([]byte("txid123456"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer explorerServer.Close()

	// Mock Privy API
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/wallets/wallet-123":
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "wallet-123",
				"address":    walletAddress,
				"chain_type": "bitcoin",
				// Compressed public key (33 bytes, starts with 0x02)
				"public_key": "0x0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
			})
		case r.Method == "POST" && r.URL.Path == "/v1/wallets/wallet-123/raw_sign":
			// 64-byte signature (R + S)
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
		WithExplorerURL(explorerServer.URL),
	)

	txID, err := h.Transfer(context.Background(), "wallet-123", "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", "50000")
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}
	if txID != "txid123456" {
		t.Errorf("Expected txID txid123456, got %s", txID)
	}
}
