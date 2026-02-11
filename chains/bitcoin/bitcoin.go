// Package bitcoin provides a high-level helper for Bitcoin transactions
// using Privy's raw_sign endpoint, btcd for transaction building, and
// a block explorer API for UTXO management and broadcasting.
//
// Transactions use SegWit P2WPKH format with BIP143 signature hashing.
package bitcoin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	privy "github.com/vadimzhukck/privy-sdk-go"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// Helper provides high-level Bitcoin transaction methods using Privy wallets.
type Helper struct {
	client      *privy.Client
	explorerURL string
	network     string
	chainParams *chaincfg.Params
	feeRate     int64 // satoshis per vByte
	httpClient  *http.Client
}

// Option configures the Helper.
type Option func(*Helper)

// WithExplorerURL sets the block explorer API endpoint.
func WithExplorerURL(url string) Option {
	return func(h *Helper) {
		h.explorerURL = url
	}
}

// WithNetwork sets the Bitcoin network ("mainnet" or "testnet").
func WithNetwork(network string) Option {
	return func(h *Helper) {
		h.network = network
		switch network {
		case "testnet":
			h.chainParams = &chaincfg.TestNet3Params
		default:
			h.chainParams = &chaincfg.MainNetParams
		}
	}
}

// WithTestnet configures the helper for Bitcoin testnet3.
func WithTestnet() Option {
	return func(h *Helper) {
		h.network = "testnet"
		h.chainParams = &chaincfg.TestNet3Params
		h.explorerURL = "https://blockstream.info/testnet/api"
	}
}

// WithFeeRate sets the fee rate in satoshis per virtual byte.
func WithFeeRate(feeRate int64) Option {
	return func(h *Helper) {
		h.feeRate = feeRate
	}
}

// WithHTTPClient sets a custom HTTP client for API calls.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Helper) {
		h.httpClient = c
	}
}

