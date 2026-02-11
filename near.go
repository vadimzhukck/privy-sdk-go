package privy

import "context"

// NearWalletsService handles NEAR-specific wallet operations.
type NearWalletsService struct {
	client *Client
}

// NEAR CAIP-2 network identifiers.
const (
	NearMainnet = "near:mainnet"
	NearTestnet = "near:testnet"
)

// RawSign signs a pre-computed hash using the NEAR wallet's key.
func (s *NearWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the NEAR wallet's key.
func (s *NearWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
