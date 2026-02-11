package privy

import (
	"context"
	"testing"
)

// ============================================
// Starknet RawSign E2E Tests
// ============================================

func TestE2E_Starknet_RawSign(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeStarknet,
	})
	if err != nil {
		t.Fatalf("Failed to create Starknet wallet: %v", err)
	}

	hash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	resp, err := client.Wallets().Starknet().RawSign(ctx, wallet.ID, hash)
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	if resp.Method != "raw_sign" {
		t.Errorf("Expected method raw_sign, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Starknet_RawSignBytes(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeStarknet,
	})
	if err != nil {
		t.Fatalf("Failed to create Starknet wallet: %v", err)
	}

	resp, err := client.Wallets().Starknet().RawSignBytes(ctx, wallet.ID, "48656c6c6f", "hex", "sha256")
	if err != nil {
		t.Fatalf("Failed to raw sign bytes: %v", err)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Starknet_RawSignWithNonExistentWallet(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Wallets().Starknet().RawSign(ctx, "non-existent-wallet-id", "0xabcdef")
	if err == nil {
		t.Error("Expected error for non-existent wallet")
	}
}

func TestE2E_Starknet_MultipleRawSignOperations(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeStarknet,
	})
	if err != nil {
		t.Fatalf("Failed to create Starknet wallet: %v", err)
	}

	resp1, err := client.Wallets().Starknet().RawSign(ctx, wallet.ID, "0xhash1")
	if err != nil {
		t.Fatalf("Failed first raw sign: %v", err)
	}
	if resp1.Data.Signature == "" {
		t.Error("Expected signature from first raw sign")
	}

	resp2, err := client.Wallets().Starknet().RawSignBytes(ctx, wallet.ID, "Hello Starknet", "utf-8", "sha256")
	if err != nil {
		t.Fatalf("Failed raw sign bytes: %v", err)
	}
	if resp2.Data.Signature == "" {
		t.Error("Expected signature from raw sign bytes")
	}
}