// NewHelper creates a new Bitcoin helper.
// Options are applied in order: testnet defaults, client-level chain options, then direct options.
func NewHelper(client *privy.Client, opts ...Option) *Helper {
	h := &Helper{
		client:      client,
		explorerURL: "https://blockstream.info/api",
		network:     "mainnet",
		chainParams: &chaincfg.MainNetParams,
		feeRate:     10,
		httpClient:  http.DefaultClient,
	}
	if client.Testnet() {
		WithTestnet()(h)
	}
	for _, raw := range client.ChainOptions("bitcoin") {
		if o, ok := raw.(Option); ok {
			o(h)
		}
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// UTXO represents an unspent transaction output.
type UTXO struct {
	TxID  string `json:"txid"`
	Vout  uint32 `json:"vout"`
	Value int64  `json:"value"`
}

// Transfer sends BTC from a Privy wallet to a destination address.
// amount is in satoshis as a decimal string.
// Returns the transaction ID (hash).
func (h *Helper) Transfer(ctx context.Context, walletID string, destination string, amount string) (string, error) {
	// Parse amount
	amountSats, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("bitcoin: invalid amount %q: %w", amount, err)
	}

	// Get wallet info from Privy
	wallet, err := h.client.Wallets().Get(ctx, walletID)
	if err != nil {
		return "", fmt.Errorf("bitcoin: get wallet: %w", err)
	}

	// Decode compressed public key
	pubKeyBytes, err := decodeHex(wallet.PublicKey)
	if err != nil {
		return "", fmt.Errorf("bitcoin: decode public key: %w", err)
	}

	// Compute pubkey hash for P2WPKH
	pubKeyHash := btcutil.Hash160(pubKeyBytes)

	// Fetch UTXOs
	utxos, err := h.fetchUTXOs(ctx, wallet.Address)
	if err != nil {
		return "", fmt.Errorf("bitcoin: fetch utxos: %w", err)
	}
	if len(utxos) == 0 {
		return "", fmt.Errorf("bitcoin: no UTXOs found for %s", wallet.Address)
	}

	// Select UTXOs and calculate fee
	selectedUTXOs, totalInput, fee, err := h.selectUTXOs(utxos, amountSats)
	if err != nil {
		return "", fmt.Errorf("bitcoin: select utxos: %w", err)
	}

	// Build unsigned transaction
	tx, err := h.buildTx(selectedUTXOs, destination, wallet.Address, amountSats, totalInput, fee)
	if err != nil {
		return "", fmt.Errorf("bitcoin: build transaction: %w", err)
	}

	// Build PrevOutputFetcher for sighash computation
	prevOuts := make(map[wire.OutPoint]*wire.TxOut)
	p2wpkhScript, err := payToWitnessPubKeyHashScript(pubKeyHash)
	if err != nil {
		return "", fmt.Errorf("bitcoin: build p2wpkh script: %w", err)
	}
	for i, utxo := range selectedUTXOs {
		prevOuts[tx.TxIn[i].PreviousOutPoint] = wire.NewTxOut(utxo.Value, p2wpkhScript)
	}
	fetcher := txscript.NewMultiPrevOutFetcher(prevOuts)

	// Sign each input
	for i, utxo := range selectedUTXOs {
		sigHash, err := h.calcWitnessSigHash(tx, fetcher, i, utxo.Value, pubKeyHash)
		if err != nil {
			return "", fmt.Errorf("bitcoin: calc sighash for input %d: %w", i, err)
		}

		// Sign via Privy raw_sign
		hashHex := "0x" + hex.EncodeToString(sigHash)
		signResp, err := h.client.RawSign(ctx, walletID, hashHex)
		if err != nil {
			return "", fmt.Errorf("bitcoin: sign input %d: %w", i, err)
		}

		// Decode signature (64 bytes: R || S)
		sigBytes, err := decodeHex(signResp.Data.Signature)
		if err != nil {
			return "", fmt.Errorf("bitcoin: decode signature for input %d: %w", i, err)
		}
		if len(sigBytes) > 64 {
			sigBytes = sigBytes[:64] // Strip recovery byte if present
		}

		// Convert to DER format and append SIGHASH_ALL
		derSig := derEncodeSignature(sigBytes[:32], sigBytes[32:64])
		derSig = append(derSig, byte(txscript.SigHashAll))

		// Set witness: [signature, pubkey]
		tx.TxIn[i].Witness = wire.TxWitness{derSig, pubKeyBytes}
	}

	// Serialize transaction
	var txBuf bytes.Buffer
	if err := tx.Serialize(&txBuf); err != nil {
		return "", fmt.Errorf("bitcoin: serialize transaction: %w", err)
	}
	txHex := hex.EncodeToString(txBuf.Bytes())

	// Broadcast
	txID, err := h.broadcastTx(ctx, txHex)
	if err != nil {
		return "", fmt.Errorf("bitcoin: broadcast: %w", err)
	}

	return txID, nil
}

// fetchUTXOs retrieves unspent transaction outputs from the block explorer API.
func (h *Helper) fetchUTXOs(ctx context.Context, address string) ([]UTXO, error) {
	url := fmt.Sprintf("%s/address/%s/utxo", h.explorerURL, address)

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("explorer API returned %d: %s", resp.StatusCode, string(body))
	}

	var utxos []UTXO
	if err := json.Unmarshal(body, &utxos); err != nil {
		return nil, err
	}
	return utxos, nil
}

// selectUTXOs selects UTXOs to cover the amount plus estimated fee.
func (h *Helper) selectUTXOs(utxos []UTXO, amount int64) (selected []UTXO, totalInput int64, fee int64, err error) {
	for _, utxo := range utxos {
		selected = append(selected, utxo)
		totalInput += utxo.Value

		// Estimate fee: ~11 overhead + 68 per input + 31 per output (2 outputs: payment + change)
		vSize := int64(11 + 68*len(selected) + 31*2)
		fee = vSize * h.feeRate

		if totalInput >= amount+fee {
			return selected, totalInput, fee, nil
		}
	}
	return nil, 0, 0, fmt.Errorf("insufficient funds: need %d sats, have %d", amount+fee, totalInput)
}

