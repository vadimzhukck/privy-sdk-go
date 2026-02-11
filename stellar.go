package privy

import "context"

// StellarWalletsService handles Stellar-specific wallet operations.
type StellarWalletsService struct {
	client *Client
}

// Stellar CAIP-2 network identifiers.
const (
	StellarMainnet = "stellar:pubnet"
	StellarTestnet = "stellar:testnet"
)

// RawSign signs a pre-computed hash using the Stellar wallet's key.
func (s *StellarWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the Stellar wallet's key.
func (s *StellarWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
