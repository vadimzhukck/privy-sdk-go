package privy

import "time"

// ChainType represents supported blockchain types.
type ChainType string

const (
	ChainTypeEthereum      ChainType = "ethereum"
	ChainTypeSolana        ChainType = "solana"
	ChainTypeStellar       ChainType = "stellar"
	ChainTypeCosmos        ChainType = "cosmos"
	ChainTypeSui           ChainType = "sui"
	ChainTypeTron          ChainType = "tron"
	ChainTypeBitcoinSegwit ChainType = "bitcoin-segwit"
	ChainTypeNear          ChainType = "near"
	ChainTypeTon           ChainType = "ton"
	ChainTypeStarknet      ChainType = "starknet"
	ChainTypeAptos         ChainType = "aptos"
)

// LinkedAccountType represents the type of linked account.
type LinkedAccountType string

const (
	LinkedAccountTypeEmail      LinkedAccountType = "email"
	LinkedAccountTypePhone      LinkedAccountType = "phone"
	LinkedAccountTypeWallet     LinkedAccountType = "wallet"
	LinkedAccountTypeSmartWallet LinkedAccountType = "smart_wallet"
	LinkedAccountTypeGoogle     LinkedAccountType = "google_oauth"
	LinkedAccountTypeTwitter    LinkedAccountType = "twitter_oauth"
	LinkedAccountTypeDiscord    LinkedAccountType = "discord_oauth"
	LinkedAccountTypeGithub     LinkedAccountType = "github_oauth"
	LinkedAccountTypeSpotify    LinkedAccountType = "spotify_oauth"
	LinkedAccountTypeInstagram  LinkedAccountType = "instagram_oauth"
	LinkedAccountTypeTiktok     LinkedAccountType = "tiktok_oauth"
	LinkedAccountTypeTwitch     LinkedAccountType = "twitch_oauth"
	LinkedAccountTypeApple      LinkedAccountType = "apple_oauth"
	LinkedAccountTypeLinkedin   LinkedAccountType = "linkedin_oauth"
	LinkedAccountTypeFarcaster  LinkedAccountType = "farcaster"
	LinkedAccountTypeTelegram   LinkedAccountType = "telegram"
	LinkedAccountTypeCustomAuth LinkedAccountType = "custom_auth"
)

// User represents a Privy user.
type User struct {
	ID             string           `json:"id"`
	CreatedAt      int64            `json:"created_at"`
	LinkedAccounts []LinkedAccount  `json:"linked_accounts"`
	MFAMethods     []MFAMethod      `json:"mfa_methods,omitempty"`
	HasAcceptedTerms bool           `json:"has_accepted_terms,omitempty"`
	IsGuest        bool             `json:"is_guest,omitempty"`
	CustomMetadata map[string]any   `json:"custom_metadata,omitempty"`
}

// CreatedAtTime returns the created_at timestamp as a time.Time.
func (u *User) CreatedAtTime() time.Time {
	return time.UnixMilli(u.CreatedAt)
}

// GetWallets returns all wallet accounts linked to this user.
func (u *User) GetWallets() []LinkedAccount {
	var wallets []LinkedAccount
	for _, account := range u.LinkedAccounts {
		if account.Type == LinkedAccountTypeWallet {
			wallets = append(wallets, account)
		}
	}
	return wallets
}

// GetWalletsByChain returns wallet accounts linked to this user filtered by chain type.
func (u *User) GetWalletsByChain(chainType ChainType) []LinkedAccount {
	var wallets []LinkedAccount
	for _, account := range u.LinkedAccounts {
		if account.Type == LinkedAccountTypeWallet && account.ChainType == chainType {
			wallets = append(wallets, account)
		}
	}
	return wallets
}

// LinkedAccount represents an account linked to a Privy user.
type LinkedAccount struct {
	Type              LinkedAccountType `json:"type"`
	Address           string            `json:"address,omitempty"`
	ChainType         ChainType         `json:"chain_type,omitempty"`
	ChainID           string            `json:"chain_id,omitempty"`
	WalletClient      string            `json:"wallet_client,omitempty"`
	WalletClientType  string            `json:"wallet_client_type,omitempty"`
	ConnectorType     string            `json:"connector_type,omitempty"`
	VerifiedAt        int64             `json:"verified_at,omitempty"`
	FirstVerifiedAt   int64             `json:"first_verified_at,omitempty"`
	LatestVerifiedAt  int64             `json:"latest_verified_at,omitempty"`

	// Email/Phone fields
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`

	// OAuth fields
	Subject     string `json:"subject,omitempty"`
	Name        string `json:"name,omitempty"`
	Username    string `json:"username,omitempty"`
	Email_      string `json:"email_,omitempty"`

	// Farcaster fields
	FID         int64  `json:"fid,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Bio         string `json:"bio,omitempty"`
	PfpURL      string `json:"pfp_url,omitempty"`

	// Telegram fields
	TelegramUserID string `json:"telegram_user_id,omitempty"`
	FirstName      string `json:"first_name,omitempty"`
	LastName       string `json:"last_name,omitempty"`
	PhotoURL       string `json:"photo_url,omitempty"`

	// Custom Auth fields
	CustomUserID string `json:"custom_user_id,omitempty"`
}

