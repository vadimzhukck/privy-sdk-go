package privy

import (
	"context"
	"fmt"
)

// SparkWalletsService handles Spark (Bitcoin Lightning)-specific wallet operations.
type SparkWalletsService struct {
	client *Client
}

// Transfer sends satoshis to a Spark address.
func (s *SparkWalletsService) Transfer(ctx context.Context, walletID string, receiverAddress string, amountSats int64, network SparkNetwork, signature string) (*SparkTransferResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "transfer",
		Network: string(network),
		Params: &SparkTransferRequest{
			ReceiverSparkAddress: receiverAddress,
			AmountSats:           amountSats,
		},
	}

	var resp SparkTransferResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetBalance retrieves the balance of a Spark wallet.
func (s *SparkWalletsService) GetBalance(ctx context.Context, walletID string, network SparkNetwork, signature string) (*SparkBalanceResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "getBalance",
		Network: string(network),
	}

	var resp SparkBalanceResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// TransferTokens transfers Spark tokens to a Spark address.
func (s *SparkWalletsService) TransferTokens(ctx context.Context, walletID string, tokenIdentifier string, tokenAmount int64, receiverAddress string, network SparkNetwork, signature string) (*SparkResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "transferTokens",
		Network: string(network),
		Params: &SparkTransferTokensRequest{
			TokenIdentifier:      tokenIdentifier,
			TokenAmount:          tokenAmount,
			ReceiverSparkAddress: receiverAddress,
		},
	}

	var resp SparkResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetStaticDepositAddress retrieves a static Bitcoin deposit address for the Spark wallet.
func (s *SparkWalletsService) GetStaticDepositAddress(ctx context.Context, walletID string, network SparkNetwork, signature string) (*SparkDepositAddressResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "getStaticDepositAddress",
		Network: string(network),
	}

	var resp SparkDepositAddressResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetClaimStaticDepositQuote retrieves a quote for claiming a static deposit.
func (s *SparkWalletsService) GetClaimStaticDepositQuote(ctx context.Context, walletID string, network SparkNetwork, signature string) (*SparkResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "getClaimStaticDepositQuote",
		Network: string(network),
	}

	var resp SparkResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ClaimStaticDeposit claims a static deposit after it has been confirmed on Bitcoin.
// Requires 3+ confirmations on the Bitcoin transaction.
func (s *SparkWalletsService) ClaimStaticDeposit(ctx context.Context, walletID string, txID string, creditAmountSats int64, sspSignature string, network SparkNetwork, signature string) (*SparkResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "claimStaticDeposit",
		Network: string(network),
		Params: &SparkClaimStaticDepositRequest{
			TxID:             txID,
			CreditAmountSats: creditAmountSats,
			SspSignature:     sspSignature,
		},
	}

	var resp SparkResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateLightningInvoice creates a Lightning Network invoice.
func (s *SparkWalletsService) CreateLightningInvoice(ctx context.Context, walletID string, amountSats int64, network SparkNetwork, signature string) (*SparkLightningInvoiceResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "createLightningInvoice",
		Network: string(network),
		Params: &SparkCreateLightningInvoiceRequest{
			AmountSats: amountSats,
		},
	}

	var resp SparkLightningInvoiceResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// PayLightningInvoice pays a Lightning Network invoice.
func (s *SparkWalletsService) PayLightningInvoice(ctx context.Context, walletID string, invoice string, maxFeeSats int64, network SparkNetwork, signature string) (*SparkResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "payLightningInvoice",
		Network: string(network),
		Params: &SparkPayLightningInvoiceRequest{
			Invoice:    invoice,
			MaxFeeSats: maxFeeSats,
		},
	}

	var resp SparkResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SignMessage signs a message using the Spark wallet's identity key.
func (s *SparkWalletsService) SignMessage(ctx context.Context, walletID string, message string, compact bool, network SparkNetwork, signature string) (*SparkSignatureResponse, error) {
	u := fmt.Sprintf("%s/wallets/%s/rpc", s.client.baseURL, walletID)

	if network == "" {
		network = SparkNetworkMainnet
	}

	req := &SparkRPCRequest{
		Method:  "signMessageWithIdentityKey",
		Network: string(network),
		Params: &SparkSignMessageRequest{
			Message: message,
			Compact: compact,
		},
	}

	var resp SparkSignatureResponse
	if err := s.client.doRequestWithSignature(ctx, "POST", u, req, &resp, signature); err != nil {
		return nil, err
	}

	return &resp, nil
}
