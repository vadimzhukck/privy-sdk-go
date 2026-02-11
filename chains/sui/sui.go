// Package sui provides a high-level helper for Sui transactions
// using Privy's raw_sign endpoint and the Sui JSON-RPC API.
//
// Transactions are built via the unsafe_transferSui RPC method,
// hashed with Blake2b-256 (with intent prefix), signed via Privy,
// and executed via sui_executeTransactionBlock.
package sui

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	privy "github.com/vadimzhukck/privy-sdk-go"
	"golang.org/x/crypto/blake2b"
)

// Helper provides high-level Sui transaction methods using Privy wallets.
type Helper struct {
	client     *privy.Client
	rpcURL     string
	httpClient *http.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithRPCURL sets the Sui JSON-RPC endpoint URL.
func WithRPCURL(url string) Option {
	return func(h *Helper) {
		h.rpcURL = url
	}
}

// WithTestnet configures the helper for Sui testnet.
func WithTestnet() Option {
	return WithRPCURL("https://fullnode.testnet.sui.io:443")
}

// WithHTTPClient sets a custom HTTP client for RPC calls.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Helper) {
		h.httpClient = c
	}
}

// NewHelper creates a new Sui helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:     client,
		rpcURL:     "https://fullnode.mainnet.sui.io:443",
		httpClient: http.DefaultClient,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("sui") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// JSON-RPC types.

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type transferSuiResult struct {
	TxBytes string `json:"txBytes"`
}

type executeResult struct {
	Digest string `json:"digest"`
}

// Transfer sends native SUI from a Privy wallet to a destination address.
// amount is in MIST (1 SUI = 1_000_000_000 MIST) as a decimal string.
// Returns the transaction digest.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet info from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("sui: get wallet: %w", err)
	}

	// Get a SUI coin object for gas
	coinObjectID, err := h.getGasCoin(ctx, wallet.Address)
	if err != nil {
		return "", fmt.Errorf("sui: get gas coin: %w", err)
	}

	// Build unsigned transaction via unsafe_transferSui
	txBytes, err := h.unsafeTransferSui(ctx, wallet.Address, coinObjectID, destination, amount)
	if err != nil {
		return "", fmt.Errorf("sui: build transaction: %w", err)
	}

	// Decode tx bytes from base64
	txBytesRaw, err := base64.StdEncoding.DecodeString(txBytes)
	if err != nil {
		return "", fmt.Errorf("sui: decode tx bytes: %w", err)
	}

	// Compute digest: Blake2b-256(intent_prefix + tx_bytes)
	// Intent prefix: [0x00, 0x00, 0x00] = TransactionData, V0, Sui
	intentMessage := append([]byte{0x00, 0x00, 0x00}, txBytesRaw...)
	digest := blake2b.Sum256(intentMessage)

	// Sign via Privy raw_sign
	digestHex := "0x" + hex.EncodeToString(digest[:])
	signResp, err := h.client.RawSign(ctx, walletID, digestHex)
	if err != nil {
		return "", fmt.Errorf("sui: sign transaction: %w", err)
	}

	// Format signature: flag (0x00 = Ed25519) + signature (64 bytes) + public_key (32 bytes)
	sigBytes, err := decodeHex(signResp.Data.Signature)
	if err != nil {
		return "", fmt.Errorf("sui: decode signature: %w", err)
	}

	pubKeyBytes, err := decodeHex(wallet.PublicKey)
	if err != nil {
		return "", fmt.Errorf("sui: decode public key: %w", err)
	}

	// Build serialized signature: scheme_flag + sig + pubkey
	serializedSig := make([]byte, 0, 1+len(sigBytes)+len(pubKeyBytes))
	serializedSig = append(serializedSig, 0x00) // Ed25519 flag
	serializedSig = append(serializedSig, sigBytes...)
	serializedSig = append(serializedSig, pubKeyBytes...)
	sigBase64 := base64.StdEncoding.EncodeToString(serializedSig)

	// Execute transaction
	txDigest, err := h.executeTransactionBlock(ctx, txBytes, sigBase64)
	if err != nil {
		return "", fmt.Errorf("sui: execute transaction: %w", err)
	}

	return txDigest, nil
}

// TransferObject transfers a Sui object to a destination address.
func (h *Helper) TransferObject(ctx context.Context, walletID string, objectID string, destination string) (string, error) {
	return "", fmt.Errorf("sui: TransferObject not yet implemented")
}

// getGasCoin gets the first available SUI coin object for gas.
func (h *Helper) getGasCoin(ctx context.Context, owner string) (string, error) {
	resp, err := h.callRPC(ctx, "suix_getCoins", []any{owner, "0x2::sui::SUI", nil, 1})
	if err != nil {
		return "", err
	}

	var result struct {
		Data []struct {
			CoinObjectID string `json:"coinObjectId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("no SUI coins found for %s", owner)
	}
	return result.Data[0].CoinObjectID, nil
}

// unsafeTransferSui builds an unsigned transfer transaction.
func (h *Helper) unsafeTransferSui(ctx context.Context, sender, coinObjectID, recipient, amount string) (string, error) {
	gasBudget := "10000000" // 0.01 SUI default gas budget
	resp, err := h.callRPC(ctx, "unsafe_transferSui", []any{
		sender, coinObjectID, gasBudget, recipient, amount,
	})
	if err != nil {
		return "", err
	}

	var result transferSuiResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.TxBytes, nil
}

// executeTransactionBlock submits a signed transaction for execution.
func (h *Helper) executeTransactionBlock(ctx context.Context, txBytes, signature string) (string, error) {
	options := map[string]bool{
		"showEffects": true,
	}
	resp, err := h.callRPC(ctx, "sui_executeTransactionBlock", []any{
		txBytes, []string{signature}, options, "WaitForLocalExecution",
	})
	if err != nil {
		return "", err
	}

	var result executeResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.Digest, nil
}

// callRPC makes a JSON-RPC call to the Sui node.
func (h *Helper) callRPC(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	reqBody := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.rpcURL, bytes.NewReader(body))
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

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	return hex.DecodeString(s)
}

// RawSign signs a pre-computed hash using the Sui wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