// buildTx creates an unsigned transaction.
func (h *Helper) buildTx(utxos []UTXO, destination, changeAddr string, amount, totalInput, fee int64) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(wire.TxVersion)

	// Add inputs
	for _, utxo := range utxos {
		hash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return nil, fmt.Errorf("invalid txid %s: %w", utxo.TxID, err)
		}
		outPoint := wire.NewOutPoint(hash, utxo.Vout)
		tx.AddTxIn(wire.NewTxIn(outPoint, nil, nil))
	}

	// Payment output
	destAddr, err := btcutil.DecodeAddress(destination, h.chainParams)
	if err != nil {
		return nil, fmt.Errorf("invalid destination address: %w", err)
	}
	destScript, err := txscript.PayToAddrScript(destAddr)
	if err != nil {
		return nil, err
	}
	tx.AddTxOut(wire.NewTxOut(amount, destScript))

	// Change output (if there's change)
	change := totalInput - amount - fee
	if change > 546 { // Dust threshold
		chgAddr, err := btcutil.DecodeAddress(changeAddr, h.chainParams)
		if err != nil {
			return nil, fmt.Errorf("invalid change address: %w", err)
		}
		chgScript, err := txscript.PayToAddrScript(chgAddr)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire.NewTxOut(change, chgScript))
	}

	return tx, nil
}

// calcWitnessSigHash computes the BIP143 witness sighash for a P2WPKH input.
func (h *Helper) calcWitnessSigHash(tx *wire.MsgTx, fetcher txscript.PrevOutputFetcher, idx int, inputAmount int64, pubKeyHash []byte) ([]byte, error) {
	// P2WPKH script code: OP_DUP OP_HASH160 <pubKeyHash> OP_EQUALVERIFY OP_CHECKSIG
	scriptCode, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_DUP).
		AddOp(txscript.OP_HASH160).
		AddData(pubKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).
		AddOp(txscript.OP_CHECKSIG).
		Script()
	if err != nil {
		return nil, err
	}

	sigHashes := txscript.NewTxSigHashes(tx, fetcher)

	return txscript.CalcWitnessSigHash(scriptCode, sigHashes, txscript.SigHashAll, tx, idx, inputAmount)
}

// broadcastTx broadcasts a raw transaction hex via the block explorer API.
func (h *Helper) broadcastTx(ctx context.Context, txHex string) (string, error) {
	url := h.explorerURL + "/tx"

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(txHex))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("broadcast failed (%d): %s", resp.StatusCode, string(body))
	}

	return strings.TrimSpace(string(body)), nil
}

// payToWitnessPubKeyHashScript creates a P2WPKH output script.
func payToWitnessPubKeyHashScript(pubKeyHash []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(pubKeyHash).
		Script()
}

// derEncodeSignature converts R, S values to DER format.
func derEncodeSignature(r, s []byte) []byte {
	rEnc := canonicalizeInt(r)
	sEnc := canonicalizeInt(s)

	totalLen := 2 + len(rEnc) + 2 + len(sEnc)
	der := make([]byte, 0, 2+totalLen)
	der = append(der, 0x30, byte(totalLen))
	der = append(der, 0x02, byte(len(rEnc)))
	der = append(der, rEnc...)
	der = append(der, 0x02, byte(len(sEnc)))
	der = append(der, sEnc...)
	return der
}

// canonicalizeInt strips leading zeros and adds padding for DER encoding.
func canonicalizeInt(b []byte) []byte {
	// Make a copy to avoid modifying original
	v := make([]byte, len(b))
	copy(v, b)

	// Strip leading zeros
	for len(v) > 1 && v[0] == 0 {
		v = v[1:]
	}
	// Add leading zero if high bit set (to keep positive)
	if len(v) > 0 && v[0]&0x80 != 0 {
		v = append([]byte{0}, v...)
	}
	return v
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	return hex.DecodeString(s)
}

// RawSign signs a pre-computed hash using the Bitcoin wallet's key via Privy.
func (h *Helper) RawSign(ctx context.Context, walletID string, hash string) (*privy.RawSignResponse, error) {
	return h.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function via Privy.
func (h *Helper) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*privy.RawSignResponse, error) {
	return h.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
