package privy

import (
	"context"
	"testing"
)

// ============================================
// Policies Service E2E Tests
// ============================================

func TestE2E_Policies_Create(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Transfer Limit Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	if policy.ID == "" {
		t.Error("Expected policy ID to be set")
	}

	if policy.Name != "Transfer Limit Policy" {
		t.Errorf("Expected name 'Transfer Limit Policy', got '%s'", policy.Name)
	}

	if policy.CreatedAt == 0 {
		t.Error("Expected created_at to be set")
	}
}

func TestE2E_Policies_CreateWithRules(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Complex Policy",
		Rules: []PolicyRule{
			{
				ID:     "rule-1",
				Action: "allow",
				Conditions: []RuleCondition{
					{Type: "max_value", Value: "1000000000000000000"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create policy with rules: %v", err)
	}

	if len(policy.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(policy.Rules))
	}
}

func TestE2E_Policies_Get(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Get Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	policy, err := client.Policies().Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if policy.ID != created.ID {
		t.Errorf("Expected policy ID %s, got %s", created.ID, policy.ID)
	}

	if policy.Name != created.Name {
		t.Errorf("Expected name '%s', got '%s'", created.Name, policy.Name)
	}
}

func TestE2E_Policies_GetNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	_, err := client.Policies().Get(ctx, "nonexistent-policy")
	if err == nil {
		t.Error("Expected error for non-existent policy")
	}
}

func TestE2E_Policies_Update(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Original Name",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	updated, err := client.Policies().Update(ctx, created.ID, &UpdatePolicyRequest{
		Name: "Updated Name",
	})
	if err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", updated.Name)
	}

	if updated.UpdatedAt == 0 {
		t.Error("Expected updated_at to be set")
	}
}

func TestE2E_Policies_Delete(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	created, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Delete Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	err = client.Policies().Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// Verify deletion
	_, err = client.Policies().Get(ctx, created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted policy")
	}
}

func TestE2E_Policies_DeleteNonExistent(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	err := client.Policies().Delete(ctx, "nonexistent-policy")
	if err == nil {
		t.Error("Expected error for deleting non-existent policy")
	}
}

func TestE2E_Policies_AddRule(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Rule Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	rule, err := client.Policies().AddRule(ctx, policy.ID, &CreateRuleRequest{
		Action: "allow",
		Conditions: []RuleCondition{
			{Type: "max_value", Value: "5000000000000000000"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	if rule.ID == "" {
		t.Error("Expected rule ID to be set")
	}

	if rule.Action != "allow" {
		t.Errorf("Expected action 'allow', got '%s'", rule.Action)
	}
}

func TestE2E_Policies_GetRule(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Get Rule Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	addedRule, err := client.Policies().AddRule(ctx, policy.ID, &CreateRuleRequest{
		Action: "deny",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	rule, err := client.Policies().GetRule(ctx, policy.ID, addedRule.ID)
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	if rule.ID != addedRule.ID {
		t.Errorf("Expected rule ID %s, got %s", addedRule.ID, rule.ID)
	}

	if rule.Action != addedRule.Action {
		t.Errorf("Expected action '%s', got '%s'", addedRule.Action, rule.Action)
	}
}

func TestE2E_Policies_UpdateRule(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Update Rule Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	rule, err := client.Policies().AddRule(ctx, policy.ID, &CreateRuleRequest{
		Action: "allow",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	updated, err := client.Policies().UpdateRule(ctx, policy.ID, rule.ID, &UpdateRuleRequest{
		Action: "deny",
	})
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	if updated.Action != "deny" {
		t.Errorf("Expected action 'deny', got '%s'", updated.Action)
	}
}

func TestE2E_Policies_DeleteRule(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Delete Rule Test Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	rule, err := client.Policies().AddRule(ctx, policy.ID, &CreateRuleRequest{
		Action: "allow",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	err = client.Policies().DeleteRule(ctx, policy.ID, rule.ID)
	if err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}

	// Verify deletion
	_, err = client.Policies().GetRule(ctx, policy.ID, rule.ID)
	if err == nil {
		t.Error("Expected error when getting deleted rule")
	}
}

func TestE2E_Policies_MultipleRules(t *testing.T) {
	client, server, _ := setupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	policy, err := client.Policies().Create(ctx, &CreatePolicyRequest{
		Name: "Multiple Rules Policy",
	})
	if err != nil {
		t.Fatalf("Failed to create policy: %v", err)
	}

	// Add multiple rules
	actions := []string{"allow", "deny", "require_approval"}
	for _, action := range actions {
		_, err := client.Policies().AddRule(ctx, policy.ID, &CreateRuleRequest{
			Action: action,
		})
		if err != nil {
			t.Fatalf("Failed to add rule with action %s: %v", action, err)
		}
	}

	// Verify all rules were added
	updatedPolicy, err := client.Policies().Get(ctx, policy.ID)
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}

	if len(updatedPolicy.Rules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(updatedPolicy.Rules))
	}
}
