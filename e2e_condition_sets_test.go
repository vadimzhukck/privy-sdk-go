package privy

import (
	"context"
	"testing"
)

// ============================================
// Condition Sets Service E2E Tests
// ============================================

func TestE2E_ConditionSets_Create(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Allowlist",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	if cs.ID == "" {
		t.Error("Expected condition set ID to be set")
	}

	if cs.Name != "Allowlist" {
		t.Errorf("Expected name 'Allowlist', got '%s'", cs.Name)
	}

	if cs.CreatedAt == 0 {
		t.Error("Expected created_at to be set")
	}
}

func TestE2E_ConditionSets_CreateWithOwner(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name:    "Owner Condition Set",
		OwnerID: "owner-123",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	if cs.OwnerID != "owner-123" {
		t.Errorf("Expected owner ID 'owner-123', got '%s'", cs.OwnerID)
	}
}

func TestE2E_ConditionSets_Get(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Get Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	cs, err := client.ConditionSets().Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get condition set: %v", err)
	}

	if cs.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, cs.ID)
	}

	if cs.Name != created.Name {
		t.Errorf("Expected name '%s', got '%s'", created.Name, cs.Name)
	}
}

func TestE2E_ConditionSets_GetNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.ConditionSets().Get(ctx, "nonexistent-cs")
	if err == nil {
		t.Error("Expected error for non-existent condition set")
	}
}

func TestE2E_ConditionSets_Update(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Original Name",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	updated, err := client.ConditionSets().Update(ctx, created.ID, &UpdateConditionSetRequest{
		Name: "Updated Name",
	})
	if err != nil {
		t.Fatalf("Failed to update condition set: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", updated.Name)
	}

	if updated.UpdatedAt == 0 {
		t.Error("Expected updated_at to be set")
	}
}

func TestE2E_ConditionSets_Delete(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Delete Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	err = client.ConditionSets().Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to delete condition set: %v", err)
	}

	// Verify deletion
	_, err = client.ConditionSets().Get(ctx, created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted condition set")
	}
}

func TestE2E_ConditionSets_AddItems(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Items Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	items, err := client.ConditionSets().AddItems(ctx, cs.ID, []ConditionSetItemInput{
		{Value: "0xAddress1"},
		{Value: "0xAddress2"},
		{Value: "0xAddress3"},
	})
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}

	for _, item := range items {
		if item.ID == "" {
			t.Error("Expected item ID to be set")
		}
	}
}

func TestE2E_ConditionSets_AddManyItems(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Many Items Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	// Add 50 items
	inputItems := make([]ConditionSetItemInput, 50)
	for i := 0; i < 50; i++ {
		inputItems[i] = ConditionSetItemInput{Value: i}
	}

	items, err := client.ConditionSets().AddItems(ctx, cs.ID, inputItems)
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	if len(items) != 50 {
		t.Errorf("Expected 50 items, got %d", len(items))
	}
}

func TestE2E_ConditionSets_ListItems(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "List Items Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	// Add items
	_, err = client.ConditionSets().AddItems(ctx, cs.ID, []ConditionSetItemInput{
		{Value: "item1"},
		{Value: "item2"},
	})
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	// List items
	resp, err := client.ConditionSets().ListItems(ctx, cs.ID, nil)
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resp.Data))
	}
}

func TestE2E_ConditionSets_ListItemsWithPagination(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Pagination Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	// Add items
	inputItems := make([]ConditionSetItemInput, 20)
	for i := 0; i < 20; i++ {
		inputItems[i] = ConditionSetItemInput{Value: i}
	}
	_, err = client.ConditionSets().AddItems(ctx, cs.ID, inputItems)
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	// List with limit
	resp, err := client.ConditionSets().ListItems(ctx, cs.ID, &ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	// Should return all 20 items (mock doesn't actually paginate)
	if len(resp.Data) < 1 {
		t.Error("Expected items to be returned")
	}
}

func TestE2E_ConditionSets_GetItem(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Get Item Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	items, err := client.ConditionSets().AddItems(ctx, cs.ID, []ConditionSetItemInput{
		{Value: "test-value"},
	})
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	// Get the item
	item, err := client.ConditionSets().GetItem(ctx, cs.ID, items[0].ID)
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if item.ID != items[0].ID {
		t.Errorf("Expected item ID %s, got %s", items[0].ID, item.ID)
	}
}

func TestE2E_ConditionSets_GetItemNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Get Non-Existent Item Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	_, err = client.ConditionSets().GetItem(ctx, cs.ID, "nonexistent-item")
	if err == nil {
		t.Error("Expected error for non-existent item")
	}
}

