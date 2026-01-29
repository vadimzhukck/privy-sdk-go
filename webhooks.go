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
	WebhookEventUserCreated        WebhookEventType = "user.created"
	WebhookEventUserUpdated        WebhookEventType = "user.updated"
	WebhookEventUserDeleted        WebhookEventType = "user.deleted"
	WebhookEventUserLinkedAccount  WebhookEventType = "user.linked_account"
	WebhookEventUserAuthenticated  WebhookEventType = "user.authenticated"

	// Wallet events
	WebhookEventWalletCreated     WebhookEventType = "wallet.created"
	WebhookEventWalletTransferred WebhookEventType = "wallet.transferred"

	// Transaction events
	WebhookEventTransactionCreated   WebhookEventType = "transaction.created"
	WebhookEventTransactionCompleted WebhookEventType = "transaction.completed"
	WebhookEventTransactionFailed    WebhookEventType = "transaction.failed"
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

// TransactionEvent represents a transaction webhook event.
type TransactionEvent struct {
	TransactionID   string    `json:"transaction_id"`
	WalletID        string    `json:"wallet_id"`
	ChainType       ChainType `json:"chain_type"`
	TransactionHash string    `json:"transaction_hash,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       int64     `json:"created_at"`
	CompletedAt     int64     `json:"completed_at,omitempty"`
	FailedAt        int64     `json:"failed_at,omitempty"`
	Error           string    `json:"error,omitempty"`
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
	case WebhookEventTransactionCreated, WebhookEventTransactionCompleted, WebhookEventTransactionFailed:
		var data TransactionEvent
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
