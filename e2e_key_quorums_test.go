package privy

import (
	"context"
	"testing"
)

// ============================================
// Key Quorums Service E2E Tests
// ============================================

func TestE2E_KeyQuorums_Create(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	kq, err := client.KeyQuorums().Create(ctx, &CreateKeyQuorumRequest{
		PublicKey: "0x04abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
	})
	if err != nil {
		t.Fatalf("Failed to create key quorum: %v", err)
	}

	if kq.ID == "" {
		t.Error("Expected key quorum ID to be set")
	}

	if kq.PublicKey == "" {
		t.Error("Expected public key to be set")
	}

	if kq.CreatedAt == 0 {
		t.Error("Expected created_at to be set")
	}
}

func TestE2E_KeyQuorums_Get(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.KeyQuorums().Create(ctx, &CreateKeyQuorumRequest{
		PublicKey: "0x04test-public-key",
	})
	if err != nil {
		t.Fatalf("Failed to create key quorum: %v", err)
	}

	kq, err := client.KeyQuorums().Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get key quorum: %v", err)
	}

	if kq.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, kq.ID)
	}

	if kq.PublicKey != created.PublicKey {
		t.Errorf("Expected public key '%s', got '%s'", created.PublicKey, kq.PublicKey)
	}
}

func TestE2E_KeyQuorums_GetNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.KeyQuorums().Get(ctx, "nonexistent-kq")
	if err == nil {
		t.Error("Expected error for non-existent key quorum")
	}
}

func TestE2E_KeyQuorums_Update(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.KeyQuorums().Create(ctx, &CreateKeyQuorumRequest{
		PublicKey: "0x04original-key",
	})
	if err != nil {
		t.Fatalf("Failed to create key quorum: %v", err)
	}

	updated, err := client.KeyQuorums().Update(ctx, created.ID, &UpdateKeyQuorumRequest{
		PublicKey: "0x04updated-key",
	})
	if err != nil {
		t.Fatalf("Failed to update key quorum: %v", err)
	}

	if updated.PublicKey != "0x04updated-key" {
		t.Errorf("Expected public key '0x04updated-key', got '%s'", updated.PublicKey)
	}
}

func TestE2E_KeyQuorums_UpdateNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.KeyQuorums().Update(ctx, "nonexistent-kq", &UpdateKeyQuorumRequest{
		PublicKey: "0x04new-key",
	})
	if err == nil {
		t.Error("Expected error for updating non-existent key quorum")
	}
}

func TestE2E_KeyQuorums_Delete(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.KeyQuorums().Create(ctx, &CreateKeyQuorumRequest{
		PublicKey: "0x04delete-me",
	})
	if err != nil {
		t.Fatalf("Failed to create key quorum: %v", err)
	}

	err = client.KeyQuorums().Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to delete key quorum: %v", err)
	}

	// Verify deletion
	_, err = client.KeyQuorums().Get(ctx, created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted key quorum")
	}
}

func TestE2E_KeyQuorums_DeleteNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	err := client.KeyQuorums().Delete(ctx, "nonexistent-kq")
	if err == nil {
		t.Error("Expected error for deleting non-existent key quorum")
	}
}

func TestE2E_KeyQuorums_MultipleQuorums(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create multiple key quorums
	var quorums []*KeyQuorum
	for i := 0; i < 5; i++ {
		kq, err := client.KeyQuorums().Create(ctx, &CreateKeyQuorumRequest{
			PublicKey: "0x04key-" + string(rune('a'+i)),
		})
		if err != nil {
			t.Fatalf("Failed to create key quorum %d: %v", i, err)
		}
		quorums = append(quorums, kq)
	}

	// Verify all were created
	for _, kq := range quorums {
		fetched, err := client.KeyQuorums().Get(ctx, kq.ID)
		if err != nil {
			t.Fatalf("Failed to get key quorum: %v", err)
		}
		if fetched.ID != kq.ID {
			t.Errorf("Expected ID %s, got %s", kq.ID, fetched.ID)
		}
	}
}

func TestE2E_KeyQuorums_Lifecycle(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create
	kq, err := client.KeyQuorums().Create(ctx, &CreateKeyQuorumRequest{
		PublicKey: "0x04lifecycle-key",
	})
	if err != nil {
		t.Fatalf("Failed to create key quorum: %v", err)
	}

	// Get
	_, err = client.KeyQuorums().Get(ctx, kq.ID)
	if err != nil {
		t.Fatalf("Failed to get key quorum: %v", err)
	}

	// Update
	updated, err := client.KeyQuorums().Update(ctx, kq.ID, &UpdateKeyQuorumRequest{
		PublicKey: "0x04updated-lifecycle-key",
	})
	if err != nil {
		t.Fatalf("Failed to update key quorum: %v", err)
	}

	if updated.PublicKey != "0x04updated-lifecycle-key" {
		t.Errorf("Expected updated public key")
	}

	// Delete
	err = client.KeyQuorums().Delete(ctx, kq.ID)
	if err != nil {
		t.Fatalf("Failed to delete key quorum: %v", err)
	}

	// Verify deleted
	_, err = client.KeyQuorums().Get(ctx, kq.ID)
	if err == nil {
		t.Error("Expected error when getting deleted key quorum")
	}
}
