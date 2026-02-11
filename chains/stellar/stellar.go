// Package stellar provides a high-level helper for Stellar transactions
// using Privy's raw_sign endpoint and the Stellar Go SDK.
//
// Transactions are built using txnbuild, hashed, signed via Privy,
// and submitted to a Horizon server.
package stellar

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	privy "github.com/vadimzhukck/privy-sdk-go"
)

// Helper provides high-level Stellar transaction methods using Privy wallets.
type Helper struct {
	client        *privy.Client
	horizonURL    string
	networkPass   string
	horizonClient horizonclient.ClientInterface
}

// Option configures the Helper.
type Option func(*Helper)

// WithHorizonURL sets the Horizon API URL.
func WithHorizonURL(url string) Option {
	return func(h *Helper) {
		h.horizonURL = url
		h.horizonClient = &horizonclient.Client{HorizonURL: url}
	}
}

// WithNetworkPassphrase sets the Stellar network passphrase.
func WithNetworkPassphrase(passphrase string) Option {
	return func(h *Helper) {
		h.networkPass = passphrase
	}
}

// WithTestnet configures the helper for Stellar testnet.
func WithTestnet() Option {
	return func(h *Helper) {
		h.horizonURL = "https://horizon-testnet.stellar.org"
		h.networkPass = network.TestNetworkPassphrase
		h.horizonClient = &horizonclient.Client{HorizonURL: "https://horizon-testnet.stellar.org"}
	}
}

// WithHorizonClient sets a custom Horizon client.
func WithHorizonClient(c horizonclient.ClientInterface) Option {
	return func(h *Helper) { h.horizonClient = c }
}

// NewHelper creates a new Stellar helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:      client,
		horizonURL:  "https://horizon.stellar.org",
		networkPass: network.PublicNetworkPassphrase,
	}
	h.horizonClient = &horizonclient.Client{HorizonURL: h.horizonURL}

	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("stellar") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Transfer sends native XLM from a Privy wallet to a destination address.
// amount is in XLM as a decimal string (e.g. "100.50").
// Returns the transaction hash.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet info from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("stellar: get wallet: %w", err)
	}

	// Load source account from Horizon
	accountReq := horizonclient.AccountRequest{AccountID: wallet.Address}
	sourceAccount, err := h.horizonClient.AccountDetail(accountReq)
	if err != nil {
		return "", fmt.Errorf("stellar: load account: %w", err)
	}

	// Build payment operation
	paymentOp := &txnbuild.Payment{
		Destination: destination,
		Amount:      amount,
		Asset:       txnbuild.NativeAsset{},
	}

	// Build transaction
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{paymentOp},
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewTimeout(300),
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("stellar: build transaction: %w", err)
	}

	// Hash for signing
	txHash, err := tx.Hash(h.networkPass)
	if err != nil {
		return "", fmt.Errorf("stellar: hash transaction: %w", err)
	}

	// Sign via Privy raw_sign (Ed25519 — signs the hash bytes directly)
	hashHex := "0x" + hex.EncodeToString(txHash[:])
	signResp, err := h.client.RawSign(ctx, walletID, hashHex)
	if err != nil {
		return "", fmt.Errorf("stellar: sign transaction: %w", err)
	}

	// Decode Ed25519 signature
	sigHex := strings.TrimPrefix(signResp.Data.Signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", fmt.Errorf("stellar: decode signature: %w", err)
	}

	// Build decorated signature (hint = last 4 bytes of public key)
	fromAddr, err := keypair.ParseAddress(wallet.Address)
	if err != nil {
		return "", fmt.Errorf("stellar: parse address: %w", err)
	}
	hint := fromAddr.Hint()

	decoratedSig := xdr.DecoratedSignature{
		Hint:      xdr.SignatureHint(hint),
		Signature: xdr.Signature(sigBytes),
	}

	// Attach signature to transaction
	signedTx, err := tx.AddSignatureDecorated(decoratedSig)
	if err != nil {
		return "", fmt.Errorf("stellar: add signature: %w", err)
	}

	// Submit to Horizon
	resp, err := h.horizonClient.SubmitTransaction(signedTx)
	if err != nil {
		return "", fmt.Errorf("stellar: submit transaction: %w", err)
	}

	return resp.Hash, nil
}

// PaymentWithAsset sends a Stellar asset from a Privy wallet to a destination.
func (h *Helper) PaymentWithAsset(ctx context.Context, walletID string, destination string, amount string, assetCode string, assetIssuer string) (string, error) {
	return "", fmt.Errorf("stellar: PaymentWithAsset not yet implemented")
}

// RawSign signs a pre-computed hash using the Stellar wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
