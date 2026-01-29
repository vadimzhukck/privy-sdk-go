package privy

import (
	"context"
	"testing"
)

// ============================================
// Ethereum Signing E2E Tests
// ============================================

func TestE2E_Ethereum_SignMessage(t *testing.T) {
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

	// Sign a message
	resp, err := client.Wallets().Ethereum().SignMessage(ctx, wallet.ID, "Hello, Privy!", "utf-8", "")
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	if resp.Method != "personal_sign" {
		t.Errorf("Expected method personal_sign, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}

	if resp.Data.Encoding != "hex" {
		t.Errorf("Expected encoding hex, got %s", resp.Data.Encoding)
	}
}

func TestE2E_Ethereum_SignMessageHex(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Sign a hex-encoded message
	resp, err := client.Wallets().Ethereum().SignMessage(ctx, wallet.ID, "0x48656c6c6f", "hex", "")
	if err != nil {
		t.Fatalf("Failed to sign hex message: %v", err)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Ethereum_SignTransaction(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	tx := &EthereumTransaction{
		To:      "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Value:   "0x2386F26FC10000", // 0.01 ETH
		ChainID: 1,
	}

	resp, err := client.Wallets().Ethereum().SignTransaction(ctx, wallet.ID, tx, "")
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	if resp.Method != "eth_signTransaction" {
		t.Errorf("Expected method eth_signTransaction, got %s", resp.Method)
	}

	if resp.Data.SignedTransaction == "" {
		t.Error("Expected signed transaction to be returned")
	}

	if resp.Data.Encoding != "rlp" {
		t.Errorf("Expected encoding rlp, got %s", resp.Data.Encoding)
	}
}

func TestE2E_Ethereum_SignTransactionWithAllFields(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	tx := &EthereumTransaction{
		To:                   "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		From:                 wallet.Address,
		Value:                "0x2386F26FC10000",
		Data:                 "0x",
		ChainID:              1,
		GasLimit:             "0x5208",
		MaxFeePerGas:         "0x59682F00",
		MaxPriorityFeePerGas: "0x3B9ACA00",
		Nonce:                0,
		Type:                 2, // EIP-1559
	}

	resp, err := client.Wallets().Ethereum().SignTransaction(ctx, wallet.ID, tx, "")
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	if resp.Data.SignedTransaction == "" {
		t.Error("Expected signed transaction to be returned")
	}
}

func TestE2E_Ethereum_SendTransaction(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	tx := &EthereumTransaction{
		To:      "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Value:   "0x2386F26FC10000",
		ChainID: 11155111, // Sepolia
	}

	resp, err := client.Wallets().Ethereum().SendTransaction(ctx, wallet.ID, tx, 11155111, false, "")
	if err != nil {
		t.Fatalf("Failed to send transaction: %v", err)
	}

	if resp.Method != "eth_sendTransaction" {
		t.Errorf("Expected method eth_sendTransaction, got %s", resp.Method)
	}

	if resp.Data.Hash == "" {
		t.Error("Expected transaction hash to be returned")
	}

	if resp.Data.CAIP2 == "" {
		t.Error("Expected CAIP2 identifier to be returned")
	}
}

func TestE2E_Ethereum_SendTransactionWithSponsor(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	tx := &EthereumTransaction{
		To:      "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Value:   "0x0",
		Data:    "0xa9059cbb", // ERC20 transfer selector
		ChainID: 1,
	}

	// Send with gas sponsorship
	resp, err := client.Wallets().Ethereum().SendTransaction(ctx, wallet.ID, tx, 1, true, "")
	if err != nil {
		t.Fatalf("Failed to send sponsored transaction: %v", err)
	}

	if resp.Data.Hash == "" {
		t.Error("Expected transaction hash to be returned")
	}
}

func TestE2E_Ethereum_SignTypedData(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	typedData := &TypedData{
		Domain: TypedDataDomain{
			Name:              "Example DApp",
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
			"Mail": {
				{Name: "from", Type: "address"},
				{Name: "to", Type: "address"},
				{Name: "contents", Type: "string"},
			},
		},
		PrimaryType: "Mail",
		Message: map[string]any{
			"from":     "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
			"to":       "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
			"contents": "Hello, Bob!",
		},
	}

	resp, err := client.Wallets().Ethereum().SignTypedData(ctx, wallet.ID, typedData, "")
	if err != nil {
		t.Fatalf("Failed to sign typed data: %v", err)
	}

	if resp.Method != "eth_signTypedData_v4" {
		t.Errorf("Expected method eth_signTypedData_v4, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Ethereum_SignHash(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	hash := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	resp, err := client.Wallets().Ethereum().SignHash(ctx, wallet.ID, hash, "")
	if err != nil {
		t.Fatalf("Failed to sign hash: %v", err)
	}

	if resp.Method != "secp256k1_sign" {
		t.Errorf("Expected method secp256k1_sign, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Ethereum_RawSign(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	hash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	resp, err := client.Wallets().Ethereum().RawSign(ctx, wallet.ID, hash, "")
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

func TestE2E_Ethereum_SignUserOperation(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	userOp := map[string]any{
		"sender":               wallet.Address,
		"nonce":                "0x0",
		"initCode":             "0x",
		"callData":             "0x",
		"callGasLimit":         "0x5208",
		"verificationGasLimit": "0x5208",
		"preVerificationGas":   "0x5208",
		"maxFeePerGas":         "0x59682F00",
		"maxPriorityFeePerGas": "0x3B9ACA00",
		"paymasterAndData":     "0x",
		"signature":            "0x",
	}

	entryPoint := "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"

	resp, err := client.Wallets().Ethereum().SignUserOperation(ctx, wallet.ID, userOp, entryPoint, 1, "")
	if err != nil {
		t.Fatalf("Failed to sign user operation: %v", err)
	}

	if resp.Method != "eth_signUserOperation" {
		t.Errorf("Expected method eth_signUserOperation, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Ethereum_Sign7702Authorization(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	contractAddress := "0x1234567890123456789012345678901234567890"

	resp, err := client.Wallets().Ethereum().Sign7702Authorization(ctx, wallet.ID, 1, contractAddress, 0, "")
	if err != nil {
		t.Fatalf("Failed to sign 7702 authorization: %v", err)
	}

	if resp.Method != "eth_sign7702Authorization" {
		t.Errorf("Expected method eth_sign7702Authorization, got %s", resp.Method)
	}

	if resp.Data.Signature == "" {
		t.Error("Expected signature to be returned")
	}
}

func TestE2E_Ethereum_SignWithNonExistentWallet(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Wallets().Ethereum().SignMessage(ctx, "nonexistent-wallet", "Hello", "utf-8", "")
	if err == nil {
		t.Error("Expected error for non-existent wallet")
	}
}

func TestE2E_Ethereum_MultipleSignOperations(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &CreateWalletRequest{
		ChainType: ChainTypeEthereum,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Sign multiple messages
	for i := 0; i < 5; i++ {
		resp, err := client.Wallets().Ethereum().SignMessage(ctx, wallet.ID, "Message "+string(rune('A'+i)), "utf-8", "")
		if err != nil {
			t.Fatalf("Failed to sign message %d: %v", i, err)
		}
		if resp.Data.Signature == "" {
			t.Errorf("Expected signature for message %d", i)
		}
	}
}
