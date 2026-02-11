package ethereum

import (
	"testing"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

func TestNewHelper(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client)
	if h == nil {
		t.Fatal("Expected non-nil helper")
	}
	if h.chainID != 1 {
		t.Errorf("Expected default chain ID 1, got %d", h.chainID)
	}
	if h.client != client {
		t.Error("Expected client to be set")
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client, WithChainID(137))

	if h.chainID != 137 {
		t.Errorf("Expected chain ID 137, got %d", h.chainID)
	}
}

func TestNewHelper_CommonChains(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	tests := []struct {
		name    string
		chainID int64
	}{
		{"Ethereum Mainnet", 1},
		{"Polygon", 137},
		{"Arbitrum", 42161},
		{"Optimism", 10},
		{"Base", 8453},
		{"Avalanche", 43114},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHelper(client, WithChainID(tt.chainID))
			if h.chainID != tt.chainID {
				t.Errorf("Expected chain ID %d, got %d", tt.chainID, h.chainID)
			}
		})
	}
}
