package privy

import "context"

// TronWalletsService handles Tron-specific wallet operations.
type TronWalletsService struct {
	client *Client
}

// Tron CAIP-2 network identifiers.
const (
	TronMainnet = "tron:0x2b6653dc"
)

// RawSign signs a pre-computed hash using the Tron wallet's key.
func (s *TronWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the Tron wallet's key.
func (s *TronWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
