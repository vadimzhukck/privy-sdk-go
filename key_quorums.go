package privy

import (
	"context"
	"fmt"
)

// KeyQuorumsService handles key quorum operations.
type KeyQuorumsService struct {
	client *Client
}

// CreateKeyQuorumRequest represents a request to create a key quorum.
type CreateKeyQuorumRequest struct {
	PublicKey string `json:"public_key"`
}

// UpdateKeyQuorumRequest represents a request to update a key quorum.
type UpdateKeyQuorumRequest struct {
	PublicKey string `json:"public_key,omitempty"`
}

// Create creates a new key quorum.
func (s *KeyQuorumsService) Create(ctx context.Context, req *CreateKeyQuorumRequest) (*KeyQuorum, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/key-quorums", s.client.baseURL)

	var kq KeyQuorum
	if err := s.client.doRequest(ctx, "POST", u, req, &kq); err != nil {
		return nil, err
	}

	return &kq, nil
}

// Get retrieves a key quorum by its ID.
func (s *KeyQuorumsService) Get(ctx context.Context, keyQuorumID string) (*KeyQuorum, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/key-quorums/%s", s.client.baseURL, keyQuorumID)

	var kq KeyQuorum
	if err := s.client.doRequest(ctx, "GET", u, nil, &kq); err != nil {
		return nil, err
	}

	return &kq, nil
}

// Update updates a key quorum.
func (s *KeyQuorumsService) Update(ctx context.Context, keyQuorumID string, req *UpdateKeyQuorumRequest) (*KeyQuorum, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/key-quorums/%s", s.client.baseURL, keyQuorumID)

	var kq KeyQuorum
	if err := s.client.doRequest(ctx, "PATCH", u, req, &kq); err != nil {
		return nil, err
	}

	return &kq, nil
}

// Delete deletes a key quorum.
func (s *KeyQuorumsService) Delete(ctx context.Context, keyQuorumID string) error {
	if s == nil || s.client == nil {
		return ErrNilClient
	}
	u := fmt.Sprintf("%s/key-quorums/%s", s.client.baseURL, keyQuorumID)
	return s.client.doRequest(ctx, "DELETE", u, nil, nil)
}
