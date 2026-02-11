package privy

import "context"

// CosmosWalletsService handles Cosmos-specific wallet operations.
type CosmosWalletsService struct {
	client *Client
}

// Cosmos CAIP-2 network identifiers.
const (
	CosmosHubMainnet = "cosmos:cosmoshub-4"
)

// RawSign signs a pre-computed hash using the Cosmos wallet's key.
func (s *CosmosWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the Cosmos wallet's key.
func (s *CosmosWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
