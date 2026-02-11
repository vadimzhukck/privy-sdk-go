// Package tron provides a high-level helper for Tron transactions
// using Privy's raw_sign endpoint and the Tron HTTP API.
//
// No external SDK dependency is needed — transactions are built via
// the Tron full-node HTTP API (e.g. TronGrid).
package tron

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

// Helper provides high-level Tron transaction methods using Privy wallets.
type Helper struct {
	client     *privy.Client
	rpcURL     string
	httpClient *http.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithRPCURL sets the Tron full-node HTTP API URL.
func WithRPCURL(url string) Option {
	return func(h *Helper) {
		h.rpcURL = url
	}
}

// WithTestnet configures the helper for Tron Shasta testnet.
func WithTestnet() Option {
	return WithRPCURL("https://api.shasta.trongrid.io")
}

// WithHTTPClient sets a custom HTTP client for Tron API calls.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Helper) {
		h.httpClient = c
	}
}

// NewHelper creates a new Tron helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:     client,
		rpcURL:     "https://api.trongrid.io",
		httpClient: http.DefaultClient,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("tron") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// createTransactionRequest is the request body for /wallet/createtransaction.
type createTransactionRequest struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       int64  `json:"amount"`
	Visible      bool   `json:"visible"`
}

// tronTransaction represents the response from /wallet/createtransaction.
type tronTransaction struct {
	Visible    bool            `json:"visible"`
	TxID       string          `json:"txid"`
	RawData    json.RawMessage `json:"raw_data"`
	RawDataHex string          `json:"raw_data_hex"`
}

// broadcastRequest is the request body for /wallet/broadcasttransaction.
type broadcastRequest struct {
	Visible    bool            `json:"visible"`
	TxID       string          `json:"txid"`
	RawData    json.RawMessage `json:"raw_data"`
	RawDataHex string          `json:"raw_data_hex"`
	Signature  []string        `json:"signature"`
}

// broadcastResponse is the response from /wallet/broadcasttransaction.
type broadcastResponse struct {
	Result  bool   `json:"result"`
	TxID    string `json:"txid"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Transfer sends native TRX from a Privy wallet to a destination address.
// amount is in sun (1 TRX = 1_000_000 sun) as a decimal string.
// Returns the transaction ID (hash).
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet address from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("tron: get wallet: %w", err)
	}

	// Parse amount
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("tron: invalid amount %q: %w", amount, err)
	}

	// Step 1: Create unsigned transaction via Tron HTTP API
	txn, err := h.createTransaction(ctx, wallet.Address, destination, amountInt)
	if err != nil {
		return "", fmt.Errorf("tron: create transaction: %w", err)
	}

	if txn.TxID == "" {
		return "", fmt.Errorf("tron: empty transaction ID from API")
	}

	// Step 2: Sign the transaction hash (txID is SHA-256 of raw_data)
	hashHex := "0x" + txn.TxID
	signResp, err := h.client.RawSign(ctx, walletID, hashHex)
	if err != nil {
		return "", fmt.Errorf("tron: sign transaction: %w", err)
	}

	// Step 3: Broadcast the signed transaction
	sig := strings.TrimPrefix(signResp.Data.Signature, "0x")
	err = h.broadcastTransaction(ctx, txn, sig)
	if err != nil {
		return "", fmt.Errorf("tron: broadcast transaction: %w", err)
	}

	return txn.TxID, nil
}

// TransferTRC20 sends a TRC-20 token from a Privy wallet to a destination.
func (h *Helper) TransferTRC20(ctx context.Context, walletID string, contractAddress string, destination string, amount string) (string, error) {
	return "", fmt.Errorf("tron: TransferTRC20 not yet implemented")
}

// createTransaction calls the Tron API to build an unsigned transfer transaction.
func (h *Helper) createTransaction(ctx context.Context, from, to string, amount int64) (*tronTransaction, error) {
	reqBody := &createTransactionRequest{
		OwnerAddress: from,
		ToAddress:    to,
		Amount:       amount,
		Visible:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := h.rpcURL + "/wallet/createtransaction"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tron API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var txn tronTransaction
	if err := json.Unmarshal(respBody, &txn); err != nil {
		return nil, fmt.Errorf("failed to parse transaction response: %w", err)
	}

	return &txn, nil
}

// broadcastTransaction submits a signed transaction to the Tron network.
func (h *Helper) broadcastTransaction(ctx context.Context, txn *tronTransaction, signature string) error {
	reqBody := &broadcastRequest{
		Visible:    txn.Visible,
		TxID:       txn.TxID,
		RawData:    txn.RawData,
		RawDataHex: txn.RawDataHex,
		Signature:  []string{signature},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := h.rpcURL + "/wallet/broadcasttransaction"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tron API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result broadcastResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse broadcast response: %w", err)
	}

	if !result.Result {
		return fmt.Errorf("broadcast failed: %s - %s", result.Code, result.Message)
	}

	return nil
}

// RawSign signs a pre-computed hash using the Tron wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
