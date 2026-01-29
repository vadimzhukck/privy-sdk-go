package privy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// WalletsService handles wallet-related operations.
type WalletsService struct {
	client *Client
}

// CreateWalletRequest represents a request to create a new wallet.
type CreateWalletRequest struct {
	ChainType         ChainType      `json:"chain_type"`
	Owner             *WalletOwner   `json:"owner,omitempty"`
	OwnerID           string         `json:"owner_id,omitempty"`
	PolicyIDs         []string       `json:"policy_ids,omitempty"`
	AdditionalSigners []string       `json:"additional_signers,omitempty"`
	IdempotencyKey    string         `json:"idempotency_key,omitempty"`
}

// UpdateWalletRequest represents a request to update a wallet.
type UpdateWalletRequest struct {
	PolicyIDs         []string `json:"policy_ids,omitempty"`
	AdditionalSigners []string `json:"additional_signers,omitempty"`
}

// ImportWalletInitRequest represents a request to initialize wallet import.
type ImportWalletInitRequest struct {
	ChainType ChainType    `json:"chain_type"`
	Owner     *WalletOwner `json:"owner,omitempty"`
	OwnerID   string       `json:"owner_id,omitempty"`
}

// ImportWalletInitResponse represents the response from initializing wallet import.
type ImportWalletInitResponse struct {
	ImportID  string `json:"import_id"`
	PublicKey string `json:"public_key"`
}

// ImportWalletSubmitRequest represents a request to complete wallet import.
type ImportWalletSubmitRequest struct {
	ImportID            string `json:"import_id"`
	EncryptedPrivateKey string `json:"encrypted_private_key"`
}

// ExportWalletResponse represents the response from exporting a wallet.
type ExportWalletResponse struct {
	PrivateKey string `json:"private_key"`
}

// Ethereum returns the Ethereum-specific wallet operations.
func (s *WalletsService) Ethereum() *EthereumWalletsService {
	return &EthereumWalletsService{client: s.client}
}

// Solana returns the Solana-specific wallet operations.
func (s *WalletsService) Solana() *SolanaWalletsService {
	return &SolanaWalletsService{client: s.client}
}

// Create creates a new wallet.
func (s *WalletsService) Create(ctx context.Context, req *CreateWalletRequest) (*Wallet, error) {
	u := fmt.Sprintf("%s/wallets", s.client.baseURL)

	var wallet Wallet
	if err := s.client.doRequest(ctx, "POST", u, req, &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

// Get retrieves a wallet by its ID.
func (s *WalletsService) Get(ctx context.Context, walletID string) (*Wallet, error) {
	u := fmt.Sprintf("%s/wallets/%s", s.client.baseURL, walletID)

	var wallet Wallet
	if err := s.client.doRequest(ctx, "GET", u, nil, &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

// WalletListOptions represents options for listing wallets.
type WalletListOptions struct {
	Cursor    string    // Pagination cursor
	Limit     int       // Maximum number of wallets per page
	UserID    string    // Filter by user ID (owner)
	ChainType ChainType // Filter by chain type (ethereum, solana, etc.)
}

// List lists all wallets with pagination and optional filters.
func (s *WalletsService) List(ctx context.Context, opts *WalletListOptions) (*PaginatedResponse[Wallet], error) {
	u := fmt.Sprintf("%s/wallets", s.client.baseURL)

	if opts != nil {
		params := url.Values{}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.UserID != "" {
			params.Set("user_id", opts.UserID)
		}
		if opts.ChainType != "" {
			params.Set("chain_type", string(opts.ChainType))
		}
		if len(params) > 0 {
			u = u + "?" + params.Encode()
		}
	}

	var resp PaginatedResponse[Wallet]
	if err := s.client.doRequest(ctx, "GET", u, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Update updates a wallet's policies or additional signers.
func (s *WalletsService) Update(ctx context.Context, walletID string, req *UpdateWalletRequest) (*Wallet, error) {
	u := fmt.Sprintf("%s/wallets/%s", s.client.baseURL, walletID)

	var wallet Wallet
	if err := s.client.doRequest(ctx, "PATCH", u, req, &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

// Export exports a wallet's private key.
func (s *WalletsService) Export(ctx context.Context, walletID string, signature string) (*ExportWalletResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/export", s.client.baseURL, walletID)

	var resp ExportWalletResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, nil, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetBalanceOptions represents options for getting wallet balance.
type GetBalanceOptions struct {
	Asset string // Asset symbol (e.g., "native", "ETH", "SOL")
	Chain string // Chain identifier (e.g., "ethereum", "solana")
}

// GetBalance retrieves the balance of a wallet.
// Either asset or chain must be provided.
func (s *WalletsService) GetBalance(ctx context.Context, walletID string, opts *GetBalanceOptions) (*WalletBalance, error) {
	u := fmt.Sprintf("%s/wallets/%s/balance", s.client.baseURL, walletID)

	if opts != nil {
		params := url.Values{}
		if opts.Asset != "" {
			params.Set("asset", opts.Asset)
		}
		if opts.Chain != "" {
			params.Set("chain", opts.Chain)
		}
		if len(params) > 0 {
			u = u + "?" + params.Encode()
		}
	}

	var balance WalletBalance
	if err := s.client.doRequest(ctx, "GET", u, nil, &balance); err != nil {
		return nil, err
	}

	return &balance, nil
}

// GetTransactions retrieves the transaction history for a wallet.
func (s *WalletsService) GetTransactions(ctx context.Context, walletID string, opts *ListOptions) (*PaginatedResponse[Transaction], error) {
	u := fmt.Sprintf("%s/wallets/%s/transactions", s.client.baseURL, walletID)

	if opts != nil {
		params := url.Values{}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if len(params) > 0 {
			u = u + "?" + params.Encode()
		}
	}

	var resp PaginatedResponse[Transaction]
	if err := s.client.doRequest(ctx, "GET", u, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// InitializeImport initializes the wallet import process.
func (s *WalletsService) InitializeImport(ctx context.Context, req *ImportWalletInitRequest) (*ImportWalletInitResponse, error) {
	u := fmt.Sprintf("%s/wallets/import/initialize", s.client.baseURL)

	var resp ImportWalletInitResponse
	if err := s.client.doRequest(ctx, "POST", u, req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SubmitImport completes the wallet import process.
func (s *WalletsService) SubmitImport(ctx context.Context, req *ImportWalletSubmitRequest) (*Wallet, error) {
	u := fmt.Sprintf("%s/wallets/import/submit", s.client.baseURL)

	var wallet Wallet
	if err := s.client.doRequest(ctx, "POST", u, req, &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

// GetTransaction retrieves a specific transaction by ID.
func (s *WalletsService) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	u := fmt.Sprintf("%s/transactions/%s", s.client.baseURL, transactionID)

	var tx Transaction
	if err := s.client.doRequest(ctx, "GET", u, nil, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}
