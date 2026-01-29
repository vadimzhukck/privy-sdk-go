package privy

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewWebhookHandler(t *testing.T) {
	// Test with raw secret
	handler := NewWebhookHandler("my-secret")
	if handler.signingSecret != "my-secret" {
		t.Errorf("Expected secret 'my-secret', got '%s'", handler.signingSecret)
	}

	// Test with whsec_ prefix
	handler = NewWebhookHandler("whsec_my-secret")
	if handler.signingSecret != "my-secret" {
		t.Errorf("Expected secret 'my-secret' (without prefix), got '%s'", handler.signingSecret)
	}

	if handler.toleranceSeconds != DefaultWebhookTolerance {
		t.Errorf("Expected default tolerance %d, got %d", DefaultWebhookTolerance, handler.toleranceSeconds)
	}
}

func TestWebhookHandler_WithTolerance(t *testing.T) {
	handler := NewWebhookHandler("secret").WithTolerance(600)
	if handler.toleranceSeconds != 600 {
		t.Errorf("Expected tolerance 600, got %d", handler.toleranceSeconds)
	}
}

func TestWebhookHandler_OnEvent(t *testing.T) {
	handler := NewWebhookHandler("secret")

	handler.OnEvent(WebhookEventUserCreated, func(e WebhookEvent) {
		// Handler registered
	})

	if len(handler.handlers[WebhookEventUserCreated]) != 1 {
		t.Error("Expected 1 handler for user.created event")
	}
}

func TestWebhookHandler_TypedHandlers(t *testing.T) {
	handler := NewWebhookHandler("secret")

	var userCreated *UserCreatedEvent
	var userUpdated *UserUpdatedEvent
	var userDeleted *UserDeletedEvent
	var walletCreated *WalletCreatedEvent
	var txCreated *TransactionEvent

	handler.OnUserCreated(func(e *UserCreatedEvent) {
		userCreated = e
	})
	handler.OnUserUpdated(func(e *UserUpdatedEvent) {
		userUpdated = e
	})
	handler.OnUserDeleted(func(e *UserDeletedEvent) {
		userDeleted = e
	})
	handler.OnWalletCreated(func(e *WalletCreatedEvent) {
		walletCreated = e
	})
	handler.OnTransactionCreated(func(e *TransactionEvent) {
		txCreated = e
	})

	// Verify handlers are registered
	if len(handler.handlers[WebhookEventUserCreated]) != 1 {
		t.Error("Expected 1 handler for user.created")
	}
	if len(handler.handlers[WebhookEventUserUpdated]) != 1 {
		t.Error("Expected 1 handler for user.updated")
	}
	if len(handler.handlers[WebhookEventUserDeleted]) != 1 {
		t.Error("Expected 1 handler for user.deleted")
	}
	if len(handler.handlers[WebhookEventWalletCreated]) != 1 {
		t.Error("Expected 1 handler for wallet.created")
	}
	if len(handler.handlers[WebhookEventTransactionCreated]) != 1 {
		t.Error("Expected 1 handler for transaction.created")
	}

	// Simulate calling handlers
	testData := UserCreatedEvent{UserID: "did:privy:123"}
	dataBytes, _ := json.Marshal(testData)
	event := WebhookEvent{
		Type: WebhookEventUserCreated,
		Data: dataBytes,
	}

	for _, h := range handler.handlers[WebhookEventUserCreated] {
		h(event)
	}

	if userCreated == nil || userCreated.UserID != "did:privy:123" {
		t.Error("User created handler was not called correctly")
	}

	// These should still be nil since their events weren't triggered
	if userUpdated != nil || userDeleted != nil || walletCreated != nil || txCreated != nil {
		t.Error("Other handlers should not have been called")
	}
}

