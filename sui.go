package privy

import "context"

// SuiWalletsService handles Sui-specific wallet operations.
type SuiWalletsService struct {
	client *Client
}

// Sui CAIP-2 network identifiers.
const (
	SuiMainnet = "sui:mainnet"
	SuiTestnet = "sui:testnet"
	SuiDevnet  = "sui:devnet"
)

// RawSign signs a pre-computed hash using the Sui wallet's key.
func (s *SuiWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the Sui wallet's key.
func (s *SuiWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
