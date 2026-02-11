package privy

import "context"

// TonWalletsService handles TON-specific wallet operations.
type TonWalletsService struct {
	client *Client
}

// TON CAIP-2 network identifiers.
const (
	TonMainnet = "ton:-239"
)

// RawSign signs a pre-computed hash using the TON wallet's key.
func (s *TonWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the TON wallet's key.
func (s *TonWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
