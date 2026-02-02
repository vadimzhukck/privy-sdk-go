package privy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WebhookHandler handles incoming Privy webhooks with signature verification.
type WebhookHandler struct {
	signingSecret    string
	toleranceSeconds int64
	handlers         map[WebhookEventType][]WebhookEventHandler
}

// WebhookEventHandler is a function that handles a webhook event.
type WebhookEventHandler func(event WebhookEvent)

// WebhookEvent represents a Privy webhook event.
type WebhookEvent struct {
	ID        string           `json:"id"`
	Type      WebhookEventType `json:"type"`
	CreatedAt int64            `json:"created_at"`
	Data      json.RawMessage  `json:"data"`
	AppID     string           `json:"app_id"`
}

// WebhookEventType represents the type of webhook event.
type WebhookEventType string

const (
	// User events
	WebhookEventUserCreated           WebhookEventType = "user.created"
	WebhookEventUserUpdated           WebhookEventType = "user.updated"
	WebhookEventUserDeleted           WebhookEventType = "user.deleted"
	WebhookEventUserLinkedAccount     WebhookEventType = "user.linked_account"
	WebhookEventUserUnlinkedAccount   WebhookEventType = "user.unlinked_account"
	WebhookEventUserUpdatedAccount    WebhookEventType = "user.updated_account"
	WebhookEventUserTransferredAccount WebhookEventType = "user.transferred_account"
	WebhookEventUserAuthenticated     WebhookEventType = "user.authenticated"
	WebhookEventUserWalletCreated     WebhookEventType = "user.wallet_created"

	// Wallet events
	WebhookEventWalletCreated        WebhookEventType = "wallet.created"
	WebhookEventWalletTransferred    WebhookEventType = "wallet.transferred"
	WebhookEventWalletFundsDeposited WebhookEventType = "wallet.funds_deposited"
	WebhookEventWalletFundsWithdrawn WebhookEventType = "wallet.funds_withdrawn"
	WebhookEventWalletPrivateKeyExport WebhookEventType = "wallet.private_key_export"
	WebhookEventWalletRecoverySetup  WebhookEventType = "wallet.recovery_setup"
	WebhookEventWalletRecovered      WebhookEventType = "wallet.recovered"

	// Transaction events
	WebhookEventTransactionCreated         WebhookEventType = "transaction.created"
	WebhookEventTransactionBroadcasted     WebhookEventType = "transaction.broadcasted"
	WebhookEventTransactionConfirmed       WebhookEventType = "transaction.confirmed"
	WebhookEventTransactionCompleted       WebhookEventType = "transaction.completed"
	WebhookEventTransactionExecutionReverted WebhookEventType = "transaction.execution_reverted"
	WebhookEventTransactionStillPending    WebhookEventType = "transaction.still_pending"
	WebhookEventTransactionFailed          WebhookEventType = "transaction.failed"
	WebhookEventTransactionReplaced        WebhookEventType = "transaction.replaced"
	WebhookEventTransactionProviderError   WebhookEventType = "transaction.provider_error"

	// MFA events
	WebhookEventMFAEnabled  WebhookEventType = "mfa.enabled"
	WebhookEventMFADisabled WebhookEventType = "mfa.disabled"
)

// UserCreatedEvent represents a user.created webhook event.
type UserCreatedEvent struct {
	UserID         string           `json:"user_id"`
	LinkedAccounts []LinkedAccount  `json:"linked_accounts"`
	CreatedAt      int64            `json:"created_at"`
	CustomMetadata map[string]any   `json:"custom_metadata,omitempty"`
}

// UserUpdatedEvent represents a user.updated webhook event.
type UserUpdatedEvent struct {
	UserID         string           `json:"user_id"`
	LinkedAccounts []LinkedAccount  `json:"linked_accounts"`
	UpdatedAt      int64            `json:"updated_at"`
	CustomMetadata map[string]any   `json:"custom_metadata,omitempty"`
}

// UserDeletedEvent represents a user.deleted webhook event.
type UserDeletedEvent struct {
	UserID    string `json:"user_id"`
	DeletedAt int64  `json:"deleted_at"`
}

// UserAuthenticatedEvent represents a user.authenticated webhook event.
type UserAuthenticatedEvent struct {
	UserID          string `json:"user_id"`
	AuthMethod      string `json:"auth_method"`
	AuthenticatedAt int64  `json:"authenticated_at"`
}

// WalletCreatedEvent represents a wallet.created webhook event.
type WalletCreatedEvent struct {
	WalletID  string    `json:"wallet_id"`
	Address   string    `json:"address"`
	ChainType ChainType `json:"chain_type"`
	OwnerID   string    `json:"owner_id,omitempty"`
	CreatedAt int64     `json:"created_at"`
}

