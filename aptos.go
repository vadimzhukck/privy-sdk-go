package privy

import "context"

// AptosWalletsService handles Aptos-specific wallet operations.
type AptosWalletsService struct {
	client *Client
}

// Aptos CAIP-2 network identifiers.
const (
	AptosMainnet = "aptos:mainnet"
	AptosTestnet = "aptos:testnet"
)

// RawSign signs a pre-computed hash using the Aptos wallet's key.
func (s *AptosWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the Aptos wallet's key.
func (s *AptosWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
