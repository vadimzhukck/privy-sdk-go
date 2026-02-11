package aptos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	privy "github.com/vadimzhukck/privy-sdk-go"
)

func skipIfNoCredentials(t *testing.T) *privy.Client {
	t.Helper()
	appID := os.Getenv("PRIVY_APP_ID")
	appSecret := os.Getenv("PRIVY_APP_SECRET")
	if appID == "" || appSecret == "" {
		t.Skip("Skipping: PRIVY_APP_ID and PRIVY_APP_SECRET required")
	}
	return privy.NewClient(appID, appSecret)
}

// fundDevnetAccount funds an Aptos devnet account via the faucet.
func fundDevnetAccount(address string, amount uint64) error {
	faucetURL := fmt.Sprintf("https://faucet.devnet.aptoslabs.com/mint?address=%s&amount=%d", address, amount)
	resp, err := http.Post(faucetURL, "", nil)
	if err != nil {
		return fmt.Errorf("faucet request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("faucet returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func TestIntegration_Aptos_Transfer(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx := context.Background()

	// Create Aptos helper configured for devnet (has open faucet API)
	h := NewHelper(client, WithNodeURL("https://fullnode.devnet.aptoslabs.com/v1"))

	// Create a Privy wallet
	wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
		ChainType: privy.ChainTypeAptos,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	t.Logf("Created wallet: %s (address: %s, pubkey: %s)", wallet.ID, wallet.Address, wallet.PublicKey)

	// Fund the wallet via devnet faucet (1 APT = 100_000_000 octas)
	t.Log("Funding wallet via devnet faucet...")
	err = fundDevnetAccount(wallet.Address, 200_000_000)
	if err != nil {
		t.Fatalf("Failed to fund wallet: %v", err)
	}

	// Wait for funding transaction to be confirmed
	time.Sleep(5 * time.Second)

	// Verify balance via Aptos SDK
	aptClient, err := aptos.NewClient(aptos.DevnetConfig)
	if err != nil {
		t.Fatalf("Failed to create aptos client: %v", err)
	}
	var senderAddr aptos.AccountAddress
	if err := senderAddr.ParseStringRelaxed(wallet.Address); err != nil {
		t.Fatalf("Failed to parse address: %v", err)
	}
	balance, err := aptClient.AccountAPTBalance(senderAddr)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	t.Logf("Wallet balance: %d octas (%f APT)", balance, float64(balance)/1e8)

	// Create a second wallet as destination
	destWallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
		ChainType: privy.ChainTypeAptos,
	})
	if err != nil {
		t.Fatalf("Failed to create destination wallet: %v", err)
	}
	t.Logf("Destination wallet: %s (address: %s)", destWallet.ID, destWallet.Address)

	// Fund destination so the account exists on-chain
	err = fundDevnetAccount(destWallet.Address, 100_000_000)
	if err != nil {
		t.Fatalf("Failed to fund destination: %v", err)
	}
	time.Sleep(5 * time.Second)

	// Transfer 10000 octas (0.0001 APT)
	t.Log("Transferring 10000 octas...")
	txHash, err := h.Transfer(ctx, wallet.ID, destWallet.Address, 10000)
	if err != nil {
		t.Fatalf("Transfer failed: %v", err)
	}

	t.Logf("Transfer successful! TX hash: %s", txHash)
	t.Logf("Explorer: https://explorer.aptoslabs.com/txn/%s?network=devnet", txHash)

	if txHash == "" {
		t.Error("Expected non-empty transaction hash")
	}
}

// TestIntegration_Aptos_WalletPublicKey verifies the Privy API returns public keys.
func TestIntegration_Aptos_WalletPublicKey(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
		ChainType: privy.ChainTypeAptos,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Wallet ID: %s", wallet.ID)
	t.Logf("Address: %s", wallet.Address)
	t.Logf("PublicKey: %s", wallet.PublicKey)

	if wallet.PublicKey == "" {
		t.Error("Expected public key to be returned")
	}

	// Ed25519 public keys can be 32 bytes (64 hex) or have a 1-byte prefix (66 hex)
	pk := wallet.PublicKey
	if len(pk) > 2 && pk[:2] == "0x" {
		pk = pk[2:]
	}
	if len(pk) != 64 && len(pk) != 66 {
		t.Errorf("Unexpected public key length: %d hex chars (expected 64 or 66)", len(pk))
	}

	// Verify Get returns the same public key
	fetched, err := client.Wallets().Get(ctx, wallet.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}
	if fetched.PublicKey != wallet.PublicKey {
		t.Errorf("Get returned different public key: %s vs %s", fetched.PublicKey, wallet.PublicKey)
	}
}

// TestIntegration_Aptos_CreateWalletResponse verifies the raw API response includes public_key.
func TestIntegration_Aptos_CreateWalletResponse(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx := context.Background()

	wallet, err := client.Wallets().Create(ctx, &privy.CreateWalletRequest{
		ChainType: privy.ChainTypeAptos,
	})
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	data, err := json.Marshal(wallet)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if _, ok := raw["public_key"]; !ok || raw["public_key"] == "" {
		t.Errorf("public_key not found or empty in wallet response: %s", string(data))
	} else {
		t.Logf("public_key present: %v", raw["public_key"])
	}
}
