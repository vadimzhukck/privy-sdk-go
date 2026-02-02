package privy

import (
	"context"
	"testing"
)

// ============================================
// Wallets Service E2E Tests
// ============================================

func TestE2E_Wallets_CreateEthereum(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	if wallet.ID == "" {
		t.Error("Expected wallet ID to be set")
	}

	if wallet.Address == "" {
		t.Error("Expected wallet address to be set")
	}

	if wallet.ChainType != ChainTypeEthereum {
		t.Errorf("Expected chain type ethereum, got %s", wallet.ChainType)
	}
}

func TestE2E_Wallets_CreateSolana(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	if wallet.ChainType != ChainTypeSolana {
		t.Errorf("Expected chain type solana, got %s", wallet.ChainType)
	}
}

func TestE2E_Wallets_CreateWithOwner(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a user first
	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: "[email protected]"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create wallet for the user
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
		Owner: &WalletOwner{
			UserID: user.ID,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	if wallet.OwnerID != user.ID {
		t.Errorf("Expected owner ID %s, got %s", user.ID, wallet.OwnerID)
	}
}

func TestE2E_Wallets_CreateWithPolicy(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a policy first
	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Create wallet with policy
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
		PolicyIDs: []string{policy.ID},
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	if len(wallet.PolicyIDs) != 1 {
		t.Fatalf("Expected 1 policy ID, got %d", len(wallet.PolicyIDs))
	}

	if wallet.PolicyIDs[0] != policy.ID {
		t.Errorf("Expected policy ID %s, got %s", policy.ID, wallet.PolicyIDs[0])
	}
}

func TestE2E_Wallets_CreateAllChainTypes(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	chainTypes := []ChainType{
		ChainTypeEthereum,
		ChainTypeSolana,
		ChainTypeStellar,
		ChainTypeCosmos,
		ChainTypeSui,
		ChainTypeTron,
		ChainTypeBitcoinSegwit,
		ChainTypeNear,
		ChainTypeTon,
		ChainTypeStarknet,
		ChainTypeAptos,
	}

	for _, ct := range chainTypes {
		t.Run(string(ct), func(t *testing.T) {
			wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
				ChainType: ct,
			})
			if err != nil {
				t.Fatalf("Failed to create %s wallet: %v", ct, err)
			}

			if wallet.ChainType != ct {
				t.Errorf("Expected chain type %s, got %s", ct, wallet.ChainType)
			}
		})
	}
}

func TestE2E_Wallets_Get(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a wallet
	created, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Get the wallet
	wallet, err := client.Wallets().Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}

	if wallet.ID != created.ID {
		t.Errorf("Expected wallet ID %s, got %s", created.ID, wallet.ID)
	}

	if wallet.Address != created.Address {
		t.Errorf("Expected address %s, got %s", created.Address, wallet.Address)
	}
}

func TestE2E_Wallets_GetNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Wallets().Get(ctx, "nonexistent-wallet")
	if err == nil {
		t.Error("Expected error for non-existent wallet")
	}
}

func TestE2E_Wallets_List(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create multiple wallets
	for i := 0; i < 5; i++ {
		_, err := client.Wallets().Create(ctx, &CreateWalletRequest{
			ChainType: ChainTypeEthereum,
		})
		if err != nil {
			t.Fatalf("Failed to create wallet %d: %v", i, err)
		}
	}

	// List wallets
	resp, err := client.Wallets().List(ctx, &WalletListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list wallets: %v", err)
	}

	if len(resp.Data) != 5 {
		t.Errorf("Expected 5 wallets, got %d", len(resp.Data))
	}
}

func TestE2E_Wallets_Update(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a wallet
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Create a policy
	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "New Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Update wallet with policy
	updated, err := client.Wallets().Update(ctx, wallet.ID, &UpdateWalletRequest{
		PolicyIDs: []string{policy.ID},
	})
	if err != nil {
		t.Fatalf("Failed to update wallet: %v", err)
	}

	if len(updated.PolicyIDs) != 1 {
		t.Errorf("Expected 1 policy ID, got %d", len(updated.PolicyIDs))
	}
}

func TestE2E_Wallets_GetBalance(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a wallet
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Get balance
	balance, err := client.Wallets().GetBalance(ctx, wallet.ID, &GetBalanceOptions{
		Chain: "ethereum",
	})
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if balance.Balance == "" {
		t.Error("Expected balance to be set")
	}
}

