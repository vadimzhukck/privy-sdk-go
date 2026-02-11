// Package near provides a high-level helper for NEAR transactions
// using Privy's raw_sign endpoint and NEAR JSON-RPC.
//
// Transactions are serialized using manual Borsh encoding, SHA-256 hashed,
// signed via Privy, and broadcast to a NEAR RPC node.
package near

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

// Helper provides high-level NEAR transaction methods using Privy wallets.
type Helper struct {
	client     *privy.Client
	rpcURL     string
	httpClient *http.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithRPCURL sets the NEAR JSON-RPC endpoint URL.
func WithRPCURL(url string) Option {
	return func(h *Helper) {
		h.rpcURL = url
	}
}

// WithTestnet configures the helper for NEAR testnet.
func WithTestnet() Option {
	return WithRPCURL("https://rpc.testnet.near.org")
}

// WithHTTPClient sets a custom HTTP client for RPC calls.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Helper) {
		h.httpClient = c
	}
}

// NewHelper creates a new NEAR helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:     client,
		rpcURL:     "https://rpc.mainnet.near.org",
		httpClient: http.DefaultClient,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("near") {
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
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type accessKeyResult struct {
	Nonce     uint64 `json:"nonce"`
	BlockHash string `json:"block_hash"`
}

type blockResult struct {
	Header struct {
		Hash string `json:"hash"`
	} `json:"header"`
}

type sendTxResult struct {
	Transaction struct {
		Hash string `json:"hash"`
	} `json:"transaction"`
}

// Transfer sends native NEAR from a Privy wallet to a destination account.
// amount is in yoctoNEAR (1 NEAR = 10^24 yoctoNEAR) as a decimal string.
// Returns the transaction hash.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet info from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("near: get wallet: %w", err)
	}

	// Parse public key from wallet
	pubKeyBytes, err := decodeHex(wallet.PublicKey)
	if err != nil {
		return "", fmt.Errorf("near: decode public key: %w", err)
	}
	if len(pubKeyBytes) != 32 {
		return "", fmt.Errorf("near: expected 32-byte public key, got %d bytes", len(pubKeyBytes))
	}

	// Format public key as ed25519:<base58> for RPC query
	pubKeyB58 := "ed25519:" + base58Encode(pubKeyBytes)

	// Query access key nonce
	accessKey, err := h.queryAccessKey(ctx, wallet.Address, pubKeyB58)
	if err != nil {
		return "", fmt.Errorf("near: query access key: %w", err)
	}

	// Query recent block hash
	block, err := h.queryBlock(ctx)
	if err != nil {
		return "", fmt.Errorf("near: query block: %w", err)
	}

	blockHash, err := base58Decode(block.Header.Hash)
	if err != nil {
		return "", fmt.Errorf("near: decode block hash: %w", err)
	}

	// Parse amount as u128
	deposit := new(big.Int)
	if _, ok := deposit.SetString(amount, 10); !ok {
		return "", fmt.Errorf("near: invalid amount %q", amount)
	}

	// Borsh-serialize the transaction manually
	serialized := serializeTransaction(wallet.Address, pubKeyBytes, accessKey.Nonce+1, destination, blockHash, deposit)

	// SHA-256 hash
	hash := sha256.Sum256(serialized)

	// Sign via Privy raw_sign
	hashHex := "0x" + hex.EncodeToString(hash[:])
	signResp, err := h.client.RawSign(ctx, walletID, hashHex)
	if err != nil {
		return "", fmt.Errorf("near: sign transaction: %w", err)
	}

	// Decode signature
	sigBytes, err := decodeHex(signResp.Data.Signature)
	if err != nil {
		return "", fmt.Errorf("near: decode signature: %w", err)
	}

	// Build signed transaction: transaction bytes + signature (key_type + 64 bytes)
	var signedBuf bytes.Buffer
	signedBuf.Write(serialized)
	signedBuf.WriteByte(0) // Ed25519 key type
	if len(sigBytes) >= 64 {
		signedBuf.Write(sigBytes[:64])
	} else {
		// Pad to 64 bytes
		padded := make([]byte, 64)
		copy(padded, sigBytes)
		signedBuf.Write(padded)
	}

	// Broadcast
	txHash, err := h.broadcastTx(ctx, signedBuf.Bytes())
	if err != nil {
		return "", fmt.Errorf("near: broadcast transaction: %w", err)
	}

	return txHash, nil
}

