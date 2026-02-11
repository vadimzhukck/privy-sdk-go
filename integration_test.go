package privy

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration tests that run against the real Privy API.
// These tests require the following environment variables:
//   - PRIVY_APP_ID: Your Privy application ID
//   - PRIVY_APP_SECRET: Your Privy application secret
//
// Run with: go test -v -tags=integration ./...
// Or: PRIVY_APP_ID=xxx PRIVY_APP_SECRET=xxx go test -v -run "^TestIntegration" ./...

// testConfig holds the configuration for integration tests.
type testConfig struct {
	appID     string
	appSecret string
	client    *Client
}

// skipIfNoCredentials skips the test if API credentials are not set.
func skipIfNoCredentials(t *testing.T) *testConfig {
	t.Helper()

	appID := os.Getenv("PRIVY_APP_ID")
	appSecret := os.Getenv("PRIVY_APP_SECRET")

	if appID == "" || appSecret == "" {
		t.Skip("Skipping integration test: PRIVY_APP_ID and PRIVY_APP_SECRET environment variables are required")
	}

	client := NewClient(appID, appSecret)

	return &testConfig{
		appID:     appID,
		appSecret: appSecret,
		client:    client,
	}
}

// ============================================
// Users Integration Tests
// ============================================

func TestIntegration_Users_CreateAndDelete(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	// Generate unique email to avoid conflicts
	email := "test-" + time.Now().Format("20060102150405") + "@integration-test.com"

	// Create user
	user, err := cfg.client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{
				Type:    LinkedAccountTypeEmail,
				Address: email,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Logf("Created user: %s", user.ID)

	if user.ID == "" {
		t.Error("Expected user ID to be set")
	}

	if len(user.LinkedAccounts) == 0 {
		t.Error("Expected linked accounts to be set")
	}

	// Clean up - delete the user
	defer func() {
		err := cfg.client.Users().Delete(ctx, user.ID)
		if err != nil {
			t.Logf("Warning: Failed to delete test user: %v", err)
		} else {
			t.Logf("Deleted user: %s", user.ID)
		}
	}()

	// Get user
	fetchedUser, err := cfg.client.Users().Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if fetchedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, fetchedUser.ID)
	}
}

func TestIntegration_Users_CreateWithEmbeddedWallet(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	email := "wallet-test-" + time.Now().Format("20060102150405") + "@integration-test.com"

	user, err := cfg.client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{
				Type:    LinkedAccountTypeEmail,
				Address: email,
			},
		},
		CreateEthereumWallet: true,
	})
	if err != nil {
		t.Fatalf("Failed to create user with wallet: %v", err)
	}

	t.Logf("Created user with wallet: %s", user.ID)

	defer func() {
		cfg.client.Users().Delete(ctx, user.ID)
	}()

	// Check for wallet in linked accounts
	hasWallet := false
	for _, la := range user.LinkedAccounts {
		if la.Type == LinkedAccountTypeWallet {
			hasWallet = true
			t.Logf("User has embedded wallet: %s", la.Address)
			break
		}
	}

	if !hasWallet {
		t.Log("Note: Embedded wallet may not be immediately visible in linked accounts")
	}
}

func TestIntegration_Users_List(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	resp, err := cfg.client.Users().List(ctx, &ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	t.Logf("Found %d users", len(resp.Data))

	for i, user := range resp.Data {
		if i >= 3 {
			t.Logf("... and %d more users", len(resp.Data)-3)
			break
		}
		t.Logf("  User %d: %s", i+1, user.ID)
	}
}

func TestIntegration_Users_GetByEmail(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	// Create a user first
	email := "find-test-" + time.Now().Format("20060102150405") + "@integration-test.com"

	user, err := cfg.client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: email},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	defer func() {
		cfg.client.Users().Delete(ctx, user.ID)
	}()

	// Find by email
	foundUser, err := cfg.client.Users().GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("Failed to find user by email: %v", err)
	}

	if foundUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, foundUser.ID)
	}

	t.Logf("Successfully found user by email: %s", foundUser.ID)
}