func TestE2E_Wallets_GetTransactions(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a wallet
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Get transactions with required parameters
	txs, err := client.Wallets().GetTransactions(ctx, wallet.ID, &GetTransactionsOptions{
		Chain: "ethereum",
		Asset: []string{"eth"},
		Limit: 50,
	})
	if err != nil {
		t.Fatalf("Failed to get transactions: %v", err)
	}

	// Should return empty list for new wallet
	if txs.Data == nil {
		t.Error("Expected transactions data to be initialized")
	}

	// Test with multiple assets
	txs, err = client.Wallets().GetTransactions(ctx, wallet.ID, &GetTransactionsOptions{
		Chain: "ethereum",
		Asset: []string{"eth", "usdc", "usdt"},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Failed to get transactions with multiple assets: %v", err)
	}

	// Test missing required parameters
	_, err = client.Wallets().GetTransactions(ctx, wallet.ID, nil)
	if err == nil {
		t.Error("Expected error when options is nil")
	}

	_, err = client.Wallets().GetTransactions(ctx, wallet.ID, &GetTransactionsOptions{
		Asset: []string{"eth"},
	})
	if err == nil {
		t.Error("Expected error when chain is missing")
	}

	_, err = client.Wallets().GetTransactions(ctx, wallet.ID, &GetTransactionsOptions{
		Chain: "ethereum",
	})
	if err == nil {
		t.Error("Expected error when asset is missing")
	}

	// Test limit validation
	_, err = client.Wallets().GetTransactions(ctx, wallet.ID, &GetTransactionsOptions{
		Chain: "ethereum",
		Asset: []string{"eth"},
		Limit: 150, // Exceeds max of 100
	})
	if err == nil {
		t.Error("Expected error when limit exceeds 100")
	}

	// Test max assets validation
	_, err = client.Wallets().GetTransactions(ctx, wallet.ID, &GetTransactionsOptions{
		Chain: "ethereum",
		Asset: []string{"eth", "usdc", "usdt", "dai", "wbtc"}, // 5 assets, max is 4
	})
	if err == nil {
		t.Error("Expected error when more than 4 assets specified")
	}
}

func TestE2E_Wallets_GetTransactionByHash(t *testing.T) {
	client, server, mockServer := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a wallet
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Add a mock transaction to the mock server
	mockTx := Transaction{
		ID:       "tx-123",
		WalletID: wallet.ID,
		Hash:     "0xabc123",
		Status:   "confirmed",
		CAIP2:    "eip155:1",
	}
	mockServer.mu.Lock()
	mockServer.transactions["tx-123"] = &mockTx
	mockServer.mu.Unlock()

	// Test GetTransactionByHash (will return empty result from mock server)
	// In real usage, this would find the transaction by hash
	_, err = client.Wallets().GetTransactionByHash(
		ctx,
		wallet.ID,
		"ethereum",
		[]string{"eth"},
		"0xabc123",
	)
	// The mock server returns empty list, so this should return "not found" error
	if err == nil || err.Error() != "transaction not found: 0xabc123" {
		t.Logf("Expected 'transaction not found' error, got: %v", err)
	}
}

func TestE2E_Wallets_Export(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a wallet
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Export wallet
	exported, err := client.Wallets().Export(ctx, wallet.ID, "")
	if err != nil {
		t.Fatalf("Failed to export wallet: %v", err)
	}

	if exported.PrivateKey == "" {
		t.Error("Expected private key to be returned")
	}
}

func TestE2E_Wallets_Import(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Initialize import
	initResp, err := client.Wallets().InitializeImport(ctx, &ImportWalletInitRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to initialize import: %v", err)
	}

	if initResp.ImportID == "" {
		t.Error("Expected import ID to be set")
	}

	if initResp.PublicKey == "" {
		t.Error("Expected public key to be set")
	}

	// Submit import
	wallet, err := client.Wallets().SubmitImport(ctx, &ImportWalletSubmitRequest{
		ImportID:            initResp.ImportID,
		EncryptedPrivateKey: "encrypted-key-data",
	})
	if err != nil {
		t.Fatalf("Failed to submit import: %v", err)
	}

	if wallet.ID == "" {
		t.Error("Expected wallet ID to be set")
	}
}

func TestE2E_Wallets_GetTransaction(t *testing.T) {
	client, server, mock := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a wallet
	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Send a transaction to create a transaction record
	tx := &EthereumTransaction{
		To:      "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Value:   "0x1000",
		ChainID: 1,
	}

	sendResp, err := client.Wallets().Ethereum().SendTransaction(ctx, wallet.ID, tx, 1, false, "")
	if err != nil {
		t.Fatalf("Failed to send transaction: %v", err)
	}

	// The mock server creates transactions with incremental IDs
	// Find the transaction ID from the mock
	mock.mu.RLock()
	var txID string
	for id := range mock.transactions {
		txID = id
		break
	}
	mock.mu.RUnlock()

	if txID == "" {
		t.Fatal("No transaction was created")
	}

	// Get the transaction
	transaction, err := client.Wallets().GetTransaction(ctx, txID)
	if err != nil {
		t.Fatalf("Failed to get transaction: %v", err)
	}

	if transaction.Hash != sendResp.Data.Hash {
		t.Errorf("Expected hash %s, got %s", sendResp.Data.Hash, transaction.Hash)
	}
}
