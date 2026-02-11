package solana

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
	if h.caip2 != MainnetCAIP2 {
		t.Errorf("Expected mainnet CAIP-2, got %s", h.caip2)
	}
	if h.client != client {
		t.Error("Expected client to be set")
	}
}

func TestNewHelper_WithDevnet(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client, WithDevnet())

	if h.caip2 != DevnetCAIP2 {
		t.Errorf("Expected devnet CAIP-2, got %s", h.caip2)
	}
}

func TestNewHelper_WithCAIP2(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	customCAIP2 := "solana:custom123"
	h := NewHelper(client, WithCAIP2(customCAIP2))

	if h.caip2 != customCAIP2 {
		t.Errorf("Expected custom CAIP-2 %s, got %s", customCAIP2, h.caip2)
	}
}

func TestNewHelper_WithRPCURL(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client, WithRPCURL("https://custom-rpc.example.com"))

	if h.rpcURL != "https://custom-rpc.example.com" {
		t.Errorf("Expected custom RPC URL, got %s", h.rpcURL)
	}
}
