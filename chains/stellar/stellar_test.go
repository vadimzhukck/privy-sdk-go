package stellar

import (
	"testing"

	"github.com/stellar/go/network"
	privy "github.com/vadimzhukck/privy-sdk-go"
)

func TestNewHelper(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client)
	if h == nil {
		t.Fatal("Expected non-nil helper")
	}
	if h.horizonURL != "https://horizon.stellar.org" {
		t.Errorf("Expected default horizon URL, got %s", h.horizonURL)
	}
	if h.networkPass != network.PublicNetworkPassphrase {
		t.Errorf("Expected public network passphrase, got %s", h.networkPass)
	}
	if h.client != client {
		t.Error("Expected client to be set")
	}
}

func TestNewHelper_WithOptions(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")

	h := NewHelper(client,
		WithHorizonURL("https://horizon-testnet.stellar.org"),
		WithNetworkPassphrase(network.TestNetworkPassphrase),
	)

	if h.horizonURL != "https://horizon-testnet.stellar.org" {
		t.Errorf("Expected custom horizon URL, got %s", h.horizonURL)
	}
	if h.networkPass != network.TestNetworkPassphrase {
		t.Errorf("Expected test network passphrase, got %s", h.networkPass)
	}
}

func TestPaymentWithAsset_NotImplemented(t *testing.T) {
	client := privy.NewClient("test-app-id", "test-app-secret")
	h := NewHelper(client)

	_, err := h.PaymentWithAsset(nil, "wallet-id", "GDEST...", "100.0", "USDC", "GA5ZS...")
	if err == nil {
		t.Error("Expected error for unimplemented PaymentWithAsset")
	}
}
