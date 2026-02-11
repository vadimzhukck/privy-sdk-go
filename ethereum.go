package privy

import (
	"context"
	"fmt"
)

// EthereumWalletsService handles Ethereum-specific wallet operations.
type EthereumWalletsService struct {
	client *Client
}

// RPCRequest represents an RPC request to a wallet.
type RPCRequest struct {
	Method    string `json:"method"`
	ChainType string `json:"chain_type,omitempty"`
	CAIP2     string `json:"caip2,omitempty"`
	Params    any    `json:"params"`
	Sponsor   bool   `json:"sponsor,omitempty"`
}

// SendTransactionRequest represents the params for eth_sendTransaction.
type SendTransactionRequest struct {
	Transaction *EthereumTransaction `json:"transaction"`
}

// SignTransactionRequest represents the params for eth_signTransaction.
type SignTransactionRequest struct {
	Transaction *EthereumTransaction `json:"transaction"`
}

// SignMessageRequest represents the params for personal_sign.
type SignMessageRequest struct {
	Message  string `json:"message"`
	Encoding string `json:"encoding"` // "utf-8" or "hex"
}

// SignTypedDataRequest represents the params for eth_signTypedData_v4.
type SignTypedDataRequest struct {
	TypedData *TypedData `json:"typed_data"`
}

// SignHashRequest represents the params for secp256k1_sign.
type SignHashRequest struct {
	Hash     string `json:"hash"`
	Encoding string `json:"encoding,omitempty"` // "hex"
}

// SignUserOperationRequest represents the params for eth_signUserOperation.
type SignUserOperationRequest struct {
	UserOperation map[string]any `json:"user_operation"`
	EntryPoint    string         `json:"entry_point"`
	ChainID       int64          `json:"chain_id"`
}

// Sign7702AuthorizationRequest represents the params for eth_sign7702Authorization.
type Sign7702AuthorizationRequest struct {
	ChainID         int64  `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	Nonce           int64  `json:"nonce,omitempty"`
}

// SendTransaction sends an Ethereum transaction.
func (s *EthereumWalletsService) SendTransaction(ctx context.Context, walletID string, tx *EthereumTransaction, chainID int64, sponsor bool, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	caip2 := fmt.Sprintf("eip155:%d", chainID)
	req := &RPCRequest{
		Method:    "eth_sendTransaction",
		CAIP2:     caip2,
		ChainType: "ethereum",
		Params:    &SendTransactionRequest{Transaction: tx},
		Sponsor:   sponsor,
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignTransaction signs an Ethereum transaction without broadcasting.
func (s *EthereumWalletsService) SignTransaction(ctx context.Context, walletID string, tx *EthereumTransaction, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method:    "eth_signTransaction",
		ChainType: "ethereum",
		Params:    &SignTransactionRequest{Transaction: tx},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignMessage signs a message using personal_sign.
func (s *EthereumWalletsService) SignMessage(ctx context.Context, walletID string, message string, encoding string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if encoding == "" {
		encoding = "utf-8"
	}

	req := &RPCRequest{
		Method:    "personal_sign",
		ChainType: "ethereum",
		Params: &SignMessageRequest{
			Message:  message,
			Encoding: encoding,
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignTypedData signs typed data using EIP-712.
func (s *EthereumWalletsService) SignTypedData(ctx context.Context, walletID string, typedData *TypedData, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method:    "eth_signTypedData_v4",
		ChainType: "ethereum",
		Params:    &SignTypedDataRequest{TypedData: typedData},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignHash signs a raw hash using secp256k1.
func (s *EthereumWalletsService) SignHash(ctx context.Context, walletID string, hash string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method:    "secp256k1_sign",
		ChainType: "ethereum",
		Params: &SignHashRequest{
			Hash:     hash,
			Encoding: "hex",
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignUserOperation signs an ERC-4337 user operation.
func (s *EthereumWalletsService) SignUserOperation(ctx context.Context, walletID string, userOp map[string]any, entryPoint string, chainID int64, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method:    "eth_signUserOperation",
		ChainType: "ethereum",
		Params: &SignUserOperationRequest{
			UserOperation: userOp,
			EntryPoint:    entryPoint,
			ChainID:       chainID,
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Sign7702Authorization signs an EIP-7702 authorization.
func (s *EthereumWalletsService) Sign7702Authorization(ctx context.Context, walletID string, chainID int64, contractAddress string, nonce int64, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method:    "eth_sign7702Authorization",
		ChainType: "ethereum",
		Params: &Sign7702AuthorizationRequest{
			ChainID:         chainID,
			ContractAddress: contractAddress,
			Nonce:           nonce,
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RawSign signs raw data using the wallet's key.
func (s *EthereumWalletsService) RawSign(ctx context.Context, walletID string, hash string, signature string) (*SignatureResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	req := &RPCRequest{
		Method:    "raw_sign",
		ChainType: "ethereum",
		Params: map[string]string{
			"hash":     hash,
			"encoding": "hex",
		},
	}

	var resp SignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}
