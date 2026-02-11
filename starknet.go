package privy

import "context"

// StarknetWalletsService handles Starknet-specific wallet operations.
type StarknetWalletsService struct {
	client *Client
}

// Starknet CAIP-2 network identifiers.
const (
	StarknetMainnet = "starknet:SN_MAIN"
	StarknetSepolia = "starknet:SN_SEPOLIA"
)

// RawSign signs a pre-computed hash using the Starknet wallet's key.
func (s *StarknetWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the Starknet wallet's key.
func (s *StarknetWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
