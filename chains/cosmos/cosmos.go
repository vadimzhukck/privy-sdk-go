// Package cosmos provides a high-level helper for Cosmos transactions
// using Privy's raw_sign endpoint and the Cosmos REST API.
//
// Transactions are built using lightweight protobuf encoding,
// SHA-256 hashed, signed via Privy, and broadcast to a Cosmos node.
package cosmos

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	privy "github.com/vadimzhukck/privy-sdk-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	basev1beta1 "cosmossdk.io/api/cosmos/base/v1beta1"
	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "cosmossdk.io/api/cosmos/tx/v1beta1"
)

// Helper provides high-level Cosmos transaction methods using Privy wallets.
type Helper struct {
	client     *privy.Client
	rpcURL     string
	chainID    string
	denom      string
	gasLimit   uint64
	feeAmount  string
	httpClient *http.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithRPCURL sets the Cosmos REST API endpoint URL.
func WithRPCURL(url string) Option {
	return func(h *Helper) {
		h.rpcURL = url
	}
}

// WithChainID sets the Cosmos chain ID (e.g. "cosmoshub-4").
func WithChainID(chainID string) Option {
	return func(h *Helper) {
		h.chainID = chainID
	}
}

// WithDenom sets the native token denomination (e.g. "uatom").
func WithDenom(denom string) Option {
	return func(h *Helper) {
		h.denom = denom
	}
}

// WithGasLimit sets the gas limit for transactions.
func WithGasLimit(gasLimit uint64) Option {
	return func(h *Helper) {
		h.gasLimit = gasLimit
	}
}

// WithFeeAmount sets the fee amount in the native denomination.
func WithFeeAmount(feeAmount string) Option {
	return func(h *Helper) {
		h.feeAmount = feeAmount
	}
}

// WithTestnet configures the helper for Cosmos Hub testnet (theta-testnet-001).
func WithTestnet() Option {
	return func(h *Helper) {
		h.rpcURL = "https://rest.cosmos.directory/theta-testnet-001"
		h.chainID = "theta-testnet-001"
	}
}

// WithHTTPClient sets a custom HTTP client for API calls.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Helper) {
		h.httpClient = c
	}
}

// NewHelper creates a new Cosmos helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:     client,
		rpcURL:     "https://rest.cosmos.directory/cosmoshub",
		chainID:    "cosmoshub-4",
		denom:      "uatom",
		gasLimit:   200000,
		feeAmount:  "5000",
		httpClient: http.DefaultClient,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("cosmos") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// accountInfo from the REST API.
type accountInfo struct {
	Account struct {
		AccountNumber string `json:"account_number"`
		Sequence      string `json:"sequence"`
	} `json:"account"`
}

// broadcastResult from the REST API.
type broadcastResult struct {
	TxResponse struct {
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
		RawLog string `json:"raw_log"`
	} `json:"tx_response"`
}

