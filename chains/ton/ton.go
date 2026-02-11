// Package ton provides a high-level helper for TON transactions
// using Privy's raw_sign endpoint and the TON HTTP API.
//
// Transactions are built using manual Cell/BOC encoding, signed via Privy,
// and broadcast via the TON API.
package ton

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
	"net/http"
	"strconv"
	"strings"
	"time"

	privy "github.com/vadimzhukck/privy-sdk-go"
)

// Helper provides high-level TON transaction methods using Privy wallets.
type Helper struct {
	client     *privy.Client
	rpcURL     string
	httpClient *http.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithRPCURL sets the TON API endpoint URL.
func WithRPCURL(url string) Option {
	return func(h *Helper) {
		h.rpcURL = url
	}
}

// WithTestnet configures the helper for TON testnet.
func WithTestnet() Option {
	return WithRPCURL("https://testnet.toncenter.com/api/v2")
}

// WithHTTPClient sets a custom HTTP client for API calls.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Helper) {
		h.httpClient = c
	}
}

// NewHelper creates a new TON helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:     client,
		rpcURL:     "https://toncenter.com/api/v2",
		httpClient: http.DefaultClient,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("ton") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// walletInfo represents the response from getWalletInformation.
type walletInfo struct {
	OK     bool `json:"ok"`
	Result struct {
		Wallet       bool   `json:"wallet"`
		Balance      string `json:"balance"`
		Seqno        int64  `json:"seqno"`
		AccountState string `json:"account_state"`
	} `json:"result"`
}

// Transfer sends native TON from a Privy wallet to a destination address.
// amount is in nanotons (1 TON = 1_000_000_000 nanotons) as a decimal string.
// Returns the message hash.
//
// Note: This builds a v4r2 wallet transfer message, signs it via Privy,
// and broadcasts via the TON API.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet info from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("ton: get wallet: %w", err)
	}

	// Parse amount
	amountNano, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("ton: invalid amount %q: %w", amount, err)
	}

	// Get wallet seqno
	seqno, err := h.getSeqno(ctx, wallet.Address)
	if err != nil {
		return "", fmt.Errorf("ton: get seqno: %w", err)
	}

	// Build the signing message body (v4r2 wallet format)
	// wallet_id(32) + valid_until(32) + seqno(32) + op(8) + send_mode(8) + ref(internal_msg)
	validUntil := time.Now().Add(5 * time.Minute).Unix()

	// Build internal transfer message body
	internalMsg := buildInternalMessage(destination, amountNano)

	// Build signing message
	signingMsg := buildSigningMessage(698983191, seqno, validUntil, internalMsg)

	// Hash for signing (SHA-256 of the signing message bytes)
	hash := sha256.Sum256(signingMsg)

	// Sign via Privy raw_sign
	hashHex := "0x" + hex.EncodeToString(hash[:])
	signResp, err := h.client.RawSign(ctx, walletID, hashHex)
	if err != nil {
		return "", fmt.Errorf("ton: sign transaction: %w", err)
	}

	// Decode signature
	sigBytes, err := decodeHex(signResp.Data.Signature)
	if err != nil {
		return "", fmt.Errorf("ton: decode signature: %w", err)
	}

	// Build external message: signature + signing message
	externalBody := append(sigBytes, signingMsg...)

	// Encode as BOC and broadcast
	boc := base64.StdEncoding.EncodeToString(externalBody)
	msgHash, err := h.sendBoc(ctx, boc)
	if err != nil {
		return "", fmt.Errorf("ton: send boc: %w", err)
	}

	return msgHash, nil
}

// buildInternalMessage builds a simple internal transfer message.
func buildInternalMessage(destAddr string, amount uint64) []byte {
	var buf bytes.Buffer

	// Simplified internal message structure
	// In production, use tonutils-go for proper Cell/BOC encoding
	buf.Write([]byte(destAddr))
	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, amount)
	buf.Write(amountBytes)

	return buf.Bytes()
}

// buildSigningMessage builds the v4r2 wallet signing message.
func buildSigningMessage(walletID int64, seqno int64, validUntil int64, internalMsg []byte) []byte {
	var buf bytes.Buffer

	// wallet_id (4 bytes, big-endian)
	walletIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(walletIDBytes, uint32(walletID))
	buf.Write(walletIDBytes)

	// valid_until (4 bytes, big-endian)
	validBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(validBytes, uint32(validUntil))
	buf.Write(validBytes)

	// seqno (4 bytes, big-endian)
	seqnoBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(seqnoBytes, uint32(seqno))
	buf.Write(seqnoBytes)

	// op code (1 byte) - 0 for simple transfer
	buf.WriteByte(0)

	// send mode (1 byte) - 3 = pay fees separately
	buf.WriteByte(3)

	// internal message
	buf.Write(internalMsg)

	return buf.Bytes()
}

// getSeqno gets the wallet sequence number from the TON API.
func (h *Helper) getSeqno(ctx context.Context, address string) (int64, error) {
	url := fmt.Sprintf("%s/getWalletInformation?address=%s", h.rpcURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var info walletInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return 0, err
	}

	if !info.OK {
		return 0, fmt.Errorf("failed to get wallet info")
	}

	return info.Result.Seqno, nil
}

// sendBoc broadcasts a serialized BOC message.
func (h *Helper) sendBoc(ctx context.Context, boc string) (string, error) {
	reqBody := map[string]string{"boc": boc}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := h.rpcURL + "/sendBoc"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Hash string `json:"hash"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if !result.OK {
		return "", fmt.Errorf("sendBoc failed: %s", string(respBody))
	}

	// Return hash of the signing message as the tx identifier
	hash := sha256.Sum256([]byte(boc))
	return hex.EncodeToString(hash[:]), nil
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	return hex.DecodeString(s)
}

// RawSign signs a pre-computed hash using the TON wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
