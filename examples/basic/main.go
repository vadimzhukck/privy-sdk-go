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

	if appID == "" || appSecret == "" {
		log.Fatal("PRIVY_APP_ID and PRIVY_APP_SECRET environment variables are required")
	}

	// Create a new Privy client
	client := privy.NewClient(appID, appSecret)

	ctx := context.Background()

	// Example 1: Create a user with an email address
	fmt.Println("Creating a user...")
	user, err := client.Users().Create(ctx, &privy.CreateUserRequest{
		LinkedAccounts: []privy.LinkedAccountInput{
			privy.LinkedAccountInputEmail("[email protected]"),
		},
		CreateEthereumWallet: true, // Automatically create an embedded wallet
	})
	if err != nil {
		log.Printf("Failed to create user: %v", err)
	} else {
		fmt.Printf("Created user: %s\n", user.ID)
		fmt.Printf("User created at: %s\n", user.CreatedAtTime())
	}

	// Example 2: List all users
	fmt.Println("\nListing users...")
	users, err := client.Users().List(ctx, &privy.ListOptions{Limit: 10})
	if err != nil {
		log.Printf("Failed to list users: %v", err)
	} else {
		fmt.Printf("Found %d users\n", len(users.Data))
		for _, u := range users.Data {
			fmt.Printf("  - User ID: %s\n", u.ID)
		}
	}

	// Example 3: Create a server-owned wallet
	fmt.Println("\nCreating a wallet...")
	wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
		ChainType: privy.ChainTypeEthereum,
	})
	if err != nil {
		log.Printf("Failed to create wallet: %v", err)
	} else {
		fmt.Printf("Created wallet: %s\n", wallet.ID)
		fmt.Printf("Wallet address: %s\n", wallet.Address)
	}

	// Example 4: Get wallet balance
	if wallet != nil {
		fmt.Println("\nGetting wallet balance...")
		balance, err := client.Wallets().GetBalance(ctx, wallet.ID, &privy.GetBalanceOptions{
			Chain: "ethereum",
		})
		if err != nil {
			log.Printf("Failed to get balance: %v", err)
		} else {
			fmt.Printf("Wallet balance: %s\n", balance.Balance)
		}
	}

	fmt.Println("\nDone!")
}