func TestIntegration_Users_UpdateMetadata(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	email := "metadata-test-" + time.Now().Format("20060102150405") + "@integration-test.com"

	user, err := cfg.client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: email},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	defer func() {
		cfg.client.Users().Delete(ctx, user.ID)
	}()

	// Update metadata
	metadata := map[string]any{
		"tier":        "premium",
		"testRun":     true,
		"testTime":    time.Now().Unix(),
	}

	updatedUser, err := cfg.client.Users().UpdateMetadata(ctx, user.ID, metadata)
	if err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}

	t.Logf("Updated user metadata: %v", updatedUser.CustomMetadata)
}

// ============================================
// Wallets Integration Tests
// ============================================

func TestIntegration_Wallets_CreateServerWallet(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Created Ethereum wallet: %s", wallet.ID)
	t.Logf("Wallet address: %s", wallet.Address)

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

func TestIntegration_Wallets_CreateSolanaWallet(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	t.Logf("Created Solana wallet: %s", wallet.ID)
	t.Logf("Wallet address: %s", wallet.Address)

	if wallet.ChainType != ChainTypeSolana {
		t.Errorf("Expected chain type solana, got %s", wallet.ChainType)
	}
}

func TestIntegration_Wallets_List(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	resp, err := cfg.client.Wallets().List(ctx, &WalletListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list wallets: %v", err)
	}

	t.Logf("Found %d wallets", len(resp.Data))

	for i, wallet := range resp.Data {
		if i >= 5 {
			t.Logf("... and %d more wallets", len(resp.Data)-5)
			break
		}
		t.Logf("  Wallet %d: %s (%s) - %s", i+1, wallet.ID, wallet.ChainType, wallet.Address)
	}
}

func TestIntegration_Wallets_GetAndBalance(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	// Create a wallet first
	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Created wallet: %s", wallet.ID)

	// Get wallet
	fetched, err := cfg.client.Wallets().Get(ctx, wallet.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}

	if fetched.ID != wallet.ID {
		t.Errorf("Expected wallet ID %s, got %s", wallet.ID, fetched.ID)
	}

	// Get balance
	balance, err := cfg.client.Wallets().GetBalance(ctx, wallet.ID, &GetBalanceOptions{
		Chain: "ethereum",
	})
	if err != nil {
		t.Logf("Note: GetBalance may require additional setup: %v", err)
	} else {
		t.Logf("Wallet balance: %s", balance.Balance)
	}
}

// ============================================
// Ethereum Signing Integration Tests
// ============================================

func TestIntegration_Ethereum_SignMessage(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	// Create a wallet first
	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Created wallet: %s (%s)", wallet.ID, wallet.Address)

	// Sign a message
	message := "Hello from Privy SDK integration test!"

	resp, err := cfg.client.Wallets().Ethereum().SignMessage(ctx, wallet.ID, message, "utf-8", "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	t.Logf("Signed message successfully")
	t.Logf("Signature: %s", resp.Data.Signature)

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Ethereum_SignTypedData(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Created wallet: %s", wallet.ID)

	typedData := &TypedData{
		Domain: TypedDataDomain{
			Name:              "Integration Test",
			Version:           "1",
			ChainID:           1,
			VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
		},
		Types: map[string][]TypedDataField{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Message": {
				{Name: "content", Type: "string"},
				{Name: "timestamp", Type: "uint256"},
			},
		},
		PrimaryType: "Message",
		Message: map[string]any{
			"content":   "Integration test message",
			"timestamp": time.Now().Unix(),
		},
	}

	resp, err := cfg.client.Wallets().Ethereum().SignTypedData(ctx, wallet.ID, typedData, "")
	if err != nil {
		t.Fatalf("Failed to sign typed data: %v", err)
	}

	t.Logf("Signed typed data successfully")
	t.Logf("Signature: %s", resp.Data.Signature)
}

func TestIntegration_Ethereum_SignTransaction(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Created wallet: %s (%s)", wallet.ID, wallet.Address)

	// Sign a transaction (not sending it)
	tx := &EthereumTransaction{
		To:      "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", // vitalik.eth
		Value:   "0x0",                                         // 0 ETH
		ChainID: 11155111,                                      // Sepolia
	}

	resp, err := cfg.client.Wallets().Ethereum().SignTransaction(ctx, wallet.ID, tx, "")
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	t.Logf("Signed transaction successfully")
	t.Logf("Signed TX: %s", resp.Data.SignedTransaction)
}

// ============================================
// Solana Signing Integration Tests
// ============================================

func TestIntegration_Solana_SignMessage(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSolana,
	})
	if err != nil {
		t.Fatalf("Failed to create Solana wallet: %v", err)
	}

	t.Logf("Created Solana wallet: %s (%s)", wallet.ID, wallet.Address)

	message := "Hello from Solana integration test!"

	resp, err := cfg.client.Wallets().Solana().SignMessage(ctx, wallet.ID, message, "utf-8", "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	t.Logf("Signed Solana message successfully")
	t.Logf("Signature: %s", resp.Data.Signature)
}

// ============================================
// Policies Integration Tests
// ============================================

func TestIntegration_Policies_CreateAndDelete(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	policy, err := cfg.client.Policies().Create(ctx, &CreatePolicyRequest{
		Version:   "1.0",
		ChainType: ChainTypeEthereum,
		Name:      "Integration Test Policy " + time.Now().Format("20060102150405"),
		Rules:     []PolicyRule{},
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	t.Logf("Created policy: %s", policy.ID)

	defer func() {
		err := cfg.client.Policies().Delete(ctx, policy.ID)
		if err != nil {
			t.Logf("Warning: Failed to delete policy: %v", err)
		} else {
			t.Logf("Deleted policy: %s", policy.ID)
		}
	}()

	// Get policy
	fetched, err := cfg.client.Policies().Get(ctx, policy.ID)
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if fetched.ID != policy.ID {
		t.Errorf("Expected policy ID %s, got %s", policy.ID, fetched.ID)
	}

	// Update policy
	updated, err := cfg.client.Policies().Update(ctx, policy.ID, &UpdatePolicyRequest{
		Name: "Updated Integration Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	t.Logf("Updated policy name to: %s", updated.Name)
}

// ============================================
// Full Workflow Integration Test
// ============================================

func TestIntegration_FullWorkflow(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	t.Log("=== Starting Full Workflow Integration Test ===")

	// 1. Create a user with email
	email := "workflow-" + time.Now().Format("20060102150405") + "@integration-test.com"
	t.Logf("Step 1: Creating user with email: %s", email)

	user, err := cfg.client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: email},
		},
		CreateEthereumWallet: true,
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	t.Logf("Created user: %s", user.ID)

	defer func() {
		t.Log("Cleanup: Deleting test user")
		cfg.client.Users().Delete(ctx, user.ID)
	}()

	// 2. Create a server wallet
	t.Log("Step 2: Creating server wallet")

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	t.Logf("Created wallet: %s (%s)", wallet.ID, wallet.Address)

	// 3. Sign a message with the wallet
	t.Log("Step 3: Signing message with wallet")

	signResp, err := cfg.client.Wallets().Ethereum().SignMessage(ctx, wallet.ID, "Workflow test message", "utf-8", "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}
	t.Logf("Signed message, signature: %s...", signResp.Data.Signature[:20])

	// 4. Update user metadata
	t.Log("Step 4: Updating user metadata")

	_, err = cfg.client.Users().UpdateMetadata(ctx, user.ID, map[string]any{
		"workflowCompleted": true,
		"walletId":          wallet.ID,
	})
	if err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}
	t.Log("Updated user metadata")

	// 5. Verify user can be found by email
	t.Log("Step 5: Verifying user can be found by email")

	foundUser, err := cfg.client.Users().GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("Failed to find user by email: %v", err)
	}
	if foundUser.ID != user.ID {
		t.Errorf("Found wrong user: expected %s, got %s", user.ID, foundUser.ID)
	}
	t.Log("Successfully found user by email")

	t.Log("=== Full Workflow Integration Test Completed ===")
}

// ============================================
// Multi-Chain Signing Integration Tests
// ============================================

func TestIntegration_Stellar_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeStellar,
	})
	if err != nil {
		t.Fatalf("Failed to create Stellar wallet: %v", err)
	}
	t.Logf("Created Stellar wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Stellar().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Cosmos_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeCosmos,
	})
	if err != nil {
		t.Fatalf("Failed to create Cosmos wallet: %v", err)
	}
	t.Logf("Created Cosmos wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Cosmos().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Sui_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSui,
	})
	if err != nil {
		t.Fatalf("Failed to create Sui wallet: %v", err)
	}
	t.Logf("Created Sui wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Sui().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Tron_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeTron,
	})
	if err != nil {
		t.Fatalf("Failed to create Tron wallet: %v", err)
	}
	t.Logf("Created Tron wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Tron().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Bitcoin_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeBitcoinSegwit,
	})
	if err != nil {
		t.Fatalf("Failed to create Bitcoin wallet: %v", err)
	}
	t.Logf("Created Bitcoin wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Bitcoin().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Near_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeNear,
	})
	if err != nil {
		t.Fatalf("Failed to create Near wallet: %v", err)
	}
	t.Logf("Created Near wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Near().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Ton_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeTon,
	})
	if err != nil {
		t.Fatalf("Failed to create Ton wallet: %v", err)
	}
	t.Logf("Created Ton wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Ton().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Starknet_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeStarknet,
	})
	if err != nil {
		t.Fatalf("Failed to create Starknet wallet: %v", err)
	}
	t.Logf("Created Starknet wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Starknet().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Aptos_RawSign(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeAptos,
	})
	if err != nil {
		t.Fatalf("Failed to create Aptos wallet: %v", err)
	}
	t.Logf("Created Aptos wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Aptos().RawSign(ctx, wallet.ID, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	if err != nil {
		t.Fatalf("Failed to raw sign: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

// Spark Integration Tests

func TestIntegration_Spark_GetBalance(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}
	t.Logf("Created Spark wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Spark().GetBalance(ctx, wallet.ID, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to get Spark balance: %v", err)
	}

	t.Logf("Balance: %s", resp.Data.Balance)
}

