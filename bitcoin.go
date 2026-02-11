package privy

import "context"

// BitcoinWalletsService handles Bitcoin SegWit-specific wallet operations.
type BitcoinWalletsService struct {
	client *Client
}

// Bitcoin CAIP-2 network identifiers (BIP-122 format).
const (
	BitcoinMainnet = "bip122:000000000019d6689c085ae165831e93"
	BitcoinTestnet = "bip122:000000000933ea01ad0ee984209779ba"
)

// RawSign signs a pre-computed hash using the Bitcoin wallet's key.
func (s *BitcoinWalletsService) RawSign(ctx context.Context, walletID string, hash string) (*RawSignResponse, error) {
	return s.client.RawSign(ctx, walletID, hash)
}

// RawSignBytes signs bytes using a specified hash function with the Bitcoin wallet's key.
func (s *BitcoinWalletsService) RawSignBytes(ctx context.Context, walletID string, data string, encoding string, hashFunction string) (*RawSignResponse, error) {
	return s.client.RawSignBytes(ctx, walletID, data, encoding, hashFunction)
}
