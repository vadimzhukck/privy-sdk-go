# Privy Go SDK

A Go SDK for the [Privy](https://www.privy.io/) wallet infrastructure API. This SDK provides a simple and idiomatic way to interact with Privy's REST API for user management, wallet operations, and transaction signing.

## Installation

```bash
go get github.com/vadimzhukck/privy-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    privy "github.com/vadimzhukck/privy-sdk"
)

func main() {
    // Create a client with your Privy credentials
    client := privy.NewClient("your-app-id", "your-app-secret")

    ctx := context.Background()

    // Create a user with an email address
    user, err := client.Users().Create(ctx, &privy.CreateUserRequest{
        LinkedAccounts: []privy.LinkedAccountInput{
            privy.LinkedAccountInputEmail("[email protected]"),
        },
        CreateEthereumWallet: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created user: %s\n", user.ID)

    // Create a wallet
    wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
        ChainType: privy.ChainTypeEthereum,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created wallet: %s at address %s\n", wallet.ID, wallet.Address)
}
```

## Features

### User Management

```go
// Create a user with an email address
user, err := client.Users().Create(ctx, &privy.CreateUserRequest{
    LinkedAccounts: []privy.LinkedAccountInput{
        privy.LinkedAccountInputEmail("[email protected]"),
    },
})

// Create a user with a Google account
user, err := client.Users().Create(ctx, &privy.CreateUserRequest{
    LinkedAccounts: []privy.LinkedAccountInput{
        privy.LinkedAccountInputGoogle("google-subject-id", "John Doe", "[email protected]"),
    },
})

// Create a user with multiple linked accounts and an embedded Ethereum wallet
// Note: Only Ethereum wallets can be created during user creation.
// For Solana or other chains, use client.Wallets().Create() after creating the user.
user, err := client.Users().Create(ctx, &privy.CreateUserRequest{
    LinkedAccounts: []privy.LinkedAccountInput{
        privy.LinkedAccountInputEmail("[email protected]"),
        privy.LinkedAccountInputWallet("0x1234567890abcdef..."),
        privy.LinkedAccountInputTwitter("twitter-user-id", "johndoe", "John Doe"),
    },
    CreateEthereumWallet: true,
})

// To create a Solana wallet for a user, create the user first, then create the wallet:
user, _ := client.Users().Create(ctx, &privy.CreateUserRequest{
    LinkedAccounts: []privy.LinkedAccountInput{
        privy.LinkedAccountInputEmail("[email protected]"),
    },
})
solanaWallet, _ := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
    ChainType: privy.ChainTypeSolana,
    Owner:     &privy.WalletOwner{UserID: user.ID},
})

// Get a user by ID
user, err := client.Users().Get(ctx, "did:privy:xxxxx")

// List all users with pagination
users, err := client.Users().List(ctx, &privy.ListOptions{Limit: 100})

// Find user by email
user, err := client.Users().GetByEmail(ctx, "[email protected]")

// Find user by wallet address
user, err := client.Users().GetByWalletAddress(ctx, "0x...")

// Update user metadata
user, err := client.Users().UpdateMetadata(ctx, "did:privy:xxxxx", map[string]any{
    "tier": "premium",
})

// Delete a user
err := client.Users().Delete(ctx, "did:privy:xxxxx")

// Get wallets from user's linked accounts
user, _ := client.Users().Get(ctx, "did:privy:xxxxx")
wallets := user.GetWallets()                              // All wallets
ethWallets := user.GetWalletsByChain(privy.ChainTypeEthereum) // Ethereum only
```

#### Linked Account Helpers

The SDK provides helper functions for creating linked accounts with the correct fields:

```go
// Basic account types
privy.LinkedAccountInputEmail("[email protected]")
privy.LinkedAccountInputPhone("+14155551234")  // E.164 format
privy.LinkedAccountInputWallet("0x1234...")
privy.LinkedAccountInputCustomAuth("custom-user-id")

// OAuth providers
privy.LinkedAccountInputGoogle(subject, name, email)
privy.LinkedAccountInputTwitter(subject, username, name)
privy.LinkedAccountInputDiscord(subject, username, name, email)
privy.LinkedAccountInputGithub(subject, username, name, email)
privy.LinkedAccountInputApple(subject, email)
privy.LinkedAccountInputLinkedIn(subject, name, email)
privy.LinkedAccountInputSpotify(subject, name, email)
privy.LinkedAccountInputInstagram(subject, username)
privy.LinkedAccountInputTiktok(subject, username, name)
privy.LinkedAccountInputTwitch(subject, username, name, email)

// Social platforms
privy.LinkedAccountInputFarcaster(fid, username, displayName, bio, pfpURL)
privy.LinkedAccountInputTelegram(telegramUserID, username, firstName, lastName, photoURL)
```

### Wallet Operations

```go
// Create an Ethereum wallet
wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
    ChainType: privy.ChainTypeEthereum,
})

// Create a Solana wallet
wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
    ChainType: privy.ChainTypeSolana,
})

// Create a user-owned wallet
wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
    ChainType: privy.ChainTypeEthereum,
    Owner: &privy.WalletOwner{
        UserID: "did:privy:xxxxx",
    },
})

// Get wallet details
wallet, err := client.Wallets().Get(ctx, "wallet-id")

// Get wallet balance
balance, err := client.Wallets().GetBalance(ctx, "wallet-id")

// List all wallets
wallets, err := client.Wallets().List(ctx, &privy.WalletListOptions{Limit: 50})

// List wallets by user ID
userWallets, err := client.Wallets().List(ctx, &privy.WalletListOptions{
    UserID: "did:privy:xxxxx",
})

// List wallets by user ID and chain type
ethWallets, err := client.Wallets().List(ctx, &privy.WalletListOptions{
    UserID:    "did:privy:xxxxx",
    ChainType: privy.ChainTypeEthereum, // optional
})

// Get transaction history (chain and asset are required)
txs, err := client.Wallets().GetTransactions(ctx, "wallet-id", &privy.GetTransactionsOptions{
    Chain: "ethereum",
    Asset: []string{"eth", "usdc"},
    Limit: 50,
})

// Get a specific transaction by hash
tx, err := client.Wallets().GetTransactionByHash(
    ctx,
    "wallet-id",
    "ethereum",
    []string{"eth"},
    "0xabc123...",
)

// Filter transactions by hash using GetTransactions
txs, err := client.Wallets().GetTransactions(ctx, "wallet-id", &privy.GetTransactionsOptions{
    Chain:  "solana",
    Asset:  []string{"sol", "usdc"},
    TxHash: "5j7s...", // Filter by specific transaction hash
    Limit:  1,
})
```

### Ethereum Signing

```go
// Sign a message
resp, err := client.Wallets().Ethereum().SignMessage(
    ctx,
    "wallet-id",
    "Hello, World!",
    "utf-8",
    "", // authorization signature (empty for server-owned wallets)
)

// Sign a transaction
tx := &privy.EthereumTransaction{
    To:      "0xRecipientAddress",
    Value:   "0x2386F26FC10000", // 0.01 ETH in wei (hex)
    ChainID: 1,
}
resp, err := client.Wallets().Ethereum().SignTransaction(ctx, "wallet-id", tx, "")

// Send a transaction
resp, err := client.Wallets().Ethereum().SendTransaction(
    ctx,
    "wallet-id",
    tx,
    1,     // chainID
    false, // sponsor (gas sponsorship)
    "",    // authorization signature
)

// Sign typed data (EIP-712)
typedData := &privy.TypedData{
    Domain: privy.TypedDataDomain{
        Name:    "My DApp",
        Version: "1",
        ChainID: 1,
    },
    Types: map[string][]privy.TypedDataField{
        "Message": {{Name: "content", Type: "string"}},
    },
    PrimaryType: "Message",
    Message:     map[string]any{"content": "Hello"},
}
resp, err := client.Wallets().Ethereum().SignTypedData(ctx, "wallet-id", typedData, "")

// Sign a raw hash
resp, err := client.Wallets().Ethereum().SignHash(ctx, "wallet-id", "0xhash...", "")

// Sign a user operation (ERC-4337)
resp, err := client.Wallets().Ethereum().SignUserOperation(
    ctx,
    "wallet-id",
    userOp,      // map[string]any
    entryPoint,  // entry point address
    chainID,
    "",
)
```

### Solana Signing

```go
// Sign and send a transaction
resp, err := client.Wallets().Solana().SignAndSendTransaction(
    ctx,
    "wallet-id",
    "base64EncodedTransaction",
    "", // authorization signature
)

// Sign a transaction (without sending)
resp, err := client.Wallets().Solana().SignTransaction(
    ctx,
    "wallet-id",
    "base64EncodedTransaction",
    "",
)

// Sign a message
resp, err := client.Wallets().Solana().SignMessage(
    ctx,
    "wallet-id",
    "Hello, Solana!",
    "utf-8",
    "",
)
```

### Policies

```go
// Create a policy
policy, err := client.Policies().Create(ctx, &privy.CreatePolicyRequest{
    Version:   "1.0",
    ChainType: privy.ChainTypeEthereum,
    Name:      "Transfer Limit Policy",
    Rules:     []privy.PolicyRule{},
})

// Add a rule to a policy
rule, err := client.Policies().AddRule(ctx, "policy-id", &privy.CreateRuleRequest{
    Action: "allow",
    Conditions: []privy.RuleCondition{
        {Type: "max_value", Value: "1000000000000000000"}, // 1 ETH
    },
})

// Update a policy
policy, err := client.Policies().Update(ctx, "policy-id", &privy.UpdatePolicyRequest{
    Name: "Updated Policy Name",
})

// Delete a policy
err := client.Policies().Delete(ctx, "policy-id")
```

### Condition Sets

```go
// Create a condition set
cs, err := client.ConditionSets().Create(ctx, &privy.CreateConditionSetRequest{
    Name: "Allowlist",
})

// Add items to a condition set
items, err := client.ConditionSets().AddItems(ctx, "condition-set-id", []privy.ConditionSetItemInput{
    {Value: "0xAddress1"},
    {Value: "0xAddress2"},
})

// List items with pagination
items, err := client.ConditionSets().ListItems(ctx, "condition-set-id", &privy.ListOptions{
    Limit: 100,
})

// Delete an item
err := client.ConditionSets().DeleteItem(ctx, "condition-set-id", "item-id")
```

### Key Quorums

```go
// Create a key quorum
kq, err := client.KeyQuorums().Create(ctx, &privy.CreateKeyQuorumRequest{
    PublicKey: "0x...",
})

// Get a key quorum
kq, err := client.KeyQuorums().Get(ctx, "key-quorum-id")

// Delete a key quorum
err := client.KeyQuorums().Delete(ctx, "key-quorum-id")
```

### Token Verification (JWKS)

Verify Privy access tokens on your backend using JWKS:

```go
// Verify an access token
claims, err := client.Auth().VerifyToken(ctx, accessToken)
if err != nil {
    if errors.Is(err, privy.ErrTokenExpired) {
        // Token has expired
    } else if errors.Is(err, privy.ErrInvalidSignature) {
        // Invalid signature
    }
    log.Fatal(err)
}

// Access token claims
userID := claims.UserID()           // did:privy:xxxxx
isExpired := claims.IsExpired()     // bool
expiresIn := claims.ExpiresIn()     // time.Duration

// Access custom claims
if val, ok := claims.GetClaim("custom_field"); ok {
    fmt.Println("Custom field:", val)
}

// Verify with custom options
claims, err := client.Auth().VerifyTokenWithOptions(ctx, accessToken, &privy.VerifyTokenOptions{
    Audience:     "your-app-id",           // Override audience check
    AllowExpired: false,                   // Set to true for debugging
    ClockSkew:    30 * time.Second,        // Allow for clock differences
})

// Manually refresh JWKS cache
err := client.Auth().RefreshJWKS(ctx)

// Fetch JWKS for custom verification
jwks, err := client.Auth().GetJWKS(ctx)
```

### Webhooks

Handle incoming Privy webhooks with signature verification:

```go
// Create a webhook handler with your signing secret
handler := privy.NewWebhookHandler("whsec_your-webhook-signing-secret")

// Set custom tolerance for timestamp validation (default: 5 minutes)
handler = handler.WithTolerance(300) // 5 minutes

// Register typed event handlers
handler.OnUserCreated(func(e *privy.UserCreatedEvent) {
    fmt.Printf("User created: %s\n", e.UserID)
})

handler.OnUserUpdated(func(e *privy.UserUpdatedEvent) {
    fmt.Printf("User updated: %s\n", e.UserID)
})

handler.OnUserDeleted(func(e *privy.UserDeletedEvent) {
    fmt.Printf("User deleted: %s\n", e.UserID)
})

handler.OnWalletCreated(func(e *privy.WalletCreatedEvent) {
    fmt.Printf("Wallet created: %s at %s\n", e.WalletID, e.Address)
})

handler.OnTransactionCompleted(func(e *privy.TransactionEvent) {
    fmt.Printf("Transaction completed: %s\n", e.TransactionID)
})

// Handle wallet fund events (deposits/withdrawals)
handler.OnWalletFundsDeposited(func(e *privy.WalletFundsEvent) {
    fmt.Printf("Funds deposited to wallet %s: %s %s from %s\n",
        e.WalletID, e.Amount, e.Asset.Type, e.Sender)
    fmt.Printf("Transaction hash: %s\n", e.TransactionHash)
})

handler.OnWalletFundsWithdrawn(func(e *privy.WalletFundsEvent) {
    fmt.Printf("Funds withdrawn from wallet %s: %s %s to %s\n",
        e.WalletID, e.Amount, e.Asset.Type, e.Recipient)
    fmt.Printf("Transaction hash: %s\n", e.TransactionHash)
})

// Handle MFA events
handler.OnMFAEnabled(func(e *privy.MFAEvent) {
    fmt.Printf("MFA enabled for user %s: %s\n", e.UserID, e.Method)
})

// Use as HTTP handler
http.Handle("/webhook", handler)

// Or manually verify and parse
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    event, err := handler.VerifyAndParse(r)
    if err != nil {
        if errors.Is(err, privy.ErrInvalidWebhookSignature) {
            http.Error(w, "Invalid signature", http.StatusUnauthorized)
            return
        }
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    // Handle event by type
    switch e := event.Type; e {
    case privy.WebhookEventUserCreated:
        data, _ := event.GetUserCreatedData()
        fmt.Printf("User created: %s\n", data.UserID)
    case privy.WebhookEventWalletCreated:
        data, _ := event.GetWalletCreatedData()
        fmt.Printf("Wallet created: %s\n", data.WalletID)
    }

    w.WriteHeader(http.StatusOK)
}
```

**Supported Webhook Events:**

**User Events:**
- `user.created` - New user registered
- `user.updated` - User profile updated
- `user.deleted` - User deleted
- `user.authenticated` - User logged in
- `user.linked_account` - Account linked to user
- `user.unlinked_account` - Account unlinked from user
- `user.updated_account` - User account updated
- `user.transferred_account` - Account transferred between users
- `user.wallet_created` - Wallet created for user

**Wallet Events:**
- `wallet.created` - New wallet created
- `wallet.transferred` - Wallet ownership transferred
- `wallet.funds_deposited` - Funds deposited into wallet
- `wallet.funds_withdrawn` - Funds withdrawn from wallet
- `wallet.private_key_export` - Private key exported
- `wallet.recovery_setup` - Recovery method set up
- `wallet.recovered` - Wallet recovered

**Transaction Events:**
- `transaction.created` - Transaction initiated
- `transaction.broadcasted` - Transaction broadcasted to network
- `transaction.confirmed` - Transaction confirmed on chain
- `transaction.completed` - Transaction succeeded
- `transaction.execution_reverted` - Transaction execution reverted
- `transaction.still_pending` - Transaction still pending
- `transaction.failed` - Transaction failed
- `transaction.replaced` - Transaction replaced
- `transaction.provider_error` - Provider error occurred

**MFA Events:**
- `mfa.enabled` - MFA enabled for user
- `mfa.disabled` - MFA disabled for user

## Supported Chain Types

- `ethereum` - Ethereum and EVM-compatible chains
- `solana` - Solana
- `stellar` - Stellar
- `cosmos` - Cosmos
- `sui` - Sui
- `tron` - Tron
- `bitcoin-segwit` - Bitcoin (SegWit)
- `near` - NEAR
- `ton` - TON
- `starknet` - StarkNet
- `aptos` - Aptos

## Configuration Options

```go
// Custom HTTP client
httpClient := &http.Client{Timeout: 60 * time.Second}
client := privy.NewClient(appID, appSecret, privy.WithHTTPClient(httpClient))

// Custom timeout
client := privy.NewClient(appID, appSecret, privy.WithTimeout(60*time.Second))

// Custom base URL (for testing)
client := privy.NewClient(appID, appSecret, privy.WithBaseURL("https://custom-api.privy.io/v1"))
```

## Error Handling

```go
user, err := client.Users().Get(ctx, "invalid-id")
if err != nil {
    if apiErr, ok := err.(*privy.APIError); ok {
        fmt.Printf("API Error: %s (status: %d)\n", apiErr.Message, apiErr.StatusCode)
    } else {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Environment Variables

For the examples, set the following environment variables:

```bash
export PRIVY_APP_ID="your-app-id"
export PRIVY_APP_SECRET="your-app-secret"
export PRIVY_WALLET_ID="your-wallet-id"  # For transaction examples
```

## Testing

### Unit Tests (with mock server)

Run unit tests that use a mock server (no API credentials needed):

```bash
# Run all tests
go test -v ./...

# Run only e2e tests with mock server
go test -v -run "^TestE2E" ./...

# Using Docker
docker-compose run --rm test
docker-compose run --rm test-e2e
```

### Integration Tests (with real Privy API)

Run integration tests against the real Privy API:

```bash
# Set credentials
export PRIVY_APP_ID="your-app-id"
export PRIVY_APP_SECRET="your-app-secret"

# Run all integration tests
go test -v -run "^TestIntegration" ./...

# Run specific integration tests
go test -v -run "^TestIntegration_Users" ./...
go test -v -run "^TestIntegration_Wallets" ./...
go test -v -run "^TestIntegration_Ethereum" ./...
go test -v -run "^TestIntegration_FullWorkflow" ./...

# Using Docker
docker-compose run --rm test-integration
docker-compose run --rm test-integration-users
docker-compose run --rm test-integration-wallets
docker-compose run --rm test-integration-signing
docker-compose run --rm test-integration-workflow
```

### Docker Commands

```bash
# Run all tests (mock server)
docker-compose run --rm test

# Run integration tests (requires .env file or env vars)
PRIVY_APP_ID=xxx PRIVY_APP_SECRET=xxx docker-compose run --rm test-integration

# Run with coverage
docker-compose run --rm test-coverage

# Lint and format
docker-compose run --rm lint
```

## Resources

- [Privy Documentation](https://docs.privy.io/)
- [Privy API Reference](https://docs.privy.io/api-reference/introduction)
- [Privy Dashboard](https://console.privy.io/)

## License

MIT License
