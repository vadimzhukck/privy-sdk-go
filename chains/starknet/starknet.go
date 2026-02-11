// Package starknet provides a high-level helper for StarkNet transactions
// using Privy's raw_sign endpoint and the StarkNet JSON-RPC API.
//
// Transactions are built as INVOKE V1, hashed using Pedersen hash,
// signed via Privy, and submitted to a StarkNet node.
package starknet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/consensys/gnark-crypto/ecc/stark-curve/fp"
	pedersenhash "github.com/consensys/gnark-crypto/ecc/stark-curve/pedersen-hash"
	privy "github.com/vadimzhukck/privy-sdk-go"
)

// Well-known StarkNet constants.
var (
	// ETH contract address on StarkNet.
	ethContractAddress = hexToBigInt("0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7")

	// Selector for the "transfer" function: starknet_keccak("transfer").
	transferSelector = hexToBigInt("0x83afd3f4caedc6eebf44246fe54e38c95e3179a5ec9ea81740eca5b482d12e")

	// INVOKE transaction type prefix.
	invokePrefix = stringToFelt("invoke")

	// StarkNet Mainnet chain ID.
	mainnetChainID = stringToFelt("SN_MAIN")
)

// Helper provides high-level StarkNet transaction methods using Privy wallets.
type Helper struct {
	client     *privy.Client
	rpcURL     string
	chainID    *big.Int
	maxFee     *big.Int
	httpClient *http.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithRPCURL sets the StarkNet JSON-RPC endpoint URL.
func WithRPCURL(url string) Option {
	return func(h *Helper) {
		h.rpcURL = url
	}
}

// WithChainID sets the StarkNet chain ID as a felt (e.g., "SN_MAIN").
func WithChainID(chainID string) Option {
	return func(h *Helper) {
		h.chainID = stringToFelt(chainID)
	}
}

// WithMaxFee sets the maximum fee for transactions (in wei).
func WithMaxFee(maxFee *big.Int) Option {
	return func(h *Helper) {
		h.maxFee = maxFee
	}
}

// WithTestnet configures the helper for StarkNet Sepolia testnet.
func WithTestnet() Option {
	return func(h *Helper) {
		h.rpcURL = "https://starknet-sepolia.public.blastapi.io"
		h.chainID = stringToFelt("SN_SEPOLIA")
	}
}

// WithHTTPClient sets a custom HTTP client for RPC calls.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Helper) {
		h.httpClient = c
	}
}

// NewHelper creates a new StarkNet helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:     client,
		rpcURL:     "https://starknet-mainnet.public.blastapi.io",
		chainID:    mainnetChainID,
		maxFee:     big.NewInt(1e16), // 0.01 ETH default
		httpClient: http.DefaultClient,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("starknet") {
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
	Params  any    `json:"params"`
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

// Transfer sends native ETH on StarkNet from a Privy wallet to a destination.
// amount is in wei as a decimal string.
// Returns the transaction hash.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet info from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("starknet: get wallet: %w", err)
	}

	senderAddress := hexToBigInt(wallet.Address)

	// Parse amount as uint256 (low, high)
	amountBig := new(big.Int)
	if _, ok := amountBig.SetString(amount, 10); !ok {
		return "", fmt.Errorf("starknet: invalid amount %q", amount)
	}

	// Query nonce
	nonce, err := h.getNonce(ctx, wallet.Address)
	if err != nil {
		return "", fmt.Errorf("starknet: get nonce: %w", err)
	}

	// Build calldata for ETH transfer via account's __execute__
	calldata := buildETHTransferCalldata(destination, amountBig)

	// Compute calldata hash
	calldataHash := computeHashOnElements(calldata)

	// Compute INVOKE V1 transaction hash
	txHash := computeHashOnElements([]*big.Int{
		invokePrefix,
		big.NewInt(1), // version
		senderAddress,
		big.NewInt(0), // entry_point_selector (0 for invoke)
		calldataHash,
		h.maxFee,
		h.chainID,
		nonce,
	})

	// Sign via Privy raw_sign
	hashHex := "0x" + txHash.Text(16)
	signResp, err := h.client.RawSign(ctx, walletID, hashHex)
	if err != nil {
		return "", fmt.Errorf("starknet: sign transaction: %w", err)
	}

	// Parse signature (r, s)
	sigHex := strings.TrimPrefix(signResp.Data.Signature, "0x")
	if len(sigHex) < 128 {
		// Pad to 128 hex chars (64 bytes = r(32) + s(32))
		sigHex = strings.Repeat("0", 128-len(sigHex)) + sigHex
	}
	r := "0x" + sigHex[:64]
	s := "0x" + sigHex[64:128]

	// Build calldata hex strings for RPC
	calldataHex := make([]string, len(calldata))
	for i, v := range calldata {
		calldataHex[i] = "0x" + v.Text(16)
	}

	// Submit via RPC
	resultHash, err := h.addInvokeTransaction(ctx, wallet.Address, calldataHex, h.maxFee, nonce, r, s)
	if err != nil {
		return "", fmt.Errorf("starknet: submit transaction: %w", err)
	}

	return resultHash, nil
}