// WalletTransferredEvent represents a wallet.transferred webhook event.
type WalletTransferredEvent struct {
	WalletID      string `json:"wallet_id"`
	OldOwnerID    string `json:"old_owner_id"`
	NewOwnerID    string `json:"new_owner_id"`
	TransferredAt int64  `json:"transferred_at"`
}

// UserUnlinkedAccountEvent represents a user.unlinked_account webhook event.
type UserUnlinkedAccountEvent struct {
	User    User          `json:"user"`
	Account LinkedAccount `json:"account"`
}

// UserUpdatedAccountEvent represents a user.updated_account webhook event.
type UserUpdatedAccountEvent struct {
	User    User          `json:"user"`
	Account LinkedAccount `json:"account"`
}

// UserTransferredAccountEvent represents a user.transferred_account webhook event.
type UserTransferredAccountEvent struct {
	FromUser    UserRef       `json:"fromUser"`
	ToUser      User          `json:"toUser"`
	Account     LinkedAccount `json:"account"`
	DeletedUser bool          `json:"deletedUser"`
}

// UserRef represents a minimal user reference.
type UserRef struct {
	ID string `json:"id"`
}

// UserWalletCreatedEvent represents a user.wallet_created webhook event.
type UserWalletCreatedEvent struct {
	User   User          `json:"user"`
	Wallet LinkedAccount `json:"wallet"`
}

