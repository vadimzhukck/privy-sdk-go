package main

import (
	"context"
	"fmt"
	"log"
	"os"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

func main() {
	// Get credentials from environment variables
	appID := os.Getenv("PRIVY_APP_ID")
	appSecret := os.Getenv("PRIVY_APP_SECRET")
	walletID := os.Getenv("PRIVY_WALLET_ID")

	if appID == "" || appSecret == "" {
		log.Fatal("PRIVY_APP_ID and PRIVY_APP_SECRET environment variables are required")
	}

	if walletID == "" {
		log.Fatal("PRIVY_WALLET_ID environment variable is required for this example")
	}

	// Create a new Privy client
	client := privy.NewClient(appID, appSecret)

	ctx := context.Background()

	// Example 1: Sign a message
	fmt.Println("Signing a message...")
	signResp, err := client.Wallets().Ethereum().SignMessage(
		ctx,
		walletID,
		"Hello, Privy!",
		"utf-8",
		"", // Authorization signature (empty for server-owned wallets)
	)
	if err != nil {
		log.Printf("Failed to sign message: %v", err)
	} else {
		fmt.Printf("Signature: %s\n", signResp.Data.Signature)
	}

	// Example 2: Sign a transaction (without sending)
	fmt.Println("\nSigning a transaction...")
	tx := &privy.EthereumTransaction{
		To:      "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", // vitalik.eth
		Value:   "0x2386F26FC10000",                           // 0.01 ETH in wei (hex)
		ChainID: 1,                                            // Ethereum mainnet
	}

	signTxResp, err := client.Wallets().Ethereum().SignTransaction(
		ctx,
		walletID,
		tx,
		"", // Authorization signature
	)
	if err != nil {
		log.Printf("Failed to sign transaction: %v", err)
	} else {
		fmt.Printf("Signed transaction: %s\n", signTxResp.Data.SignedTransaction)
	}

	// Example 3: Send a transaction on Sepolia testnet
	fmt.Println("\nSending a transaction on Sepolia...")
	sepoliaTx := &privy.EthereumTransaction{
		To:      "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Value:   "0x2386F26FC10000", // 0.01 ETH
		ChainID: 11155111,           // Sepolia testnet
	}

	sendResp, err := client.Wallets().Ethereum().SendTransaction(
		ctx,
		walletID,
		sepoliaTx,
		11155111, // Chain ID for CAIP-2 identifier
		false,    // sponsor: false (user pays gas)
		"",       // Authorization signature
	)
	if err != nil {
		log.Printf("Failed to send transaction: %v", err)
	} else {
		fmt.Printf("Transaction hash: %s\n", sendResp.Data.Hash)
		fmt.Printf("CAIP-2: %s\n", sendResp.Data.CAIP2)
	}

	// Example 4: Sign typed data (EIP-712)
	fmt.Println("\nSigning typed data (EIP-712)...")
	typedData := &privy.TypedData{
		Domain: privy.TypedDataDomain{
			Name:              "Example DApp",
			Version:           "1",
			ChainID:           1,
			VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
		},
		Types: map[string][]privy.TypedDataField{
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

	typedDataResp, err := client.Wallets().Ethereum().SignTypedData(
		ctx,
		walletID,
		typedData,
		"", // Authorization signature
	)
	if err != nil {
		log.Printf("Failed to sign typed data: %v", err)
	} else {
		fmt.Printf("Typed data signature: %s\n", typedDataResp.Data.Signature)
	}

	fmt.Println("\nDone!")
}
