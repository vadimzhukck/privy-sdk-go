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

// Stellar returns the Stellar-specific wallet operations.
func (s *WalletsService) Stellar() *StellarWalletsService {
	return &StellarWalletsService{client: s.client}
}

// Cosmos returns the Cosmos-specific wallet operations.
func (s *WalletsService) Cosmos() *CosmosWalletsService {
	return &CosmosWalletsService{client: s.client}
}

// Sui returns the Sui-specific wallet operations.
func (s *WalletsService) Sui() *SuiWalletsService {
	return &SuiWalletsService{client: s.client}
}

// Tron returns the Tron-specific wallet operations.
func (s *WalletsService) Tron() *TronWalletsService {
	return &TronWalletsService{client: s.client}
}

// Bitcoin returns the Bitcoin SegWit-specific wallet operations.
func (s *WalletsService) Bitcoin() *BitcoinWalletsService {
	return &BitcoinWalletsService{client: s.client}
}

// Near returns the NEAR-specific wallet operations.
func (s *WalletsService) Near() *NearWalletsService {
	return &NearWalletsService{client: s.client}
}

// Ton returns the TON-specific wallet operations.
func (s *WalletsService) Ton() *TonWalletsService {
	return &TonWalletsService{client: s.client}
}

// Starknet returns the Starknet-specific wallet operations.
func (s *WalletsService) Starknet() *StarknetWalletsService {
	return &StarknetWalletsService{client: s.client}
}

// Aptos returns the Aptos-specific wallet operations.
func (s *WalletsService) Aptos() *AptosWalletsService {
	return &AptosWalletsService{client: s.client}
}

// Spark returns the Spark (Bitcoin Lightning)-specific wallet operations.
func (s *WalletsService) Spark() *SparkWalletsService {
	return &SparkWalletsService{client: s.client}
}

// Create creates a new wallet.
func (s *WalletsService) Create(ctx context.Context, req *CreateWalletRequest) (*Wallet, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets", s.client.baseURL)

	var wallet Wallet
	if err := s.client.doRequest(ctx, "POST", u, req, &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

// Get retrieves a wallet by its ID.
func (s *WalletsService) Get(ctx context.Context, walletID string) (*Wallet, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
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
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
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
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s", s.client.baseURL, walletID)

	var wallet Wallet
	if err := s.client.doRequest(ctx, "PATCH", u, req, &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

// Export exports a wallet's private key.
func (s *WalletsService) Export(ctx context.Context, walletID string, signature string) (*ExportWalletResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
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
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
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

// GetTransactionsOptions represents options for getting wallet transactions.
type GetTransactionsOptions struct {
	Chain  string   // Required: blockchain network (ethereum, arbitrum, base, solana, etc.)
	Asset  []string // Required: token types (usdc, eth, sol, etc.) - max 4 assets
	TxHash string   // Optional: filter by specific transaction hash
	Cursor string   // Optional: pagination cursor
	Limit  int      // Optional: maximum number of transactions per page (max 100)
}

// GetTransactions retrieves the transaction history for a wallet.
// Chain and Asset parameters are required by the Privy API.
func (s *WalletsService) GetTransactions(ctx context.Context, walletID string, opts *GetTransactionsOptions) (*PaginatedResponse[Transaction], error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/transactions", s.client.baseURL, walletID)

	if opts == nil {
		return nil, fmt.Errorf("GetTransactionsOptions is required (chain and asset parameters are mandatory)")
	}

	params := url.Values{}

	// Required parameters
	if opts.Chain == "" {
		return nil, fmt.Errorf("chain parameter is required")
	}
	params.Set("chain", opts.Chain)

	if len(opts.Asset) == 0 {
		return nil, fmt.Errorf("asset parameter is required (at least one asset)")
	}
	if len(opts.Asset) > 4 {
		return nil, fmt.Errorf("asset parameter supports maximum 4 assets")
	}
	for _, asset := range opts.Asset {
		params.Add("asset", asset)
	}

	// Optional parameters
	if opts.TxHash != "" {
		params.Set("tx_hash", opts.TxHash)
	}
	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}
	if opts.Limit > 0 {
		if opts.Limit > 100 {
			return nil, fmt.Errorf("limit cannot exceed 100")
		}
		params.Set("limit", strconv.Itoa(opts.Limit))
	}

	u = u + "?" + params.Encode()

	var resp PaginatedResponse[Transaction]
	if err := s.client.doRequest(ctx, "GET", u, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// InitializeImport initializes the wallet import process.
func (s *WalletsService) InitializeImport(ctx context.Context, req *ImportWalletInitRequest) (*ImportWalletInitResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/import/initialize", s.client.baseURL)

	var resp ImportWalletInitResponse
	if err := s.client.doRequest(ctx, "POST", u, req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SubmitImport completes the wallet import process.
func (s *WalletsService) SubmitImport(ctx context.Context, req *ImportWalletSubmitRequest) (*Wallet, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/import/submit", s.client.baseURL)

	var wallet Wallet
	if err := s.client.doRequest(ctx, "POST", u, req, &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

// GetTransaction retrieves a specific transaction by ID.
func (s *WalletsService) GetTransaction(ctx context.Context, transactionID string) (*Transaction, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/transactions/%s", s.client.baseURL, transactionID)

	var tx Transaction
	if err := s.client.doRequest(ctx, "GET", u, nil, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

// GetTransactionByHash retrieves a specific transaction by wallet ID and transaction hash.
// This is a convenience method that filters GetTransactions by tx_hash.
func (s *WalletsService) GetTransactionByHash(ctx context.Context, walletID, chain string, assets []string, txHash string) (*Transaction, error) {
	opts := &GetTransactionsOptions{
		Chain:  chain,
		Asset:  assets,
		TxHash: txHash,
		Limit:  1,
	}

	resp, err := s.GetTransactions(ctx, walletID, opts)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("transaction not found: %s", txHash)
	}

	return &resp.Data[0], nil
}