// TransactionEvent represents a transaction webhook event.
type TransactionEvent struct {
	TransactionID   string    `json:"transaction_id"`
	WalletID        string    `json:"wallet_id"`
	ChainType       ChainType `json:"chain_type,omitempty"`
	CAIP2           string    `json:"caip2,omitempty"`
	TransactionHash string    `json:"transaction_hash,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       int64     `json:"created_at,omitempty"`
	CompletedAt     int64     `json:"completed_at,omitempty"`
	FailedAt        int64     `json:"failed_at,omitempty"`
	Error           string    `json:"error,omitempty"`
	TransactionRequest *TransactionRequest `json:"transaction_request,omitempty"` // For still_pending events
}

// TransactionRequest represents a transaction request in still_pending events.
type TransactionRequest struct {
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Value    string `json:"value,omitempty"`
	Data     string `json:"data,omitempty"`
	Gas      string `json:"gas,omitempty"`
	GasPrice string `json:"gasPrice,omitempty"`
}

// WalletFundsEvent represents wallet.funds_deposited and wallet.funds_withdrawn events.
type WalletFundsEvent struct {
	WalletID        string       `json:"wallet_id"`
	IdempotencyKey  string       `json:"idempotency_key"`
	CAIP2           string       `json:"caip2"`
	Asset           AssetInfo    `json:"asset"`
	Amount          string       `json:"amount"`
	TransactionHash string       `json:"transaction_hash"`
	Sender          string       `json:"sender"`
	Recipient       string       `json:"recipient"`
	Block           BlockInfo    `json:"block"`
}

// AssetInfo represents asset information in fund events.
type AssetInfo struct {
	Type    string `json:"type"`              // "native" or "erc20" or "spl"
	Address string `json:"address,omitempty"` // For ERC20 tokens
	Mint    string `json:"mint,omitempty"`    // For SPL tokens
}

// BlockInfo represents block information.
type BlockInfo struct {
	Number int64 `json:"number"`
}

// WalletSecurityEvent represents wallet security events (private_key_export, recovery_setup, recovered).
type WalletSecurityEvent struct {
	UserID        string `json:"user_id"`
	WalletID      string `json:"wallet_id"`
	WalletAddress string `json:"wallet_address"`
	Method        string `json:"method,omitempty"` // For recovery_setup events
}

// MFAEvent represents MFA events (mfa.enabled, mfa.disabled).
type MFAEvent struct {
	UserID string `json:"user_id"`
	Method string `json:"method"` // "sms", "totp", or "passkey"
}

var (
	ErrInvalidWebhookSignature = errors.New("privy: invalid webhook signature")
	ErrWebhookTimestampExpired = errors.New("privy: webhook timestamp expired")
	ErrMissingWebhookHeaders   = errors.New("privy: missing webhook headers")
)

const (
	// DefaultWebhookTolerance is the default tolerance for webhook timestamps (5 minutes).
	DefaultWebhookTolerance = 300

	// WebhookSignatureHeader is the header containing the webhook signature.
	WebhookSignatureHeader = "svix-signature"

	// WebhookTimestampHeader is the header containing the webhook timestamp.
	WebhookTimestampHeader = "svix-timestamp"

	// WebhookIDHeader is the header containing the webhook ID.
	WebhookIDHeader = "svix-id"
)

// NewWebhookHandler creates a new webhook handler with the given signing secret.
func NewWebhookHandler(signingSecret string) *WebhookHandler {
	// Remove "whsec_" prefix if present
	secret := strings.TrimPrefix(signingSecret, "whsec_")

	return &WebhookHandler{
		signingSecret:    secret,
		toleranceSeconds: DefaultWebhookTolerance,
		handlers:         make(map[WebhookEventType][]WebhookEventHandler),
	}
}

// WithTolerance sets the tolerance for webhook timestamps in seconds.
func (h *WebhookHandler) WithTolerance(seconds int64) *WebhookHandler {
	h.toleranceSeconds = seconds
	return h
}

// OnEvent registers a handler for a specific event type.
func (h *WebhookHandler) OnEvent(eventType WebhookEventType, handler WebhookEventHandler) {
	h.handlers[eventType] = append(h.handlers[eventType], handler)
}

// OnUserCreated registers a handler for user.created events.
func (h *WebhookHandler) OnUserCreated(handler func(*UserCreatedEvent)) {
	h.OnEvent(WebhookEventUserCreated, func(e WebhookEvent) {
		var data UserCreatedEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnUserUpdated registers a handler for user.updated events.
func (h *WebhookHandler) OnUserUpdated(handler func(*UserUpdatedEvent)) {
	h.OnEvent(WebhookEventUserUpdated, func(e WebhookEvent) {
		var data UserUpdatedEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnUserDeleted registers a handler for user.deleted events.
func (h *WebhookHandler) OnUserDeleted(handler func(*UserDeletedEvent)) {
	h.OnEvent(WebhookEventUserDeleted, func(e WebhookEvent) {
		var data UserDeletedEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnUserAuthenticated registers a handler for user.authenticated events.
func (h *WebhookHandler) OnUserAuthenticated(handler func(*UserAuthenticatedEvent)) {
	h.OnEvent(WebhookEventUserAuthenticated, func(e WebhookEvent) {
		var data UserAuthenticatedEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnWalletCreated registers a handler for wallet.created events.
func (h *WebhookHandler) OnWalletCreated(handler func(*WalletCreatedEvent)) {
	h.OnEvent(WebhookEventWalletCreated, func(e WebhookEvent) {
		var data WalletCreatedEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnWalletTransferred registers a handler for wallet.transferred events.
func (h *WebhookHandler) OnWalletTransferred(handler func(*WalletTransferredEvent)) {
	h.OnEvent(WebhookEventWalletTransferred, func(e WebhookEvent) {
		var data WalletTransferredEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionCreated registers a handler for transaction.created events.
func (h *WebhookHandler) OnTransactionCreated(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionCreated, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionCompleted registers a handler for transaction.completed events.
func (h *WebhookHandler) OnTransactionCompleted(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionCompleted, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionFailed registers a handler for transaction.failed events.
func (h *WebhookHandler) OnTransactionFailed(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionFailed, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnUserUnlinkedAccount registers a handler for user.unlinked_account events.
func (h *WebhookHandler) OnUserUnlinkedAccount(handler func(*UserUnlinkedAccountEvent)) {
	h.OnEvent(WebhookEventUserUnlinkedAccount, func(e WebhookEvent) {
		var data UserUnlinkedAccountEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnUserUpdatedAccount registers a handler for user.updated_account events.
func (h *WebhookHandler) OnUserUpdatedAccount(handler func(*UserUpdatedAccountEvent)) {
	h.OnEvent(WebhookEventUserUpdatedAccount, func(e WebhookEvent) {
		var data UserUpdatedAccountEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnUserTransferredAccount registers a handler for user.transferred_account events.
func (h *WebhookHandler) OnUserTransferredAccount(handler func(*UserTransferredAccountEvent)) {
	h.OnEvent(WebhookEventUserTransferredAccount, func(e WebhookEvent) {
		var data UserTransferredAccountEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnUserWalletCreated registers a handler for user.wallet_created events.
func (h *WebhookHandler) OnUserWalletCreated(handler func(*UserWalletCreatedEvent)) {
	h.OnEvent(WebhookEventUserWalletCreated, func(e WebhookEvent) {
		var data UserWalletCreatedEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionBroadcasted registers a handler for transaction.broadcasted events.
func (h *WebhookHandler) OnTransactionBroadcasted(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionBroadcasted, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionConfirmed registers a handler for transaction.confirmed events.
func (h *WebhookHandler) OnTransactionConfirmed(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionConfirmed, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionExecutionReverted registers a handler for transaction.execution_reverted events.
func (h *WebhookHandler) OnTransactionExecutionReverted(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionExecutionReverted, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionStillPending registers a handler for transaction.still_pending events.
func (h *WebhookHandler) OnTransactionStillPending(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionStillPending, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionReplaced registers a handler for transaction.replaced events.
func (h *WebhookHandler) OnTransactionReplaced(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionReplaced, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnTransactionProviderError registers a handler for transaction.provider_error events.
func (h *WebhookHandler) OnTransactionProviderError(handler func(*TransactionEvent)) {
	h.OnEvent(WebhookEventTransactionProviderError, func(e WebhookEvent) {
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnWalletFundsDeposited registers a handler for wallet.funds_deposited events.
func (h *WebhookHandler) OnWalletFundsDeposited(handler func(*WalletFundsEvent)) {
	h.OnEvent(WebhookEventWalletFundsDeposited, func(e WebhookEvent) {
		var data WalletFundsEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnWalletFundsWithdrawn registers a handler for wallet.funds_withdrawn events.
func (h *WebhookHandler) OnWalletFundsWithdrawn(handler func(*WalletFundsEvent)) {
	h.OnEvent(WebhookEventWalletFundsWithdrawn, func(e WebhookEvent) {
		var data WalletFundsEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnWalletPrivateKeyExport registers a handler for wallet.private_key_export events.
func (h *WebhookHandler) OnWalletPrivateKeyExport(handler func(*WalletSecurityEvent)) {
	h.OnEvent(WebhookEventWalletPrivateKeyExport, func(e WebhookEvent) {
		var data WalletSecurityEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnWalletRecoverySetup registers a handler for wallet.recovery_setup events.
func (h *WebhookHandler) OnWalletRecoverySetup(handler func(*WalletSecurityEvent)) {
	h.OnEvent(WebhookEventWalletRecoverySetup, func(e WebhookEvent) {
		var data WalletSecurityEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnWalletRecovered registers a handler for wallet.recovered events.
func (h *WebhookHandler) OnWalletRecovered(handler func(*WalletSecurityEvent)) {
	h.OnEvent(WebhookEventWalletRecovered, func(e WebhookEvent) {
		var data WalletSecurityEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnMFAEnabled registers a handler for mfa.enabled events.
func (h *WebhookHandler) OnMFAEnabled(handler func(*MFAEvent)) {
	h.OnEvent(WebhookEventMFAEnabled, func(e WebhookEvent) {
		var data MFAEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// OnMFADisabled registers a handler for mfa.disabled events.
func (h *WebhookHandler) OnMFADisabled(handler func(*MFAEvent)) {
	h.OnEvent(WebhookEventMFADisabled, func(e WebhookEvent) {
		var data MFAEvent
		if err := json.Unmarshal(e.Data, &data); err == nil {
			handler(&data)
		}
	})
}

// VerifyAndParse verifies the webhook signature and parses the event.
func (h *WebhookHandler) VerifyAndParse(r *http.Request) (*WebhookEvent, error) {
	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook body: %w", err)
	}

	// Get headers
	signature := r.Header.Get(WebhookSignatureHeader)
	timestamp := r.Header.Get(WebhookTimestampHeader)
	webhookID := r.Header.Get(WebhookIDHeader)

	if signature == "" || timestamp == "" || webhookID == "" {
		return nil, ErrMissingWebhookHeaders
	}

	// Verify timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook timestamp: %w", err)
	}

	now := time.Now().Unix()
	if now-ts > h.toleranceSeconds {
		return nil, ErrWebhookTimestampExpired
	}

	// Verify signature
	if err := h.verifySignature(webhookID, timestamp, body, signature); err != nil {
		return nil, err
	}

	// Parse event
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}

	return &event, nil
}

// HandleRequest verifies the webhook and dispatches to registered handlers.
func (h *WebhookHandler) HandleRequest(r *http.Request) (*WebhookEvent, error) {
	event, err := h.VerifyAndParse(r)
	if err != nil {
		return nil, err
	}

	// Dispatch to handlers
	if handlers, ok := h.handlers[event.Type]; ok {
		for _, handler := range handlers {
			handler(*event)
		}
	}

	return event, nil
}

// ServeHTTP implements http.Handler for use with http.Handle/http.HandleFunc.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event, err := h.HandleRequest(r)
	if err != nil {
		if errors.Is(err, ErrInvalidWebhookSignature) || errors.Is(err, ErrMissingWebhookHeaders) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, ErrWebhookTimestampExpired) {
			http.Error(w, "Request expired", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"event_id": event.ID,
	})
}

// verifySignature verifies the Svix webhook signature.
func (h *WebhookHandler) verifySignature(webhookID, timestamp string, body []byte, signatureHeader string) error {
	// Decode the secret (base64 encoded)
	secret, err := base64.StdEncoding.DecodeString(h.signingSecret)
	if err != nil {
		// If not base64, use raw
		secret = []byte(h.signingSecret)
	}

	// Construct the signed payload
	signedPayload := fmt.Sprintf("%s.%s.%s", webhookID, timestamp, string(body))

	// Compute expected signature
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signedPayload))
	expectedSig := mac.Sum(nil)
	expectedSigHex := hex.EncodeToString(expectedSig)

	// Parse signatures from header (format: "v1,sig1 v1,sig2")
	signatures := strings.Split(signatureHeader, " ")
	for _, sigPart := range signatures {
		parts := strings.SplitN(sigPart, ",", 2)
		if len(parts) != 2 {
			continue
		}
		version := parts[0]
		sig := parts[1]

		if version != "v1" {
			continue
		}

		// Compare signatures
		sigBytes, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			continue
		}

		if hmac.Equal(sigBytes, expectedSig) || sig == expectedSigHex {
			return nil
		}
	}

	return ErrInvalidWebhookSignature
}

// ParseEvent parses a raw webhook event into a typed event struct.
func (e *WebhookEvent) ParseEvent() (any, error) {
	switch e.Type {
	case WebhookEventUserCreated:
		var data UserCreatedEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventUserUpdated:
		var data UserUpdatedEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventUserDeleted:
		var data UserDeletedEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventUserAuthenticated:
		var data UserAuthenticatedEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventUserUnlinkedAccount:
		var data UserUnlinkedAccountEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventUserUpdatedAccount:
		var data UserUpdatedAccountEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventUserTransferredAccount:
		var data UserTransferredAccountEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventUserWalletCreated:
		var data UserWalletCreatedEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventWalletCreated:
		var data WalletCreatedEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventWalletTransferred:
		var data WalletTransferredEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventWalletFundsDeposited, WebhookEventWalletFundsWithdrawn:
		var data WalletFundsEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventWalletPrivateKeyExport, WebhookEventWalletRecoverySetup, WebhookEventWalletRecovered:
		var data WalletSecurityEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventTransactionCreated, WebhookEventTransactionBroadcasted, WebhookEventTransactionConfirmed,
		WebhookEventTransactionCompleted, WebhookEventTransactionExecutionReverted, WebhookEventTransactionStillPending,
		WebhookEventTransactionFailed, WebhookEventTransactionReplaced, WebhookEventTransactionProviderError:
		var data TransactionEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case WebhookEventMFAEnabled, WebhookEventMFADisabled:
		var data MFAEvent
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return nil, err
		}
		return &data, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", e.Type)
	}
}

// GetUserCreatedData returns the event data as UserCreatedEvent.
func (e *WebhookEvent) GetUserCreatedData() (*UserCreatedEvent, error) {
	if e.Type != WebhookEventUserCreated {
		return nil, fmt.Errorf("event type is %s, not user.created", e.Type)
	}
	var data UserCreatedEvent
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetUserUpdatedData returns the event data as UserUpdatedEvent.
func (e *WebhookEvent) GetUserUpdatedData() (*UserUpdatedEvent, error) {
	if e.Type != WebhookEventUserUpdated {
		return nil, fmt.Errorf("event type is %s, not user.updated", e.Type)
	}
	var data UserUpdatedEvent
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetUserDeletedData returns the event data as UserDeletedEvent.
func (e *WebhookEvent) GetUserDeletedData() (*UserDeletedEvent, error) {
	if e.Type != WebhookEventUserDeleted {
		return nil, fmt.Errorf("event type is %s, not user.deleted", e.Type)
	}
	var data UserDeletedEvent
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetWalletCreatedData returns the event data as WalletCreatedEvent.
func (e *WebhookEvent) GetWalletCreatedData() (*WalletCreatedEvent, error) {
	if e.Type != WebhookEventWalletCreated {
		return nil, fmt.Errorf("event type is %s, not wallet.created", e.Type)
	}
	var data WalletCreatedEvent
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetTransactionData returns the event data as TransactionEvent.
func (e *WebhookEvent) GetTransactionData() (*TransactionEvent, error) {
	if e.Type != WebhookEventTransactionCreated &&
	   e.Type != WebhookEventTransactionCompleted &&
	   e.Type != WebhookEventTransactionFailed {
		return nil, fmt.Errorf("event type is %s, not a transaction event", e.Type)
	}
	var data TransactionEvent
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