// MFAMethod represents a multi-factor authentication method.
type MFAMethod struct {
	Type       string `json:"type"`
	VerifiedAt int64  `json:"verified_at"`
}

// Wallet represents a Privy wallet.
type Wallet struct {
	ID                string            `json:"id"`
	Address           string            `json:"address"`
	ChainType         ChainType         `json:"chain_type"`
	PolicyIDs         []string          `json:"policy_ids,omitempty"`
	OwnerID           string            `json:"owner_id,omitempty"`
	AdditionalSigners []AdditionalSigner `json:"additional_signers,omitempty"`
	CreatedAt         int64             `json:"created_at"`
}

// CreatedAtTime returns the created_at timestamp as a time.Time.
func (w *Wallet) CreatedAtTime() time.Time {
	return time.UnixMilli(w.CreatedAt)
}

// AdditionalSigner represents an additional signer for a wallet.
type AdditionalSigner struct {
	SignerID string `json:"signer_id"`
}

// WalletOwner represents the owner of a wallet.
type WalletOwner struct {
	UserID    string `json:"user_id,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

// Transaction represents a blockchain transaction.
type Transaction struct {
	ID        string    `json:"id"`
	WalletID  string    `json:"wallet_id"`
	ChainType ChainType `json:"chain_type"`
	CAIP2     string    `json:"caip2"`
	Hash      string    `json:"hash,omitempty"`
	Status    string    `json:"status"`
	CreatedAt int64     `json:"created_at"`
}

// EthereumTransaction represents an Ethereum transaction request.
type EthereumTransaction struct {
	To                   string `json:"to"`
	From                 string `json:"from,omitempty"`
	Value                string `json:"value,omitempty"`
	Data                 string `json:"data,omitempty"`
	ChainID              int64  `json:"chain_id,omitempty"`
	GasLimit             string `json:"gas_limit,omitempty"`
	GasPrice             string `json:"gas_price,omitempty"`
	MaxFeePerGas         string `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFeePerGas string `json:"max_priority_fee_per_gas,omitempty"`
	Nonce                int64  `json:"nonce,omitempty"`
	Type                 int    `json:"type,omitempty"`
}

// SolanaTransaction represents a Solana transaction.
type SolanaTransaction struct {
	Message  string `json:"message"`
	Encoding string `json:"encoding,omitempty"`
}

// TypedData represents EIP-712 typed data for signing.
type TypedData struct {
	Domain      TypedDataDomain             `json:"domain"`
	Types       map[string][]TypedDataField `json:"types"`
	PrimaryType string                      `json:"primary_type"`
	Message     map[string]any              `json:"message"`
}

// TypedDataDomain represents the domain of EIP-712 typed data.
type TypedDataDomain struct {
	Name              string `json:"name,omitempty"`
	Version           string `json:"version,omitempty"`
	ChainID           int64  `json:"chainId,omitempty"`
	VerifyingContract string `json:"verifyingContract,omitempty"`
	Salt              string `json:"salt,omitempty"`
}

// TypedDataField represents a field in EIP-712 typed data.
type TypedDataField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Policy represents a wallet policy.
type Policy struct {
	ID        string       `json:"id"`
	Name      string       `json:"name,omitempty"`
	Rules     []PolicyRule `json:"rules,omitempty"`
	CreatedAt int64        `json:"created_at"`
	UpdatedAt int64        `json:"updated_at,omitempty"`
}

// PolicyRule represents a rule within a policy.
type PolicyRule struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"`
	Conditions []RuleCondition `json:"conditions,omitempty"`
}

// RuleCondition represents a condition in a policy rule.
type RuleCondition struct {
	Type    string `json:"type"`
	Field   string `json:"field,omitempty"`
	Value   any    `json:"value,omitempty"`
	Operator string `json:"operator,omitempty"`
}

// ConditionSet represents a condition set for access control.
type ConditionSet struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	OwnerID   string `json:"owner_id,omitempty"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at,omitempty"`
}

// ConditionSetItem represents an item in a condition set.
type ConditionSetItem struct {
	ID    string `json:"id"`
	Value any    `json:"value"`
}

// KeyQuorum represents a key quorum for wallet authorization.
type KeyQuorum struct {
	ID        string `json:"id"`
	PublicKey string `json:"public_key"`
	CreatedAt int64  `json:"created_at"`
}

// PaginatedResponse represents a paginated API response.
type PaginatedResponse[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// ListOptions represents pagination options for list operations.
type ListOptions struct {
	Cursor string
	Limit  int
}

// SignatureResponse represents a signature response from the API.
type SignatureResponse struct {
	Method string `json:"method"`
	Data   struct {
		Signature         string `json:"signature,omitempty"`
		SignedTransaction string `json:"signed_transaction,omitempty"`
		Hash              string `json:"hash,omitempty"`
		Encoding          string `json:"encoding,omitempty"`
		CAIP2             string `json:"caip2,omitempty"`
	} `json:"data"`
}

// WalletBalance represents the balance of a wallet.
type WalletBalance struct {
	Balance  string `json:"balance"`
	Currency string `json:"currency,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
}