// buildETHTransferCalldata builds the calldata for an ETH transfer via __execute__.
// Uses the standard account contract format for INVOKE V1.
func buildETHTransferCalldata(recipient string, amount *big.Int) []*big.Int {
	recipientFelt := hexToBigInt(recipient)

	// Split amount into Uint256 (low, high)
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	amountLow := new(big.Int).And(amount, mask)
	amountHigh := new(big.Int).Rsh(amount, 128)

	return []*big.Int{
		big.NewInt(1),          // call_array_len
		ethContractAddress,     // to
		transferSelector,       // selector
		big.NewInt(0),          // data_offset
		big.NewInt(3),          // data_len
		big.NewInt(3),          // calldata_len
		recipientFelt,          // recipient
		amountLow,              // amount.low
		amountHigh,             // amount.high
	}
}

// getNonce queries the nonce for an account.
func (h *Helper) getNonce(ctx context.Context, address string) (*big.Int, error) {
	resp, err := h.callRPC(ctx, "starknet_getNonce", []any{
		"latest",
		address,
	})
	if err != nil {
		return nil, err
	}

	var nonceHex string
	if err := json.Unmarshal(resp, &nonceHex); err != nil {
		return nil, err
	}

	nonce := new(big.Int)
	nonceHex = strings.TrimPrefix(nonceHex, "0x")
	nonce.SetString(nonceHex, 16)
	return nonce, nil
}

// addInvokeTransaction submits a signed INVOKE V1 transaction.
func (h *Helper) addInvokeTransaction(ctx context.Context, senderAddr string, calldata []string, maxFee *big.Int, nonce *big.Int, r, s string) (string, error) {
	tx := map[string]any{
		"type":               "INVOKE",
		"sender_address":     senderAddr,
		"calldata":           calldata,
		"max_fee":            "0x" + maxFee.Text(16),
		"version":            "0x1",
		"signature":          []string{r, s},
		"nonce":              "0x" + nonce.Text(16),
	}

	resp, err := h.callRPC(ctx, "starknet_addInvokeTransaction", []any{tx})
	if err != nil {
		return "", err
	}

	var result struct {
		TransactionHash string `json:"transaction_hash"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	return result.TransactionHash, nil
}

// TransferERC20 sends an ERC-20 token on StarkNet.
func (h *Helper) TransferERC20(ctx context.Context, walletID string, tokenContract string, destination string, amount string) (string, error) {
	return "", fmt.Errorf("starknet: TransferERC20 not yet implemented")
}

// callRPC makes a JSON-RPC call to the StarkNet node.
func (h *Helper) callRPC(ctx context.Context, method string, params any) (json.RawMessage, error) {
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

// computeHashOnElements computes the Pedersen hash chain: h(h(...h(h(0, a0), a1)..., an), len).
// This matches StarkNet's standard hash-on-elements convention.
func computeHashOnElements(elements []*big.Int) *big.Int {
	elems := make([]*fp.Element, len(elements)+1)
	for i, e := range elements {
		var el fp.Element
		el.SetBigInt(e)
		elems[i] = &el
	}
	// Append length per StarkNet convention
	var lenEl fp.Element
	lenEl.SetUint64(uint64(len(elements)))
	elems[len(elements)] = &lenEl

	hash := pedersenhash.PedersenArray(elems...)
	var result big.Int
	hash.BigInt(&result)
	return &result
}

// hexToBigInt converts a hex string to *big.Int.
func hexToBigInt(s string) *big.Int {
	s = strings.TrimPrefix(s, "0x")
	n := new(big.Int)
	n.SetString(s, 16)
	return n
}

// stringToFelt converts a short ASCII string to a felt.
func stringToFelt(s string) *big.Int {
	result := big.NewInt(0)
	for _, c := range s {
		result.Lsh(result, 8)
		result.Or(result, big.NewInt(int64(c)))
	}
	return result
}

// RawSign signs a pre-computed hash using the StarkNet wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
