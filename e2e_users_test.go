package privy

import (
	"context"
	"net/http/httptest"
	"testing"
)

func setupTestServer(t *testing.T) (*Client, *httptest.Server, *MockPrivyServer) {
	t.Helper()
	mock := NewMockPrivyServer()
	server := httptest.NewServer(mock)
	client := NewClient(
		"test-app-id",
		"test-app-secret",
		WithBaseURL(server.URL+"/v1"),
		WithAuthURL(server.URL+"/api/v1"),
	)
	return client, server, mock
}

// ============================================
// Users Service E2E Tests
// ============================================

func TestE2E_Users_CreateAndGet(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a user with email
	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{
				Type:    LinkedAccountTypeEmail,
				Address: "[email protected]",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == "" {
		t.Error("Expected user ID to be set")
	}

	if len(user.LinkedAccounts) != 1 {
		t.Fatalf("Expected 1 linked account, got %d", len(user.LinkedAccounts))
	}

	if user.LinkedAccounts[0].Type != LinkedAccountTypeEmail {
		t.Errorf("Expected email linked account, got %s", user.LinkedAccounts[0].Type)
	}

	// Get the user
	fetchedUser, err := client.Users().Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if fetchedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, fetchedUser.ID)
	}
}

func TestE2E_Users_CreateWithEmbeddedWallet(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{
				Type:    LinkedAccountTypeEmail,
				Address: "[email protected]",
			},
		},
		CreateEthereumWallet: true,
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Should have email and wallet linked accounts
	if len(user.LinkedAccounts) < 2 {
		t.Fatalf("Expected at least 2 linked accounts (email + wallet), got %d", len(user.LinkedAccounts))
	}

	hasWallet := false
	for _, la := range user.LinkedAccounts {
		if la.Type == LinkedAccountTypeWallet {
			hasWallet = true
			if la.ChainType != ChainTypeEthereum {
				t.Errorf("Expected Ethereum wallet, got %s", la.ChainType)
			}
			break
		}
	}

	if !hasWallet {
		t.Error("Expected user to have a wallet linked account")
	}
}

func TestE2E_Users_CreateWithPhone(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{
				Type:        LinkedAccountTypePhone,
				PhoneNumber: "+1234567890",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.LinkedAccounts[0].Type != LinkedAccountTypePhone {
		t.Errorf("Expected phone linked account, got %s", user.LinkedAccounts[0].Type)
	}
}

func TestE2E_Users_CreateWithCustomMetadata(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	metadata := map[string]any{
		"tier":       "premium",
		"signupDate": "2024-01-01",
	}

	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: "[email protected]"},
		},
		CustomMetadata: metadata,
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.CustomMetadata == nil {
		t.Fatal("Expected custom metadata to be set")
	}

	if user.CustomMetadata["tier"] != "premium" {
		t.Errorf("Expected tier 'premium', got %v", user.CustomMetadata["tier"])
	}
}

func TestE2E_Users_List(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 3; i++ {
		_, err := client.Users().Create(ctx, &CreateUserRequest{
			LinkedAccounts: []LinkedAccountInput{
				{Type: LinkedAccountTypeEmail, Address: "user" + string(rune('a'+i)) + "@test.com"},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create user %d: %v", i, err)
		}
	}

	// List users
	resp, err := client.Users().List(ctx, &ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Errorf("Expected 3 users, got %d", len(resp.Data))
	}
}

func TestE2E_Users_Delete(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a user
	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: "[email protected]"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Delete the user
	err = client.Users().Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user is deleted
	_, err = client.Users().Get(ctx, user.ID)
	if err == nil {
		t.Error("Expected error when getting deleted user")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("Expected 404 status code, got %d", apiErr.StatusCode)
	}
}

func TestE2E_Users_GetByEmail(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	email := "[email protected]"

	// Create a user
	createdUser, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: email},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Find by email
	user, err := client.Users().GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if user.ID != createdUser.ID {
		t.Errorf("Expected user ID %s, got %s", createdUser.ID, user.ID)
	}
}

func TestE2E_Users_GetByEmail_NotFound(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Users().GetByEmail(ctx, "[email protected]")
	if err == nil {
		t.Error("Expected error for non-existent email")
	}
}

func TestE2E_Users_GetByPhone(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	phone := "+1987654321"

	// Create a user
	createdUser, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypePhone, PhoneNumber: phone},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Find by phone
	user, err := client.Users().GetByPhone(ctx, phone)
	if err != nil {
		t.Fatalf("Failed to get user by phone: %v", err)
	}

	if user.ID != createdUser.ID {
		t.Errorf("Expected user ID %s, got %s", createdUser.ID, user.ID)
	}
}

func TestE2E_Users_GetByWalletAddress(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a user with embedded wallet
	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: "[email protected]"},
		},
		CreateEthereumWallet: true,
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Find wallet address
	var walletAddress string
	for _, la := range user.LinkedAccounts {
		if la.Type == LinkedAccountTypeWallet {
			walletAddress = la.Address
			break
		}
	}

	if walletAddress == "" {
		t.Fatal("User has no wallet address")
	}

	// Find by wallet
	foundUser, err := client.Users().GetByWalletAddress(ctx, walletAddress)
	if err != nil {
		t.Fatalf("Failed to get user by wallet address: %v", err)
	}

	if foundUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, foundUser.ID)
	}
}

func TestE2E_Users_UpdateMetadata(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create a user
	user, err := client.Users().Create(ctx, &CreateUserRequest{
		LinkedAccounts: []LinkedAccountInput{
			{Type: LinkedAccountTypeEmail, Address: "[email protected]"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update metadata
	newMetadata := map[string]any{
		"tier":       "enterprise",
		"trialEnded": true,
	}

	updatedUser, err := client.Users().UpdateMetadata(ctx, user.ID, newMetadata)
	if err != nil {
		t.Fatalf("Failed to update user metadata: %v", err)
	}

	if updatedUser.CustomMetadata["tier"] != "enterprise" {
		t.Errorf("Expected tier 'enterprise', got %v", updatedUser.CustomMetadata["tier"])
	}
}

func TestE2E_Users_GetNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Users().Get(ctx, "did:privy:nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 404 {
		t.Errorf("Expected 404 status code, got %d", apiErr.StatusCode)
	}
}

func TestE2E_Users_DeleteNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	err := client.Users().Delete(ctx, "did:privy:nonexistent")
	if err == nil {
		t.Error("Expected error for deleting non-existent user")
	}
}
