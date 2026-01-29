package privy

import (
	"context"
	"testing"
)

// ============================================
// Solana Signing E2E Tests
// ============================================

func TestE2E_Solana_SignMessage(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a Solana wallet
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	// Sign a message
	resp, err := client.Wallets().Solana().SignMessage(ctx, wallet.ID, "Hello, Solana!", "utf-8", "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	if resp.Method != "signMessage" {
		t.Errorf("Expected method signMessage, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Solana_SignMessageBase64(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	// Sign a base64-encoded message
	resp, err := client.Wallets().Solana().SignMessage(ctx, wallet.ID, "SGVsbG8sIFNvbGFuYSE=", "base64", "")
	if err != nil {
		t.Fatalf("Failed to sign base64 message: %v", err)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Solana_SignTransaction(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	// Base64 encoded transaction (mock)
	transaction := "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAEDAENCQUdHRVJFRFNPTEFOQVRSQU5TQUNUSQlPTg=="

	resp, err := client.Wallets().Solana().SignTransaction(ctx, wallet.ID, transaction, "")
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	if resp.Method != "signTransaction" {
		t.Errorf("Expected method signTransaction, got %s", resp.Method)
	}

	if resp.Data.SignedTransaction == "" {
		t.Error("Expected signed transaction to be returned")
	}
}

func TestE2E_Solana_SignAndSendTransaction(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	transaction := "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAEDAENCQUdHRVJFRFNPTEFOQVRSQU5TQUNUSQlPTg=="

	resp, err := client.Wallets().Solana().SignAndSendTransaction(ctx, wallet.ID, transaction, "")
	if err != nil {
		t.Fatalf("Failed to sign and send transaction: %v", err)
	}

	if resp.Method != "signAndSendTransaction" {
		t.Errorf("Expected method signAndSendTransaction, got %s", resp.Method)
	}

	if resp.Data.Hash == "" {
		t.Error("Expected transaction hash to be returned")
	}
}

func TestE2E_Solana_SignAndSendTransactionOnDevnet(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	transaction := "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAEDAENCQUdHRVJFRFNPTEFOQVRSQU5TQUNUSQlPTg=="

	resp, err := client.Wallets().Solana().SignAndSendTransactionOnDevnet(ctx, wallet.ID, transaction, "")
	if err != nil {
		t.Fatalf("Failed to sign and send transaction on devnet: %v", err)
	}

	if resp.Data.Hash == "" {
		t.Error("Expected transaction hash to be returned")
	}
}

func TestE2E_Solana_SignAndSendTransactionWithCustomCAIP2(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	transaction := "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAEDAENCQUdHRVJFRFNPTEFOQVRSQU5TQUNUSQlPTg=="
	customCAIP2 := "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp" // Mainnet

	resp, err := client.Wallets().Solana().SignAndSendTransactionWithCAIP2(ctx, wallet.ID, transaction, customCAIP2, "")
	if err != nil {
		t.Fatalf("Failed to sign and send with custom CAIP2: %v", err)
	}

	if resp.Data.Hash == "" {
		t.Error("Expected transaction hash to be returned")
	}
}

func TestE2E_Solana_SignWithNonExistentWallet(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Wallets().Solana().SignMessage(ctx, "nonexistent-wallet", "Hello", "utf-8", "")
	if err == nil {
		t.Error("Expected error for non-existent wallet")
	}
}

func TestE2E_Solana_MultipleSignOperations(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Sign multiple messages
	for i := 0; i < 3; i++ {
		resp, err := client.Wallets().Solana().SignMessage(ctx, wallet.ID, "Solana Message "+string(rune('A'+i)), "utf-8", "")
		if err != nil {
			t.Fatalf("Failed to sign message %d: %v", i, err)
		}
		if resp.Data.Signature == "" {
			t.Errorf("Expected signature for message %d", i)
		}
	}
}