// Transfer sends native tokens from a Privy wallet to a destination address.
// amount is in the smallest denomination (e.g. uatom) as a decimal string.
// Returns the transaction hash.
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Get wallet info from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("cosmos: get wallet: %w", err)
	}

	// Query account info (sequence, account_number)
	acctInfo, err := h.queryAccount(ctx, wallet.Address)
	if err != nil {
		return "", fmt.Errorf("cosmos: query account: %w", err)
	}

	// Parse account number and sequence
	var accountNumber, sequence uint64
	fmt.Sscanf(acctInfo.Account.AccountNumber, "%d", &accountNumber)
	fmt.Sscanf(acctInfo.Account.Sequence, "%d", &sequence)

	// Build MsgSend
	msg := &bankv1beta1.MsgSend{
		FromAddress: wallet.Address,
		ToAddress:   destination,
		Amount: []*basev1beta1.Coin{
			{Denom: h.denom, Amount: amount},
		},
	}

	// Wrap in Any
	msgAny, err := anypb.New(msg)
	if err != nil {
		return "", fmt.Errorf("cosmos: wrap message: %w", err)
	}

	// Build TxBody
	txBody := &txv1beta1.TxBody{
		Messages: []*anypb.Any{msgAny},
	}
	bodyBytes, err := proto.Marshal(txBody)
	if err != nil {
		return "", fmt.Errorf("cosmos: marshal tx body: %w", err)
	}

	// Decode public key for AuthInfo
	pubKeyBytes, err := decodeHex(wallet.PublicKey)
	if err != nil {
		return "", fmt.Errorf("cosmos: decode public key: %w", err)
	}

	// Build AuthInfo
	pubKeyAny := &anypb.Any{
		TypeUrl: "/cosmos.crypto.secp256k1.PubKey",
		Value:   pubKeyBytes,
	}

	authInfo := &txv1beta1.AuthInfo{
		SignerInfos: []*txv1beta1.SignerInfo{
			{
				PublicKey: pubKeyAny,
				ModeInfo: &txv1beta1.ModeInfo{
					Sum: &txv1beta1.ModeInfo_Single_{
						Single: &txv1beta1.ModeInfo_Single{
							Mode: signingv1beta1.SignMode_SIGN_MODE_DIRECT,
						},
					},
				},
				Sequence: sequence,
			},
		},
		Fee: &txv1beta1.Fee{
			Amount: []*basev1beta1.Coin{
				{Denom: h.denom, Amount: h.feeAmount},
			},
			GasLimit: h.gasLimit,
		},
	}
	authInfoBytes, err := proto.Marshal(authInfo)
	if err != nil {
		return "", fmt.Errorf("cosmos: marshal auth info: %w", err)
	}

	// Build SignDoc
	signDoc := &txv1beta1.SignDoc{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authInfoBytes,
		ChainId:       h.chainID,
		AccountNumber: accountNumber,
	}
	signDocBytes, err := proto.Marshal(signDoc)
	if err != nil {
		return "", fmt.Errorf("cosmos: marshal sign doc: %w", err)
	}

	// SHA-256 hash
	hash := sha256.Sum256(signDocBytes)

	// Sign via Privy raw_sign
	hashHex := "0x" + hex.EncodeToString(hash[:])
	signResp, err := h.client.RawSign(ctx, walletID, hashHex)
	if err != nil {
		return "", fmt.Errorf("cosmos: sign transaction: %w", err)
	}

	// Decode signature (should be 64 bytes: R + S, no recovery ID)
	sigBytes, err := decodeHex(signResp.Data.Signature)
	if err != nil {
		return "", fmt.Errorf("cosmos: decode signature: %w", err)
	}
	// Strip recovery byte if present (65 → 64)
	if len(sigBytes) == 65 {
		sigBytes = sigBytes[:64]
	}

	// Build TxRaw
	txRaw := &txv1beta1.TxRaw{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authInfoBytes,
		Signatures:    [][]byte{sigBytes},
	}
	txBytes, err := proto.Marshal(txRaw)
	if err != nil {
		return "", fmt.Errorf("cosmos: marshal tx raw: %w", err)
	}

	// Broadcast
	txHash, err := h.broadcastTx(ctx, txBytes)
	if err != nil {
		return "", fmt.Errorf("cosmos: broadcast: %w", err)
	}

	return txHash, nil
}

// Delegate delegates tokens to a validator.
func (h *Helper) Delegate(ctx context.Context, walletID string, validatorAddr string, amount string) (string, error) {
	return "", fmt.Errorf("cosmos: Delegate not yet implemented")
}

// Undelegate undelegates tokens from a validator.
func (h *Helper) Undelegate(ctx context.Context, walletID string, validatorAddr string, amount string) (string, error) {
	return "", fmt.Errorf("cosmos: Undelegate not yet implemented")
}

// queryAccount queries account info from the Cosmos REST API.
func (h *Helper) queryAccount(ctx context.Context, address string) (*accountInfo, error) {
	url := fmt.Sprintf("%s/cosmos/auth/v1beta1/accounts/%s", h.rpcURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info accountInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// broadcastTx broadcasts a signed transaction via the Cosmos REST API.
func (h *Helper) broadcastTx(ctx context.Context, txBytes []byte) (string, error) {
	reqBody := map[string]any{
		"tx_bytes": base64.StdEncoding.EncodeToString(txBytes),
		"mode":     "BROADCAST_MODE_SYNC",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := h.rpcURL + "/cosmos/tx/v1beta1/txs"
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

	var result broadcastResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if result.TxResponse.Code != 0 {
		return "", fmt.Errorf("broadcast failed (code %d): %s", result.TxResponse.Code, result.TxResponse.RawLog)
	}

	return result.TxResponse.TxHash, nil
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	return hex.DecodeString(s)
}

// RawSign signs a pre-computed hash using the Cosmos wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
