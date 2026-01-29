package privy

import (
	"context"
	"fmt"
)

// PoliciesService handles policy-related operations.
type PoliciesService struct {
	client *Client
}

// CreatePolicyRequest represents a request to create a new policy.
type CreatePolicyRequest struct {
	Version   string       `json:"version"`    // Must be "1.0"
	ChainType ChainType    `json:"chain_type"` // Required: ethereum, solana, etc.
	Name      string       `json:"name,omitempty"`
	Rules     []PolicyRule `json:"rules"` // Required
}

// UpdatePolicyRequest represents a request to update a policy.
type UpdatePolicyRequest struct {
	Name  string       `json:"name,omitempty"`
	Rules []PolicyRule `json:"rules,omitempty"`
}

// CreateRuleRequest represents a request to add a rule to a policy.
type CreateRuleRequest struct {
	Action     string          `json:"action"`
	Conditions []RuleCondition `json:"conditions,omitempty"`
}

// UpdateRuleRequest represents a request to update a rule.
type UpdateRuleRequest struct {
	Action     string          `json:"action,omitempty"`
	Conditions []RuleCondition `json:"conditions,omitempty"`
}

// Create creates a new policy.
func (s *PoliciesService) Create(ctx context.Context, req *CreatePolicyRequest) (*Policy, error) {
	u := fmt.Sprintf("%s/policies", s.client.baseURL)

	var policy Policy
	if err := s.client.doRequest(ctx, "POST", u, req, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// Get retrieves a policy by its ID.
func (s *PoliciesService) Get(ctx context.Context, policyID string) (*Policy, error) {
	u := fmt.Sprintf("%s/policies/%s", s.client.baseURL, policyID)

	var policy Policy
	if err := s.client.doRequest(ctx, "GET", u, nil, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// Update updates a policy.
func (s *PoliciesService) Update(ctx context.Context, policyID string, req *UpdatePolicyRequest) (*Policy, error) {
	u := fmt.Sprintf("%s/policies/%s", s.client.baseURL, policyID)

	var policy Policy
	if err := s.client.doRequest(ctx, "PATCH", u, req, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// Delete deletes a policy.
func (s *PoliciesService) Delete(ctx context.Context, policyID string) error {
	u := fmt.Sprintf("%s/policies/%s", s.client.baseURL, policyID)
	return s.client.doRequest(ctx, "DELETE", u, nil, nil)
}

// AddRule adds a rule to a policy.
func (s *PoliciesService) AddRule(ctx context.Context, policyID string, req *CreateRuleRequest) (*PolicyRule, error) {
	u := fmt.Sprintf("%s/policies/%s/rules", s.client.baseURL, policyID)

	var rule PolicyRule
	if err := s.client.doRequest(ctx, "POST", u, req, &rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

// GetRule retrieves a specific rule from a policy.
func (s *PoliciesService) GetRule(ctx context.Context, policyID, ruleID string) (*PolicyRule, error) {
	u := fmt.Sprintf("%s/policies/%s/rules/%s", s.client.baseURL, policyID, ruleID)

	var rule PolicyRule
	if err := s.client.doRequest(ctx, "GET", u, nil, &rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

// UpdateRule updates a rule in a policy.
func (s *PoliciesService) UpdateRule(ctx context.Context, policyID, ruleID string, req *UpdateRuleRequest) (*PolicyRule, error) {
	u := fmt.Sprintf("%s/policies/%s/rules/%s", s.client.baseURL, policyID, ruleID)

	var rule PolicyRule
	if err := s.client.doRequest(ctx, "PATCH", u, req, &rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

// DeleteRule deletes a rule from a policy.
func (s *PoliciesService) DeleteRule(ctx context.Context, policyID, ruleID string) error {
	u := fmt.Sprintf("%s/policies/%s/rules/%s", s.client.baseURL, policyID, ruleID)
	return s.client.doRequest(ctx, "DELETE", u, nil, nil)
}
