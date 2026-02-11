package privy

import (
	"context"
	"encoding/base64"
	"fmt"
)

// SolanaWalletsService handles Solana-specific wallet operations.
type SolanaWalletsService struct {
	client *Client
}

// SolanaSignAndSendTransactionRequest represents the params for Solana signAndSendTransaction.
type SolanaSignAndSendTransactionRequest struct {
	Transaction string `json:"transaction"` // Base64 encoded transaction
}

// SolanaSignTransactionRequest represents the params for Solana signTransaction.
type SolanaSignTransactionRequest struct {
	Transaction string `json:"transaction"` // Base64 encoded transaction
}

// SolanaSignMessageRequest represents the params for Solana signMessage.
type SolanaSignMessageRequest struct {
	Message  string `json:"message"`  // Base64 encoded message
	Encoding string `json:"encoding"` // Must be "base64"
}

// SignAndSendTransaction signs and sends a Solana transaction.
func (s *SolanaWalletsService) SignAndSendTransaction(ctx context.Context, walletID string, transaction string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method: "signAndSendTransaction",
		CAIP2:  "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp", // Mainnet
		Params: &SolanaSignAndSendTransactionRequest{
			Transaction: transaction,
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignAndSendTransactionOnDevnet signs and sends a Solana transaction on devnet.
func (s *SolanaWalletsService) SignAndSendTransactionOnDevnet(ctx context.Context, walletID string, transaction string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method: "signAndSendTransaction",
		CAIP2:  "solana:EtWTRABZaYq6iMfeYKouRu166VU2xqa1", // Devnet
		Params: &SolanaSignAndSendTransactionRequest{
			Transaction: transaction,
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignTransaction signs a Solana transaction without sending.
func (s *SolanaWalletsService) SignTransaction(ctx context.Context, walletID string, transaction string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method: "signTransaction",
		Params: &SolanaSignTransactionRequest{
			Transaction: transaction,
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignMessage signs a message using the Solana wallet.
// The message will be base64 encoded before sending.
func (s *SolanaWalletsService) SignMessage(ctx context.Context, walletID string, message string, encoding string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	// Base64 encode the message if it's not already
	encodedMessage := message
	if encoding == "" || encoding == "utf-8" {
		encodedMessage = base64.StdEncoding.EncodeToString([]byte(message))
	}

	req := &RPCRequest{
		Method: "signMessage",
		Params: &SolanaSignMessageRequest{
			Message:  encodedMessage,
			Encoding: "base64",
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignAndSendTransactionWithCAIP2 signs and sends a Solana transaction with a custom CAIP-2 identifier.
func (s *SolanaWalletsService) SignAndSendTransactionWithCAIP2(ctx context.Context, walletID string, transaction string, caip2 string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method: "signAndSendTransaction",
		CAIP2:  caip2,
		Params: &SolanaSignAndSendTransactionRequest{
			Transaction: transaction,
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}