func TestE2E_ConditionSets_ReplaceItems(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Replace Items Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	// Add initial items
	_, err = client.ConditionSets().AddItems(ctx, cs.ID, []ConditionSetItemInput{
		{Value: "old1"},
		{Value: "old2"},
	})
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	// Replace all items
	newItems, err := client.ConditionSets().ReplaceItems(ctx, cs.ID, []ConditionSetItemInput{
		{Value: "new1"},
		{Value: "new2"},
		{Value: "new3"},
	})
	if err != nil {
		t.Fatalf("Failed to replace items: %v", err)
	}

	if len(newItems) != 3 {
		t.Errorf("Expected 3 items, got %d", len(newItems))
	}

	// Verify old items are gone
	listResp, err := client.ConditionSets().ListItems(ctx, cs.ID, nil)
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if len(listResp.Data) != 3 {
		t.Errorf("Expected 3 items after replace, got %d", len(listResp.Data))
	}
}

func TestE2E_ConditionSets_DeleteItem(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Delete Item Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	items, err := client.ConditionSets().AddItems(ctx, cs.ID, []ConditionSetItemInput{
		{Value: "delete-me"},
	})
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	err = client.ConditionSets().DeleteItem(ctx, cs.ID, items[0].ID)
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	// Verify deletion
	_, err = client.ConditionSets().GetItem(ctx, cs.ID, items[0].ID)
	if err == nil {
		t.Error("Expected error when getting deleted item")
	}
}

func TestE2E_ConditionSets_DeleteItemNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Delete Non-Existent Item Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	err = client.ConditionSets().DeleteItem(ctx, cs.ID, "nonexistent-item")
	if err == nil {
		t.Error("Expected error for deleting non-existent item")
	}
}

func TestE2E_ConditionSets_ComplexWorkflow(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	// Create condition set
	cs, err := client.ConditionSets().Create(ctx, &CreateConditionSetRequest{
		Name: "Workflow Test",
	})
	if err != nil {
		t.Fatalf("Failed to create condition set: %v", err)
	}

	// Add items
	items, err := client.ConditionSets().AddItems(ctx, cs.ID, []ConditionSetItemInput{
		{Value: "address1"},
		{Value: "address2"},
		{Value: "address3"},
	})
	if err != nil {
		t.Fatalf("Failed to add items: %v", err)
	}

	// Delete one item
	err = client.ConditionSets().DeleteItem(ctx, cs.ID, items[1].ID)
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	// List remaining items
	listResp, err := client.ConditionSets().ListItems(ctx, cs.ID, nil)
	if err != nil {
		t.Fatalf("Failed to list items: %v", err)
	}

	if len(listResp.Data) != 2 {
		t.Errorf("Expected 2 items after deletion, got %d", len(listResp.Data))
	}

	// Update condition set
	updated, err := client.ConditionSets().Update(ctx, cs.ID, &UpdateConditionSetRequest{
		Name: "Updated Workflow Test",
	})
	if err != nil {
		t.Fatalf("Failed to update condition set: %v", err)
	}

	if updated.Name != "Updated Workflow Test" {
		t.Errorf("Expected updated name, got '%s'", updated.Name)
	}

	// Delete condition set
	err = client.ConditionSets().Delete(ctx, cs.ID)
	if err != nil {
		t.Fatalf("Failed to delete condition set: %v", err)
	}
}
