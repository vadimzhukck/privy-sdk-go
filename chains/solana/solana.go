// Package solana provides a high-level helper for Solana transactions
// using Privy's /rpc endpoint and the solana-go SDK.
//
// The Transfer method builds a system transfer instruction, serializes
// the transaction, and delegates signing + submission to Privy.
package solana

import (
	"context"
	"fmt"
	"strconv"

	solanago "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	privy "github.com/vadimzhukck/privy-sdk-go"
)

const (
	// MainnetCAIP2 is the CAIP-2 identifier for Solana mainnet.
	MainnetCAIP2 = "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"
	// DevnetCAIP2 is the CAIP-2 identifier for Solana devnet.
	DevnetCAIP2 = "solana:EtWTRABZaYq6iMfeYKouRu166VU2xqa1"
)

// Helper provides convenience methods for Solana operations using Privy wallets.
type Helper struct {
	client    *privy.Client
	rpcURL    string
	caip2     string
	rpcClient *rpc.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithRPCURL sets the Solana JSON-RPC endpoint URL.
func WithRPCURL(url string) Option {
	return func(h *Helper) {
		h.rpcURL = url
		h.rpcClient = rpc.New(url)
	}
}

// WithCAIP2 sets the CAIP-2 chain identifier for network selection.
func WithCAIP2(caip2 string) Option {
	return func(h *Helper) { h.caip2 = caip2 }
}

// WithDevnet configures the helper for Solana devnet.
func WithDevnet() Option {
	return func(h *Helper) {
		h.caip2 = DevnetCAIP2
		h.rpcURL = rpc.DevNet_RPC
		h.rpcClient = rpc.New(rpc.DevNet_RPC)
	}
}

// WithTestnet configures the helper for Solana devnet (alias for WithDevnet).
func WithTestnet() Option {
	return WithDevnet()
}

// NewHelper creates a new Solana helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client: client,
		rpcURL: rpc.MainNetBeta_RPC,
		caip2:  MainnetCAIP2,
	}
	h.rpcClient = rpc.New(h.rpcURL)

	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("solana") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Transfer sends native SOL from a Privy wallet to a destination address.
// amount is in lamports (1 SOL = 1_000_000_000 lamports) as a decimal string.
// Returns the transaction signature (hash).
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet address from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("solana: get wallet: %w", err)
	}

	// Parse amount
	lamports, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("solana: invalid amount %q: %w", amount, err)
	}

	// Parse addresses
	fromPubKey, err := solanago.PublicKeyFromBase58(wallet.Address)
	if err != nil {
		return "", fmt.Errorf("solana: invalid sender address %q: %w", wallet.Address, err)
	}

	toPubKey, err := solanago.PublicKeyFromBase58(destination)
	if err != nil {
		return "", fmt.Errorf("solana: invalid destination address %q: %w", destination, err)
	}

	// Get recent blockhash
	recent, err := h.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return "", fmt.Errorf("solana: get recent blockhash: %w", err)
	}

	// Build transfer instruction
	transferIx := system.NewTransferInstruction(lamports, fromPubKey, toPubKey).Build()

	// Build transaction
	tx, err := solanago.NewTransaction(
		[]solanago.Instruction{transferIx},
		recent.Value.Blockhash,
		solanago.TransactionPayer(fromPubKey),
	)
	if err != nil {
		return "", fmt.Errorf("solana: build transaction: %w", err)
	}

	// Serialize to base64
	txBase64, err := tx.ToBase64()
	if err != nil {
		return "", fmt.Errorf("solana: serialize transaction: %w", err)
	}

	// Sign and send via Privy
	resp, err := h.client.Wallets().Solana().SignAndSendTransactionWithCAIP2(
		ctx, walletID, txBase64, h.caip2, "",
	)
	if err != nil {
		return "", fmt.Errorf("solana: sign and send: %w", err)
	}

	return resp.Data.Hash, nil
}
