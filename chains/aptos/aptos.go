// Package aptos provides a high-level helper for Aptos transactions
// using Privy's raw_sign endpoint and the Aptos Go SDK.
package aptos

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
	privy "github.com/vadimzhukck/privy-sdk-go"
)

// Helper provides convenience methods for Aptos operations using Privy wallets.
type Helper struct {
	client      *privy.Client
	aptosClient *aptos.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithNodeURL sets the Aptos node URL.
func WithNodeURL(url string) Option {
	return func(h *Helper) {
		cfg := aptos.NetworkConfig{NodeUrl: url}
		c, err := aptos.NewClient(cfg)
		if err == nil {
			h.aptosClient = c
		}
	}
}

// WithTestnet configures the helper for Aptos devnet.
func WithTestnet() Option {
	return WithNodeURL("https://fullnode.devnet.aptoslabs.com/v1")
}

// WithAptosClient sets a pre-configured Aptos client.
func WithAptosClient(c *aptos.Client) Option {
	return func(h *Helper) { h.aptosClient = c }
}

// NewHelper creates a new Aptos helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{client: client}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("aptos") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	if h.aptosClient == nil {
		c, err := aptos.NewClient(aptos.MainnetConfig)
		if err == nil {
			h.aptosClient = c
		}
	}
	return h
}

// Transfer sends APT from a Privy wallet to a destination address.
// Amount is in octas (1 APT = 100_000_000 octas).
// The walletID is the Privy wallet ID. The public key and address are fetched
// from Privy automatically.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount uint64) (string, error) {
	if h.aptosClient == nil {
		return "", fmt.Errorf("aptos client not initialized")
	}

	// Get wallet from Privy to obtain address and public key
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("aptos: get wallet: %w", err)
	}
	if wallet.PublicKey == "" {
		return "", fmt.Errorf("aptos: wallet %s has no public key", walletID)
	}

	// Parse sender address
	sender, err := parseAddress(wallet.Address)
	if err != nil {
		return "", fmt.Errorf("aptos: invalid sender address: %w", err)
	}

	// Parse recipient address
	recipient, err := parseAddress(destination)
	if err != nil {
		return "", fmt.Errorf("aptos: invalid recipient address: %w", err)
	}

	// Serialize arguments for the transfer entry function
	recipientBytes, err := bcs.Serialize(&recipient)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to serialize recipient: %w", err)
	}

	amountBytes, err := bcs.SerializeU64(amount)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to serialize amount: %w", err)
	}

	// Build raw transaction
	rawTxn, err := h.aptosClient.BuildTransaction(sender,
		aptos.TransactionPayload{
			Payload: &aptos.EntryFunction{
				Module: aptos.ModuleId{
					Address: aptos.AccountOne,
					Name:    "aptos_account",
				},
				Function: "transfer",
				ArgTypes: []aptos.TypeTag{},
				Args:     [][]byte{recipientBytes, amountBytes},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to build transaction: %w", err)
	}

	// Get signing message — includes sha3_256("APTOS::RawTransaction") prefix + BCS bytes
	signingMessage, err := rawTxn.SigningMessage()
	if err != nil {
		return "", fmt.Errorf("aptos: failed to get signing message: %w", err)
	}

	// Sign via Privy raw_sign — pass the signing message bytes directly as "hash".
	// For Ed25519 wallets, Privy signs the provided bytes directly (Ed25519 does
	// its own internal SHA-512 hashing as part of the signing algorithm).
	hashHex := "0x" + hex.EncodeToString(signingMessage)
	signResp, err := h.client.RawSign(ctx, walletID, hashHex)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to sign: %w", err)
	}

	// Decode the Ed25519 signature
	sigBytes, err := decodeHexSignature(signResp.Data.Signature)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to decode signature: %w", err)
	}

	sig := &crypto.Ed25519Signature{}
	copy(sig.Inner[:], sigBytes)

	// Decode the public key from the wallet.
	// Privy returns it as hex, possibly with a 1-byte scheme prefix (0x00 = Ed25519).
	pubKeyBytes, err := decodeHexSignature(wallet.PublicKey)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to decode public key: %w", err)
	}
	// Strip the Ed25519 scheme prefix byte if present (33 bytes → 32 bytes)
	if len(pubKeyBytes) == 33 && pubKeyBytes[0] == 0x00 {
		pubKeyBytes = pubKeyBytes[1:]
	}

	pubKey := &crypto.Ed25519PublicKey{}
	if err := pubKey.FromBytes(pubKeyBytes); err != nil {
		return "", fmt.Errorf("aptos: failed to parse public key (%d bytes): %w", len(pubKeyBytes), err)
	}

	// Build authenticator
	auth := &crypto.AccountAuthenticator{
		Variant: crypto.AccountAuthenticatorEd25519,
		Auth: &crypto.Ed25519Authenticator{
			PubKey: pubKey,
			Sig:    sig,
		},
	}

	// Create signed transaction
	signedTxn, err := rawTxn.SignedTransactionWithAuthenticator(auth)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to create signed transaction: %w", err)
	}

	// Submit to the network
	submitResult, err := h.aptosClient.SubmitTransaction(signedTxn)
	if err != nil {
		return "", fmt.Errorf("aptos: failed to submit transaction: %w", err)
	}

	return submitResult.Hash, nil
}

func parseAddress(addr string) (aptos.AccountAddress, error) {
	var address aptos.AccountAddress
	err := address.ParseStringRelaxed(addr)
	return address, err
}

func decodeHexSignature(sig string) ([]byte, error) {
	sig = strings.TrimPrefix(sig, "0x")
	return hex.DecodeString(sig)
}
