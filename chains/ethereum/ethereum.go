// Package ethereum provides a high-level helper for Ethereum transactions
// using Privy's /rpc endpoint.
//
// The Transfer method wraps the core SDK's SendTransaction to provide
// a simple one-call native ETH transfer.
package ethereum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

var (
	// ErrZeroAmount is returned when the transfer amount is zero or negative.
	ErrZeroAmount = errors.New("ethereum: transfer amount must be greater than zero")
	// ErrInvalidAmount is returned when the amount string cannot be parsed.
	ErrInvalidAmount = errors.New("ethereum: invalid amount format")
)

// parseAmount parses a wei amount from either a hex string (0x-prefixed) or decimal string.
// Returns an error if the value is zero or negative.
func parseAmount(amount string) (*big.Int, error) {
	v := new(big.Int)
	if strings.HasPrefix(amount, "0x") || strings.HasPrefix(amount, "0X") {
		if _, ok := v.SetString(amount[2:], 16); !ok {
			return nil, ErrInvalidAmount
		}
	} else {
		if _, ok := v.SetString(amount, 10); !ok {
			return nil, ErrInvalidAmount
		}
	}
	if v.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	return v, nil
}

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
//
// NOTE: When sweeping the entire wallet balance, callers must subtract the
// estimated gas cost (gasLimit * gasPrice) from the amount. Sending the full
// balance as value will cause the transaction to be rejected because
// value + gas > balance. Use TransferSponsored for gas-sponsored sweeps.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	if _, err := parseAmount(amount); err != nil {
		return "", err
	}

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
// Because gas is sponsored, the full amount can be sent without deducting gas costs.
func (h *Helper) TransferSponsored(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	if _, err := parseAmount(amount); err != nil {
		return "", err
	}

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

// TransferERC20 sends an ERC-20 token transfer from a Privy wallet.
// contractAddress is the token contract address.
// destination is the recipient address.
// amount is the token amount in base units (e.g. "1000000" for 1 USDC with 6 decimals).
func (h *Helper) TransferERC20(ctx context.Context, walletID string, contractAddress string, destination string, amount string) (string, error) {
	amt, err := parseAmount(amount)
	if err != nil {
		return "", err
	}

	// ERC-20 transfer(address,uint256) function selector = 0xa9059cbb
	// Encode: 4 bytes selector + 32 bytes address (left-padded) + 32 bytes amount (left-padded)
	paddedAddr := fmt.Sprintf("%064s", strings.TrimPrefix(destination, "0x"))
	paddedAmt := fmt.Sprintf("%064x", amt)
	data := "0xa9059cbb" + paddedAddr + paddedAmt

	tx := &privy.EthereumTransaction{
		To:    contractAddress,
		Data:  data,
		Value: "0x0", // No ETH value for ERC-20 transfer
	}

	resp, err := h.client.Wallets().Ethereum().SendTransaction(ctx, walletID, tx, h.chainID, false, "")
	if err != nil {
		return "", fmt.Errorf("ethereum: erc20 transfer: %w", err)
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

// GetERC20Balance returns the ERC-20 token balance for a wallet address.
// Returns the raw balance as a decimal string (e.g. "9500000" for 9.5 USDC).
func (h *Helper) GetERC20Balance(ctx context.Context, walletAddress string, contractAddress string) (string, error) {
	// balanceOf(address) selector = 0x70a08231
	paddedAddr := fmt.Sprintf("%064s", strings.TrimPrefix(strings.ToLower(walletAddress), "0x"))
	data := "0x70a08231" + paddedAddr

	rpcURL := "https://mainnet.base.org"
	if h.chainID == 1 {
		rpcURL = "https://eth.llamarpc.com"
	} else if h.chainID == 84532 {
		rpcURL = "https://sepolia.base.org"
	} else if h.chainID == 8453 {
		rpcURL = "https://mainnet.base.org"
	}

	payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":"%s","data":"%s"},"latest"]}`, contractAddress, data)

	req, err := http.NewRequestWithContext(ctx, "POST", rpcURL, strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("ethereum: create balance request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ethereum: balance request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("ethereum: decode balance response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("ethereum: rpc error: %s", result.Error.Message)
	}

	// Parse hex result to decimal string
	balance := new(big.Int)
	if _, ok := balance.SetString(strings.TrimPrefix(result.Result, "0x"), 16); !ok {
		return "", fmt.Errorf("ethereum: parse balance hex: %s", result.Result)
	}
	return balance.String(), nil
}