func TestWebhookEvent_ParseEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType WebhookEventType
		data      any
		wantType  string
	}{
		{
			name:      "user.created",
			eventType: WebhookEventUserCreated,
			data:      UserCreatedEvent{UserID: "123"},
			wantType:  "*privy.UserCreatedEvent",
		},
		{
			name:      "user.updated",
			eventType: WebhookEventUserUpdated,
			data:      UserUpdatedEvent{UserID: "123"},
			wantType:  "*privy.UserUpdatedEvent",
		},
		{
			name:      "wallet.created",
			eventType: WebhookEventWalletCreated,
			data:      WalletCreatedEvent{WalletID: "123"},
			wantType:  "*privy.WalletCreatedEvent",
		},
		{
			name:      "transaction.completed",
			eventType: WebhookEventTransactionCompleted,
			data:      TransactionEvent{TransactionID: "123"},
			wantType:  "*privy.TransactionEvent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataBytes, _ := json.Marshal(tt.data)
			event := &WebhookEvent{
				Type: tt.eventType,
				Data: dataBytes,
			}

			parsed, err := event.ParseEvent()
			if err != nil {
				t.Fatalf("ParseEvent() error = %v", err)
			}

			gotType := fmt.Sprintf("%T", parsed)
			if gotType != tt.wantType {
				t.Errorf("ParseEvent() type = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestWebhookEvent_GetTypedData(t *testing.T) {
	// Test GetUserCreatedData
	userCreatedData := UserCreatedEvent{UserID: "did:privy:123", CreatedAt: 123456789}
	dataBytes, _ := json.Marshal(userCreatedData)
	event := &WebhookEvent{Type: WebhookEventUserCreated, Data: dataBytes}

	userData, err := event.GetUserCreatedData()
	if err != nil {
		t.Fatalf("GetUserCreatedData() error = %v", err)
	}
	if userData.UserID != "did:privy:123" {
		t.Errorf("GetUserCreatedData().UserID = %v, want %v", userData.UserID, "did:privy:123")
	}

	// Test wrong event type
	event.Type = WebhookEventUserDeleted
	_, err = event.GetUserCreatedData()
	if err == nil {
		t.Error("GetUserCreatedData() should error for wrong event type")
	}

	// Test GetWalletCreatedData
	walletData := WalletCreatedEvent{WalletID: "wallet-123", Address: "0x123"}
	dataBytes, _ = json.Marshal(walletData)
	walletEvent := &WebhookEvent{Type: WebhookEventWalletCreated, Data: dataBytes}

	wallet, err := walletEvent.GetWalletCreatedData()
	if err != nil {
		t.Fatalf("GetWalletCreatedData() error = %v", err)
	}
	if wallet.WalletID != "wallet-123" {
		t.Errorf("GetWalletCreatedData().WalletID = %v, want %v", wallet.WalletID, "wallet-123")
	}

	// Test GetTransactionData
	txData := TransactionEvent{TransactionID: "tx-123", Status: "completed"}
	dataBytes, _ = json.Marshal(txData)
	txEvent := &WebhookEvent{Type: WebhookEventTransactionCompleted, Data: dataBytes}

	tx, err := txEvent.GetTransactionData()
	if err != nil {
		t.Fatalf("GetTransactionData() error = %v", err)
	}
	if tx.TransactionID != "tx-123" {
		t.Errorf("GetTransactionData().TransactionID = %v, want %v", tx.TransactionID, "tx-123")
	}
}

func TestWebhookHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	handler := NewWebhookHandler("secret")

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestWebhookHandler_ServeHTTP_MissingHeaders(t *testing.T) {
	handler := NewWebhookHandler("secret")

	body := []byte(`{"type": "user.created", "data": {}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func createSignedWebhookRequest(t *testing.T, secret string, body []byte) *http.Request {
	t.Helper()

	webhookID := "msg_123"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// Decode secret if base64
	secretBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		secretBytes = []byte(secret)
	}

	// Create signature
	signedPayload := fmt.Sprintf("%s.%s.%s", webhookID, timestamp, string(body))
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedPayload))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(WebhookIDHeader, webhookID)
	req.Header.Set(WebhookTimestampHeader, timestamp)
	req.Header.Set(WebhookSignatureHeader, fmt.Sprintf("v1,%s", signature))

	return req
}

func TestWebhookHandler_VerifyAndParse_ValidSignature(t *testing.T) {
	secret := "test-secret"
	handler := NewWebhookHandler(secret)

	eventData := map[string]any{
		"id":         "evt_123",
		"type":       "user.created",
		"created_at": time.Now().Unix(),
		"data": map[string]any{
			"user_id": "did:privy:abc123",
		},
	}
	body, _ := json.Marshal(eventData)

	req := createSignedWebhookRequest(t, secret, body)

	event, err := handler.VerifyAndParse(req)
	if err != nil {
		t.Fatalf("VerifyAndParse() error = %v", err)
	}

	if event.ID != "evt_123" {
		t.Errorf("Event ID = %v, want %v", event.ID, "evt_123")
	}
	if event.Type != WebhookEventUserCreated {
		t.Errorf("Event Type = %v, want %v", event.Type, WebhookEventUserCreated)
	}
}

func TestWebhookHandler_VerifyAndParse_ExpiredTimestamp(t *testing.T) {
	secret := "test-secret"
	handler := NewWebhookHandler(secret).WithTolerance(60) // 1 minute tolerance

	webhookID := "msg_123"
	// Timestamp from 10 minutes ago
	timestamp := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())

	body := []byte(`{"type": "user.created", "data": {}}`)

	secretBytes := []byte(secret)
	signedPayload := fmt.Sprintf("%s.%s.%s", webhookID, timestamp, string(body))
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedPayload))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(WebhookIDHeader, webhookID)
	req.Header.Set(WebhookTimestampHeader, timestamp)
	req.Header.Set(WebhookSignatureHeader, fmt.Sprintf("v1,%s", signature))

	_, err := handler.VerifyAndParse(req)
	if err != ErrWebhookTimestampExpired {
		t.Errorf("Expected ErrWebhookTimestampExpired, got %v", err)
	}
}

func TestWebhookHandler_VerifyAndParse_InvalidSignature(t *testing.T) {
	handler := NewWebhookHandler("correct-secret")

	webhookID := "msg_123"
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	body := []byte(`{"type": "user.created", "data": {}}`)

	// Sign with wrong secret
	wrongSecret := []byte("wrong-secret")
	signedPayload := fmt.Sprintf("%s.%s.%s", webhookID, timestamp, string(body))
	mac := hmac.New(sha256.New, wrongSecret)
	mac.Write([]byte(signedPayload))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(WebhookIDHeader, webhookID)
	req.Header.Set(WebhookTimestampHeader, timestamp)
	req.Header.Set(WebhookSignatureHeader, fmt.Sprintf("v1,%s", signature))

	_, err := handler.VerifyAndParse(req)
	if err != ErrInvalidWebhookSignature {
		t.Errorf("Expected ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestWebhookHandler_HandleRequest_DispatchesEvents(t *testing.T) {
	secret := "test-secret"
	handler := NewWebhookHandler(secret)

	var receivedUserID string
	handler.OnUserCreated(func(e *UserCreatedEvent) {
		receivedUserID = e.UserID
	})

	eventData := map[string]any{
		"id":         "evt_123",
		"type":       "user.created",
		"created_at": time.Now().Unix(),
		"data": map[string]any{
			"user_id": "did:privy:test123",
		},
	}
	body, _ := json.Marshal(eventData)

	req := createSignedWebhookRequest(t, secret, body)

	event, err := handler.HandleRequest(req)
	if err != nil {
		t.Fatalf("HandleRequest() error = %v", err)
	}

	if event.ID != "evt_123" {
		t.Errorf("Event ID = %v, want %v", event.ID, "evt_123")
	}

	if receivedUserID != "did:privy:test123" {
		t.Errorf("Handler received user_id = %v, want %v", receivedUserID, "did:privy:test123")
	}
}
