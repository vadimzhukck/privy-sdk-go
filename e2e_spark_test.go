package privy

import (
	"context"
	"testing"
)

func TestE2E_Spark_GetBalance(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().GetBalance(ctx, wallet.ID, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if resp.Method != "getBalance" {
		t.Errorf("Expected method getBalance, got %s", resp.Method)
	}

	if resp.Data.Balance == "" {
		t.Error("Expected balance to be returned")
	}
}

func TestE2E_Spark_Transfer(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().Transfer(ctx, wallet.ID, "sprt1mockaddress", 1000, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to transfer: %v", err)
	}

	if resp.Method != "transfer" {
		t.Errorf("Expected method transfer, got %s", resp.Method)
	}

	if resp.Data.ID == "" {
		t.Error("Expected transfer ID to be returned")
	}

	if resp.Data.TransferDirection != "OUTGOING" {
		t.Errorf("Expected transfer direction OUTGOING, got %s", resp.Data.TransferDirection)
	}
}

func TestE2E_Spark_TransferTokens(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().TransferTokens(ctx, wallet.ID, "btkn-mock-token", 100, "sprt1mockaddress", SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to transfer tokens: %v", err)
	}

	if resp.Method != "transferTokens" {
		t.Errorf("Expected method transferTokens, got %s", resp.Method)
	}
}

func TestE2E_Spark_GetStaticDepositAddress(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().GetStaticDepositAddress(ctx, wallet.ID, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to get deposit address: %v", err)
	}

	if resp.Method != "getStaticDepositAddress" {
		t.Errorf("Expected method getStaticDepositAddress, got %s", resp.Method)
	}

	if resp.Data.Address == "" {
		t.Error("Expected deposit address to be returned")
	}
}

func TestE2E_Spark_GetClaimStaticDepositQuote(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().GetClaimStaticDepositQuote(ctx, wallet.ID, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to get claim quote: %v", err)
	}

	if resp.Method != "getClaimStaticDepositQuote" {
		t.Errorf("Expected method getClaimStaticDepositQuote, got %s", resp.Method)
	}
}

func TestE2E_Spark_ClaimStaticDeposit(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().ClaimStaticDeposit(ctx, wallet.ID, "btc-tx-001", 5000, "mock-ssp-sig", SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to claim deposit: %v", err)
	}

	if resp.Method != "claimStaticDeposit" {
		t.Errorf("Expected method claimStaticDeposit, got %s", resp.Method)
	}
}

func TestE2E_Spark_CreateLightningInvoice(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().CreateLightningInvoice(ctx, wallet.ID, 10000, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to create Lightning invoice: %v", err)
	}

	if resp.Method != "createLightningInvoice" {
		t.Errorf("Expected method createLightningInvoice, got %s", resp.Method)
	}

	if resp.Data.Invoice.EncodedInvoice == "" {
		t.Error("Expected encoded invoice to be returned")
	}

	if resp.Data.Status != "INVOICE_CREATED" {
		t.Errorf("Expected status INVOICE_CREATED, got %s", resp.Data.Status)
	}
}

func TestE2E_Spark_PayLightningInvoice(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().PayLightningInvoice(ctx, wallet.ID, "lnbc100u1pmockinvoice", 100, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to pay Lightning invoice: %v", err)
	}

	if resp.Method != "payLightningInvoice" {
		t.Errorf("Expected method payLightningInvoice, got %s", resp.Method)
	}
}

func TestE2E_Spark_SignMessage(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	resp, err := client.Wallets().Spark().SignMessage(ctx, wallet.ID, "Hello, Spark!", false, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	if resp.Method != "signMessageWithIdentityKey" {
		t.Errorf("Expected method signMessageWithIdentityKey, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Spark_SignWithNonExistentWallet(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Wallets().Spark().SignMessage(ctx, "nonexistent-wallet", "Hello", false, SparkNetworkMainnet, "")
	if err == nil {
		t.Error("Expected error for non-existent wallet")
	}
}

func TestE2E_Spark_MultipleOperations(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}

	// Get balance
	balanceResp, err := client.Wallets().Spark().GetBalance(ctx, wallet.ID, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	if balanceResp.Data.Balance == "" {
		t.Error("Expected balance")
	}

	// Get deposit address
	addrResp, err := client.Wallets().Spark().GetStaticDepositAddress(ctx, wallet.ID, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to get deposit address: %v", err)
	}
	if addrResp.Data.Address == "" {
		t.Error("Expected address")
	}

	// Sign message
	sigResp, err := client.Wallets().Spark().SignMessage(ctx, wallet.ID, "test", false, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}
	if sigResp.Data.Signature == "" {
		t.Error("Expected signature")
	}

	// Create Lightning invoice
	invoiceResp, err := client.Wallets().Spark().CreateLightningInvoice(ctx, wallet.ID, 5000, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}
	if invoiceResp.Data.Invoice.EncodedInvoice == "" {
		t.Error("Expected invoice")
	}
}
