// Package ethereum provides a high-level helper for Ethereum transactions
// using Privy's /rpc endpoint.
//
// The Transfer method wraps the core SDK's SendTransaction to provide
// a simple one-call native ETH transfer.
package ethereum

import (
	"context"
	"fmt"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

// Helper provides convenience methods for Ethereum operations using Privy wallets.
type Helper struct {
	client  *privy.Client
	chainID int64
}

// Option configures the Helper.
type Option func(*Helper)

// WithChainID sets the EVM chain ID (default: 1 for Ethereum mainnet).
func WithChainID(chainID int64) Option {
	return func(h *Helper) { h.chainID = chainID }
}

// WithTestnet configures the helper for Ethereum Sepolia testnet (chain ID 11155111).
func WithTestnet() Option {
	return WithChainID(11155111)
}

// NewHelper creates a new Ethereum helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:  client,
		chainID: 1,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("ethereum") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Transfer sends native ETH from a Privy wallet to a destination address.
// amount is in wei as a hex string (e.g. "0xDE0B6B3A7640000" for 1 ETH)
// or decimal string (e.g. "1000000000000000000").
// Returns the transaction hash.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	tx := &privy.EthereumTransaction{
		To:    destination,
		Value: amount,
	}

	resp, err := h.client.Wallets().Ethereum().SendTransaction(ctx, walletID, tx, h.chainID, false, "")
	if err != nil {
		return "", fmt.Errorf("ethereum: send transaction: %w", err)
	}

	return resp.Data.Hash, nil
}

// TransferSponsored sends native ETH with gas sponsorship enabled.
func (h *Helper) TransferSponsored(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	tx := &privy.EthereumTransaction{
		To:    destination,
		Value: amount,
	}

	resp, err := h.client.Wallets().Ethereum().SendTransaction(ctx, walletID, tx, h.chainID, true, "")
	if err != nil {
		return "", fmt.Errorf("ethereum: send sponsored transaction: %w", err)
	}

	return resp.Data.Hash, nil
}

// SendTransaction sends a custom Ethereum transaction.
// This gives full control over gas, data, nonce, etc.
func (h *Helper) SendTransaction(ctx context.Context, walletID string, tx *privy.EthereumTransaction, sponsor bool) (string, error) {
	resp, err := h.client.Wallets().Ethereum().SendTransaction(ctx, walletID, tx, h.chainID, sponsor, "")
	if err != nil {
		return "", fmt.Errorf("ethereum: send transaction: %w", err)
	}

	return resp.Data.Hash, nil
}
