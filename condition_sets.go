package privy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ConditionSetsService handles condition set operations.
type ConditionSetsService struct {
	client *Client
}

// CreateConditionSetRequest represents a request to create a condition set.
type CreateConditionSetRequest struct {
	Name    string       `json:"name,omitempty"`
	Owner   *WalletOwner `json:"owner,omitempty"`
	OwnerID string       `json:"owner_id,omitempty"`
}

// UpdateConditionSetRequest represents a request to update a condition set.
type UpdateConditionSetRequest struct {
	Name string `json:"name,omitempty"`
}

// AddConditionSetItemsRequest represents a request to add items to a condition set.
type AddConditionSetItemsRequest struct {
	Items []ConditionSetItemInput `json:"items"`
}

// ConditionSetItemInput represents input for a condition set item.
type ConditionSetItemInput struct {
	Value any `json:"value"`
}

// ReplaceConditionSetItemsRequest represents a request to replace all items in a condition set.
type ReplaceConditionSetItemsRequest struct {
	Items []ConditionSetItemInput `json:"items"`
}

// Create creates a new condition set.
func (s *ConditionSetsService) Create(ctx context.Context, req *CreateConditionSetRequest) (*ConditionSet, error) {
	u := fmt.Sprintf("%s/condition-sets", s.client.baseURL)

	var cs ConditionSet
	if err := s.client.doRequest(ctx, "POST", u, req, &cs); err != nil {
		return nil, err
	}

	return &cs, nil
}

// Get retrieves a condition set by its ID.
func (s *ConditionSetsService) Get(ctx context.Context, conditionSetID string) (*ConditionSet, error) {
	u := fmt.Sprintf("%s/condition-sets/%s", s.client.baseURL, conditionSetID)

	var cs ConditionSet
	if err := s.client.doRequest(ctx, "GET", u, nil, &cs); err != nil {
		return nil, err
	}

	return &cs, nil
}

// Update updates a condition set.
func (s *ConditionSetsService) Update(ctx context.Context, conditionSetID string, req *UpdateConditionSetRequest) (*ConditionSet, error) {
	u := fmt.Sprintf("%s/condition-sets/%s", s.client.baseURL, conditionSetID)

	var cs ConditionSet
	if err := s.client.doRequest(ctx, "PATCH", u, req, &cs); err != nil {
		return nil, err
	}

	return &cs, nil
}

// Delete deletes a condition set.
func (s *ConditionSetsService) Delete(ctx context.Context, conditionSetID string) error {
	u := fmt.Sprintf("%s/condition-sets/%s", s.client.baseURL, conditionSetID)
	return s.client.doRequest(ctx, "DELETE", u, nil, nil)
}

// AddItems adds items to a condition set (up to 100 items).
func (s *ConditionSetsService) AddItems(ctx context.Context, conditionSetID string, items []ConditionSetItemInput) ([]ConditionSetItem, error) {
	u := fmt.Sprintf("%s/condition-sets/%s/items", s.client.baseURL, conditionSetID)

	req := &AddConditionSetItemsRequest{Items: items}
	var result []ConditionSetItem
	if err := s.client.doRequest(ctx, "POST", u, req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ListItems lists items in a condition set with pagination.
func (s *ConditionSetsService) ListItems(ctx context.Context, conditionSetID string, opts *ListOptions) (*PaginatedResponse[ConditionSetItem], error) {
	u := fmt.Sprintf("%s/condition-sets/%s/items", s.client.baseURL, conditionSetID)

	if opts != nil {
		params := url.Values{}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if len(params) > 0 {
			u = u + "?" + params.Encode()
		}
	}

	var resp PaginatedResponse[ConditionSetItem]
	if err := s.client.doRequest(ctx, "GET", u, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetItem retrieves a specific item from a condition set.
func (s *ConditionSetsService) GetItem(ctx context.Context, conditionSetID, itemID string) (*ConditionSetItem, error) {
	u := fmt.Sprintf("%s/condition-sets/%s/items/%s", s.client.baseURL, conditionSetID, itemID)

	var item ConditionSetItem
	if err := s.client.doRequest(ctx, "GET", u, nil, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

// ReplaceItems replaces all items in a condition set (up to 100 items).
func (s *ConditionSetsService) ReplaceItems(ctx context.Context, conditionSetID string, items []ConditionSetItemInput) ([]ConditionSetItem, error) {
	u := fmt.Sprintf("%s/condition-sets/%s/items", s.client.baseURL, conditionSetID)

	req := &ReplaceConditionSetItemsRequest{Items: items}
	var result []ConditionSetItem
	if err := s.client.doRequest(ctx, "PATCH", u, req, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteItem deletes an item from a condition set.
func (s *ConditionSetsService) DeleteItem(ctx context.Context, conditionSetID, itemID string) error {
	u := fmt.Sprintf("%s/condition-sets/%s/items/%s", s.client.baseURL, conditionSetID, itemID)
	return s.client.doRequest(ctx, "DELETE", u, nil, nil)
}