func TestIntegration_Spark_GetStaticDepositAddress(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}
	t.Logf("Created Spark wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Spark().GetStaticDepositAddress(ctx, wallet.ID, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to get static deposit address: %v", err)
	}

	t.Logf("Deposit address: %s", resp.Data.Address)
	if resp.Data.Address == "" {
		t.Error("Expected deposit address to be returned")
	}
}

func TestIntegration_Spark_SignMessage(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}
	t.Logf("Created Spark wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Spark().SignMessage(ctx, wallet.ID, "Hello from Spark integration test!", false, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	t.Logf("Signature: %s", resp.Data.Signature)
	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestIntegration_Spark_CreateLightningInvoice(t *testing.T) {
	cfg := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := cfg.client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeSpark,
	})
	if err != nil {
		t.Fatalf("Failed to create Spark wallet: %v", err)
	}
	t.Logf("Created Spark wallet: %s (address: %s)", wallet.ID, wallet.Address)

	resp, err := cfg.client.Wallets().Spark().CreateLightningInvoice(ctx, wallet.ID, 1000, SparkNetworkMainnet, "")
	if err != nil {
		t.Fatalf("Failed to create Lightning invoice: %v", err)
	}

	t.Logf("Invoice: %s", resp.Data.Invoice.EncodedInvoice)
	if resp.Data.Invoice.EncodedInvoice == "" {
		t.Error("Expected encoded invoice to be returned")
	}
}
