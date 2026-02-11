// Package privy provides a Go SDK for the Privy API.
// Privy is a wallet infrastructure service that enables user authentication,
// embedded wallet creation, and transaction signing across multiple blockchains.
package privy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrNilClient is returned when a service method is called on a nil or uninitialized client.
var ErrNilClient = errors.New("privy: client is not initialized")

const (
	// DefaultBaseURL is the default Privy API base URL.
	DefaultBaseURL = "https://api.privy.io/v1"

	// DefaultAuthBaseURL is the Privy authentication API base URL.
	DefaultAuthBaseURL = "https://auth.privy.io/api/v1"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second
)

// Client is the main Privy API client.
type Client struct {
	appID      string
	appSecret  string
	baseURL    string
	authURL    string
	httpClient *http.Client
	testnet    bool
	chainOpts  map[string][]any
}

// ClientOption is a function that configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the API.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithAuthURL sets a custom auth URL for the API.
func WithAuthURL(url string) ClientOption {
	return func(c *Client) {
		c.authURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTestnet configures the client for testnet/devnet networks.
// Chain helpers created with this client will automatically use testnet defaults.
func WithTestnet() ClientOption {
	return func(c *Client) {
		c.testnet = true
	}
}

// Testnet returns whether the client is configured for testnet.
func (c *Client) Testnet() bool {
	return c.testnet
}

// ChainOptions returns stored options for a specific chain.
// Chain helpers use this to retrieve options set at client creation time.
func (c *Client) ChainOptions(chain string) []any {
	if c.chainOpts == nil {
		return nil
	}
	return c.chainOpts[chain]
}

// withChain stores chain-specific options on the client.
func withChain(chain string, opts ...any) ClientOption {
	return func(c *Client) {
		if c.chainOpts == nil {
			c.chainOpts = make(map[string][]any)
		}
		c.chainOpts[chain] = append(c.chainOpts[chain], opts...)
	}
}

// WithEthereum sets Ethereum chain helper options at the client level.
func WithEthereum(opts ...any) ClientOption { return withChain("ethereum", opts...) }

// WithSolana sets Solana chain helper options at the client level.
func WithSolana(opts ...any) ClientOption { return withChain("solana", opts...) }

// WithBitcoin sets Bitcoin chain helper options at the client level.
func WithBitcoin(opts ...any) ClientOption { return withChain("bitcoin", opts...) }

// WithStellar sets Stellar chain helper options at the client level.
func WithStellar(opts ...any) ClientOption { return withChain("stellar", opts...) }

// WithNEAR sets NEAR chain helper options at the client level.
func WithNEAR(opts ...any) ClientOption { return withChain("near", opts...) }

// WithSui sets Sui chain helper options at the client level.
func WithSui(opts ...any) ClientOption { return withChain("sui", opts...) }

// WithTON sets TON chain helper options at the client level.
func WithTON(opts ...any) ClientOption { return withChain("ton", opts...) }

// WithCosmos sets Cosmos chain helper options at the client level.
func WithCosmos(opts ...any) ClientOption { return withChain("cosmos", opts...) }

// WithTron sets Tron chain helper options at the client level.
func WithTron(opts ...any) ClientOption { return withChain("tron", opts...) }

// WithStarknet sets StarkNet chain helper options at the client level.
func WithStarknet(opts ...any) ClientOption { return withChain("starknet", opts...) }

// WithAptos sets Aptos chain helper options at the client level.
func WithAptos(opts ...any) ClientOption { return withChain("aptos", opts...) }

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new Privy API client.
func NewClient(appID, appSecret string, opts ...ClientOption) *Client {
	c := &Client{
		appID:     appID,
		appSecret: appSecret,
		baseURL:   DefaultBaseURL,
		authURL:   DefaultAuthBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Users returns the Users service for user management operations.
func (c *Client) Users() *UsersService {
	return &UsersService{client: c}
}

// Wallets returns the Wallets service for wallet operations.
func (c *Client) Wallets() *WalletsService {
	return &WalletsService{client: c}
}

// Policies returns the Policies service for policy management.
func (c *Client) Policies() *PoliciesService {
	return &PoliciesService{client: c}
}

// ConditionSets returns the ConditionSets service for condition set management.
func (c *Client) ConditionSets() *ConditionSetsService {
	return &ConditionSetsService{client: c}
}

// KeyQuorums returns the KeyQuorums service for key quorum management.
func (c *Client) KeyQuorums() *KeyQuorumsService {
	return &KeyQuorumsService{client: c}
}

// Auth returns the Auth service for token verification.
func (c *Client) Auth() *AuthService {
	return newAuthService(c)
}

// RawSign signs a pre-computed hash using the wallet's key.
// Uses POST /wallets/{walletID}/raw_sign endpoint.
func (c *Client) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/raw_sign", c.baseURL, walletID)
	req := &RawSignHashRequest{Params: RawSignHashParams{Hash: hash}}
	var resp RawSignResponse
	if err := c.doRequest(ctx, "POST", u, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RawSignBytes signs bytes using a specified hash function.
// Uses POST /wallets/{walletID}/raw_sign endpoint.
func (c *Client) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/wallets/%s/raw_sign", c.baseURL, walletID)
	req := &RawSignBytesRequest{Params: RawSignBytesParams{
		Bytes: data, Encoding: encoding, HashFunction: hashFunction,
	}}
	var resp RawSignResponse
	if err := c.doRequest(ctx, "POST", u, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// basicAuth returns the Basic Auth header value.
func (c *Client) basicAuth() string {
	auth := c.appID + ":" + c.appSecret
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// doRequest performs an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.basicAuth())
	req.Header.Set("privy-app-id", c.appID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// doRequestWithSignature performs an HTTP request with authorization signature.
func (c *Client) doRequestWithSignature(ctx context.Context, method, url string, body interface{}, result interface{}, signature string) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.basicAuth())
	req.Header.Set("privy-app-id", c.appID)
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("privy-authorization-signature", signature)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// APIError represents an error response from the Privy API.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	Error_     string `json:"error"`
	Code       string `json:"code"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("privy: API error (status %d): %s", e.StatusCode, e.Message)
	}
	if e.Error_ != "" {
		return fmt.Sprintf("privy: API error (status %d): %s", e.StatusCode, e.Error_)
	}
	return fmt.Sprintf("privy: API error (status %d)", e.StatusCode)
}