// serializeTransaction manually Borsh-serializes a NEAR transaction with a single Transfer action.
func serializeTransaction(signerID string, pubKey []byte, nonce uint64, receiverID string, blockHash []byte, deposit *big.Int) []byte {
	var buf bytes.Buffer

	// SignerID: 4-byte length (LE) + string bytes
	borshWriteString(&buf, signerID)

	// PublicKey: 1 byte key_type (0 = Ed25519) + 32 bytes data
	buf.WriteByte(0)
	if len(pubKey) >= 32 {
		buf.Write(pubKey[:32])
	}

	// Nonce: 8 bytes little-endian
	borshWriteU64(&buf, nonce)

	// ReceiverID: 4-byte length (LE) + string bytes
	borshWriteString(&buf, receiverID)

	// BlockHash: 32 bytes
	if len(blockHash) >= 32 {
		buf.Write(blockHash[:32])
	} else {
		padded := make([]byte, 32)
		copy(padded, blockHash)
		buf.Write(padded)
	}

	// Actions: 4-byte length (LE) = 1 action
	borshWriteU32(&buf, 1)

	// Action enum: 1 byte action type = 3 (Transfer)
	buf.WriteByte(3)

	// Transfer: deposit as u128 (16 bytes, little-endian)
	borshWriteU128(&buf, deposit)

	return buf.Bytes()
}

func borshWriteString(buf *bytes.Buffer, s string) {
	borshWriteU32(buf, uint32(len(s)))
	buf.WriteString(s)
}

func borshWriteU32(buf *bytes.Buffer, v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	buf.Write(b)
}

func borshWriteU64(buf *bytes.Buffer, v uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	buf.Write(b)
}

func borshWriteU128(buf *bytes.Buffer, v *big.Int) {
	// u128 little-endian: 16 bytes
	b := make([]byte, 16)
	if v != nil && v.Sign() > 0 {
		vBytes := v.Bytes() // big-endian
		// Copy in reverse to get little-endian
		for i, j := 0, len(vBytes)-1; j >= 0 && i < 16; i, j = i+1, j-1 {
			b[i] = vBytes[j]
		}
	}
	buf.Write(b)
}

// queryAccessKey queries the access key for the given account and public key.
func (h *Helper) queryAccessKey(ctx context.Context, accountID, pubKey string) (*accessKeyResult, error) {
	resp, err := h.callRPC(ctx, "query", map[string]any{
		"request_type": "view_access_key",
		"finality":     "final",
		"account_id":   accountID,
		"public_key":   pubKey,
	})
	if err != nil {
		return nil, err
	}

	var result accessKeyResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// queryBlock queries the latest finalized block.
func (h *Helper) queryBlock(ctx context.Context) (*blockResult, error) {
	resp, err := h.callRPC(ctx, "block", map[string]any{
		"finality": "final",
	})
	if err != nil {
		return nil, err
	}

	var result blockResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// broadcastTx broadcasts a signed transaction.
func (h *Helper) broadcastTx(ctx context.Context, signedTxBytes []byte) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(signedTxBytes)

	resp, err := h.callRPC(ctx, "broadcast_tx_commit", []string{encoded})
	if err != nil {
		return "", err
	}

	var result sendTxResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.Transaction.Hash, nil
}

// callRPC makes a JSON-RPC call to the NEAR node.
func (h *Helper) callRPC(ctx context.Context, method string, params any) (json.RawMessage, error) {
	reqBody := &jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      "privy",
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

// decodeHex decodes a hex string, stripping optional 0x prefix.
func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	return hex.DecodeString(s)
}

// base58Encode encodes bytes to base58 (NEAR uses Bitcoin-style base58).
func base58Encode(data []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	bi := new(big.Int).SetBytes(data)
	var result []byte
	mod := new(big.Int)
	zero := big.NewInt(0)
	base := big.NewInt(58)

	for bi.Cmp(zero) > 0 {
		bi.DivMod(bi, base, mod)
		result = append([]byte{alphabet[mod.Int64()]}, result...)
	}

	// Leading zeros
	for _, b := range data {
		if b != 0 {
			break
		}
		result = append([]byte{alphabet[0]}, result...)
	}

	return string(result)
}

// base58Decode decodes a base58 string to bytes.
func base58Decode(s string) ([]byte, error) {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	result := big.NewInt(0)
	base := big.NewInt(58)

	for _, c := range s {
		idx := strings.IndexRune(alphabet, c)
		if idx < 0 {
			return nil, fmt.Errorf("invalid base58 character: %c", c)
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(idx)))
	}

	// Convert to bytes
	resultBytes := result.Bytes()

	// Count leading '1's (zeros)
	numLeadingZeros := 0
	for _, c := range s {
		if c != '1' {
			break
		}
		numLeadingZeros++
	}

	// Prepend zero bytes
	if numLeadingZeros > 0 {
		padding := make([]byte, numLeadingZeros)
		resultBytes = append(padding, resultBytes...)
	}

	return resultBytes, nil
}

// RawSign signs a pre-computed hash using the NEAR wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
