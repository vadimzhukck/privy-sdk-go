package starknet

import (
	"context"
	"encoding/json"
	"math/big"
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
	if h.rpcURL != "https://starknet-mainnet.public.blastapi.io" {
		t.Errorf("Expected default RPC URL, got %s", h.rpcURL)
	}
	if h.maxFee.Cmp(big.NewInt(1e16)) != 0 {
		t.Errorf("Expected default max fee 1e16, got %s", h.maxFee.String())
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client,
		WithRPCURL("https://custom-rpc.example.com"),
		WithChainID("SN_GOERLI"),
		WithMaxFee(big.NewInt(1e15)),
	)

	if h.rpcURL != "https://custom-rpc.example.com" {
		t.Errorf("Expected custom RPC URL, got %s", h.rpcURL)
	}
	if h.maxFee.Cmp(big.NewInt(1e15)) != 0 {
		t.Errorf("Expected custom max fee, got %s", h.maxFee.String())
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

func TestStringToFelt(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SN_MAIN", "23448594291968334"},
		{"invoke", "115923154332517"},
	}

	for _, tt := range tests {
		result := stringToFelt(tt.input)
		if result.String() != tt.expected {
			t.Errorf("stringToFelt(%q) = %s, want %s", tt.input, result.String(), tt.expected)
		}
	}
}

func TestHexToBigInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0x0", 0},
		{"0x1", 1},
		{"0xff", 255},
		{"0x100", 256},
	}

	for _, tt := range tests {
		result := hexToBigInt(tt.input)
		if result.Int64() != tt.expected {
			t.Errorf("hexToBigInt(%q) = %d, want %d", tt.input, result.Int64(), tt.expected)
		}
	}
}

func TestBuildETHTransferCalldata(t *testing.T) {
	recipient := "0x1234567890abcdef"
	amount := big.NewInt(1000000)

	calldata := buildETHTransferCalldata(recipient, amount)

	// Should have 9 elements
	if len(calldata) != 9 {
		t.Fatalf("Expected 9 calldata elements, got %d", len(calldata))
	}

	// call_array_len should be 1
	if calldata[0].Int64() != 1 {
		t.Errorf("Expected call_array_len=1, got %d", calldata[0].Int64())
	}

	// data_len and calldata_len should be 3
	if calldata[4].Int64() != 3 {
		t.Errorf("Expected data_len=3, got %d", calldata[4].Int64())
	}
	if calldata[5].Int64() != 3 {
		t.Errorf("Expected calldata_len=3, got %d", calldata[5].Int64())
	}

	// amount.low should be 1000000 (fits in 128 bits)
	if calldata[7].Int64() != 1000000 {
		t.Errorf("Expected amount.low=1000000, got %d", calldata[7].Int64())
	}

	// amount.high should be 0
	if calldata[8].Int64() != 0 {
		t.Errorf("Expected amount.high=0, got %d", calldata[8].Int64())
	}
}

func TestTransfer_WithMockServer(t *testing.T) {
	// Mock StarkNet JSON-RPC server
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "starknet_getNonce":
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`"0x5"`),
			})
		case "starknet_addInvokeTransaction":
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{"transaction_hash":"0xabc123"}`),
			})
		default:
			http.Error(w, "unknown method", http.StatusBadRequest)
		}
	}))
	defer rpcServer.Close()

	// Mock Privy API
	privyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/wallets/wallet-123":
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "wallet-123",
				"address":    "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7",
				"chain_type": "starknet",
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

	h := NewHelper(client,
		WithRPCURL(rpcServer.URL),
		WithMaxFee(big.NewInt(1e15)),
	)

	txHash, err := h.Transfer(context.Background(), "wallet-123",
		"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"1000000000000000000") // 1 ETH in wei
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}
	if txHash != "0xabc123" {
		t.Errorf("Expected tx hash 0xabc123, got %s", txHash)
	}
}

func TestTransferERC20_NotImplemented(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client)

	_, err := h.TransferERC20(context.Background(), "wallet-123", "0xtoken", "0xdest", "1000")
	if err == nil {
		t.Error("Expected error for unimplemented TransferERC20")
	}
}
