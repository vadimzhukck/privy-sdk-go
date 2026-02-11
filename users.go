package privy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// UsersService handles user-related operations.
type UsersService struct {
	client *Client
}

// CreateUserRequest represents a request to create a new user.
// Note: Only Ethereum embedded wallets can be created during user creation.
// For Solana or other chain wallets, use client.Wallets().Create() after creating the user.
type CreateUserRequest struct {
	LinkedAccounts       []LinkedAccountInput `json:"linked_accounts"`
	CreateEthereumWallet bool                 `json:"create_ethereum_wallet,omitempty"`
	CustomMetadata       map[string]any       `json:"custom_metadata,omitempty"`
}

// LinkedAccountInput represents input for linking an account.
// Use the helper functions below to create inputs for specific account types.
type LinkedAccountInput struct {
	Type           LinkedAccountType `json:"type"`
	Address        string            `json:"address,omitempty"`          // For email, wallet
	PhoneNumber    string            `json:"phone_number,omitempty"`     // For phone
	Subject        string            `json:"subject,omitempty"`          // For OAuth (Google, Twitter, etc.) - the provider's user ID
	Name           string            `json:"name,omitempty"`             // For OAuth - user's display name
	Email          string            `json:"email,omitempty"`            // For OAuth - user's email from provider
	Username       string            `json:"username,omitempty"`         // For Twitter, Discord, GitHub, Telegram
	CustomUserID   string            `json:"custom_user_id,omitempty"`   // For custom auth
	FID            int64             `json:"fid,omitempty"`              // For Farcaster
	DisplayName    string            `json:"display_name,omitempty"`     // For Farcaster
	Bio            string            `json:"bio,omitempty"`              // For Farcaster
	PfpURL         string            `json:"pfp_url,omitempty"`          // For Farcaster, Telegram
	TelegramUserID string            `json:"telegram_user_id,omitempty"` // For Telegram
	FirstName      string            `json:"first_name,omitempty"`       // For Telegram
	LastName       string            `json:"last_name,omitempty"`        // For Telegram
}

// ===========================================
// Helper functions for creating linked accounts
// ===========================================

// LinkedAccountInputEmail creates a linked account input for an email address.
//
// Example:
//
//	privy.LinkedAccountInputEmail("user@example.com")
func LinkedAccountInputEmail(email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:    LinkedAccountTypeEmail,
		Address: email,
	}
}

// LinkedAccountInputPhone creates a linked account input for a phone number.
// The phone number should be in E.164 format (e.g., "+14155551234").
//
// Example:
//
//	privy.LinkedAccountInputPhone("+14155551234")
func LinkedAccountInputPhone(phoneNumber string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:        LinkedAccountTypePhone,
		PhoneNumber: phoneNumber,
	}
}

// LinkedAccountInputWallet creates a linked account input for a wallet address.
//
// Example:
//
//	privy.LinkedAccountInputWallet("0x1234567890abcdef...")
func LinkedAccountInputWallet(address string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:    LinkedAccountTypeWallet,
		Address: address,
	}
}

// LinkedAccountInputGoogle creates a linked account input for a Google OAuth account.
// The subject is the Google user ID (from the "sub" claim in the ID token).
//
// Example:
//
//	privy.LinkedAccountInputGoogle("123456789", "John Doe", "john@gmail.com")
func LinkedAccountInputGoogle(subject, name, email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:    LinkedAccountTypeGoogle,
		Subject: subject,
		Name:    name,
		Email:   email,
	}
}

// LinkedAccountInputTwitter creates a linked account input for a Twitter/X OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputTwitter("twitter_user_id", "johndoe", "John Doe")
func LinkedAccountInputTwitter(subject, username, name string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:     LinkedAccountTypeTwitter,
		Subject:  subject,
		Username: username,
		Name:     name,
	}
}

// LinkedAccountInputDiscord creates a linked account input for a Discord OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputDiscord("discord_user_id", "johndoe#1234", "John Doe", "john@example.com")
func LinkedAccountInputDiscord(subject, username, name, email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:     LinkedAccountTypeDiscord,
		Subject:  subject,
		Username: username,
		Name:     name,
		Email:    email,
	}
}

// LinkedAccountInputGithub creates a linked account input for a GitHub OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputGithub("github_user_id", "johndoe", "John Doe", "john@example.com")
func LinkedAccountInputGithub(subject, username, name, email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:     LinkedAccountTypeGithub,
		Subject:  subject,
		Username: username,
		Name:     name,
		Email:    email,
	}
}

// LinkedAccountInputApple creates a linked account input for an Apple OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputApple("apple_user_id", "john@privaterelay.appleid.com")
func LinkedAccountInputApple(subject, email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:    LinkedAccountTypeApple,
		Subject: subject,
		Email:   email,
	}
}

// LinkedAccountInputLinkedIn creates a linked account input for a LinkedIn OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputLinkedIn("linkedin_user_id", "John Doe", "john@example.com")
func LinkedAccountInputLinkedIn(subject, name, email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:    LinkedAccountTypeLinkedin,
		Subject: subject,
		Name:    name,
		Email:   email,
	}
}

// LinkedAccountInputSpotify creates a linked account input for a Spotify OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputSpotify("spotify_user_id", "John Doe", "john@example.com")
func LinkedAccountInputSpotify(subject, name, email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:    LinkedAccountTypeSpotify,
		Subject: subject,
		Name:    name,
		Email:   email,
	}
}

// LinkedAccountInputInstagram creates a linked account input for an Instagram OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputInstagram("instagram_user_id", "johndoe")
func LinkedAccountInputInstagram(subject, username string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:     LinkedAccountTypeInstagram,
		Subject:  subject,
		Username: username,
	}
}

// LinkedAccountInputTiktok creates a linked account input for a TikTok OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputTiktok("tiktok_user_id", "johndoe", "John Doe")
func LinkedAccountInputTiktok(subject, username, name string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:     LinkedAccountTypeTiktok,
		Subject:  subject,
		Username: username,
		Name:     name,
	}
}

// LinkedAccountInputTwitch creates a linked account input for a Twitch OAuth account.
//
// Example:
//
//	privy.LinkedAccountInputTwitch("twitch_user_id", "johndoe", "John Doe", "john@example.com")
func LinkedAccountInputTwitch(subject, username, name, email string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:     LinkedAccountTypeTwitch,
		Subject:  subject,
		Username: username,
		Name:     name,
		Email:    email,
	}
}

// LinkedAccountInputFarcaster creates a linked account input for a Farcaster account.
//
// Example:
//
//	privy.LinkedAccountInputFarcaster(12345, "johndoe", "John Doe", "Web3 enthusiast", "https://example.com/pfp.jpg")
func LinkedAccountInputFarcaster(fid int64, username, displayName, bio, pfpURL string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:        LinkedAccountTypeFarcaster,
		FID:         fid,
		Username:    username,
		DisplayName: displayName,
		Bio:         bio,
		PfpURL:      pfpURL,
	}
}

// LinkedAccountInputTelegram creates a linked account input for a Telegram account.
//
// Example:
//
//	privy.LinkedAccountInputTelegram("123456789", "johndoe", "John", "Doe", "https://example.com/photo.jpg")
func LinkedAccountInputTelegram(telegramUserID, username, firstName, lastName, photoURL string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:           LinkedAccountTypeTelegram,
		TelegramUserID: telegramUserID,
		Username:       username,
		FirstName:      firstName,
		LastName:       lastName,
		PfpURL:         photoURL,
	}
}

// LinkedAccountInputCustomAuth creates a linked account input for a custom authentication provider.
//
// Example:
//
//	privy.LinkedAccountInputCustomAuth("custom_user_id_123")
func LinkedAccountInputCustomAuth(customUserID string) LinkedAccountInput {
	return LinkedAccountInput{
		Type:         LinkedAccountTypeCustomAuth,
		CustomUserID: customUserID,
	}
}

// UpdateUserMetadataRequest represents a request to update user metadata.
type UpdateUserMetadataRequest struct {
	CustomMetadata map[string]any `json:"custom_metadata"`
}

// Create creates a new user with the specified linked accounts.
func (s *UsersService) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	url := fmt.Sprintf("%s/users", s.client.baseURL)

	var user User
	if err := s.client.doRequest(ctx, "POST", url, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// Get retrieves a user by their Privy ID.
func (s *UsersService) Get(ctx context.Context, userID string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	url := fmt.Sprintf("%s/users/%s", s.client.authURL, userID)

	var user User
	if err := s.client.doRequest(ctx, "GET", url, nil, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByIDToken retrieves a user using their identity token.
func (s *UsersService) GetByIDToken(ctx context.Context, idToken string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	url := fmt.Sprintf("%s/users/me", s.client.authURL)

	var user User
	// This needs special handling with the ID token as bearer
	if err := s.client.doRequest(ctx, "GET", url, nil, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// Delete deletes a user by their Privy ID.
func (s *UsersService) Delete(ctx context.Context, userID string) error {
	if s == nil || s.client == nil {
		return ErrNilClient
	}
	url := fmt.Sprintf("%s/users/%s", s.client.authURL, userID)
	return s.client.doRequest(ctx, "DELETE", url, nil, nil)
}

// List lists all users with pagination.
func (s *UsersService) List(ctx context.Context, opts *ListOptions) (*PaginatedResponse[User], error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users", s.client.authURL)

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

	var resp PaginatedResponse[User]
	if err := s.client.doRequest(ctx, "GET", u, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetByEmail retrieves a user by their email address.
func (s *UsersService) GetByEmail(ctx context.Context, email string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/email/address", s.client.authURL)

	req := map[string]string{"address": email}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByPhone retrieves a user by their phone number.
func (s *UsersService) GetByPhone(ctx context.Context, phone string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/phone/number", s.client.authURL)

	req := map[string]string{"number": phone}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByWalletAddress retrieves a user by their wallet address.
func (s *UsersService) GetByWalletAddress(ctx context.Context, address string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/wallet/address", s.client.authURL)

	req := map[string]string{"address": address}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetBySmartWalletAddress retrieves a user by their smart wallet address.
func (s *UsersService) GetBySmartWalletAddress(ctx context.Context, address string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/smart_wallet/address", s.client.authURL)

	req := map[string]string{"address": address}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByCustomAuthID retrieves a user by their custom auth ID.
func (s *UsersService) GetByCustomAuthID(ctx context.Context, customAuthID string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/custom_auth/id", s.client.authURL)

	req := map[string]string{"id": customAuthID}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByFarcasterFID retrieves a user by their Farcaster FID.
func (s *UsersService) GetByFarcasterFID(ctx context.Context, fid int64) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/farcaster/fid", s.client.authURL)

	req := map[string]int64{"fid": fid}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByTwitterSubject retrieves a user by their Twitter subject.
func (s *UsersService) GetByTwitterSubject(ctx context.Context, subject string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/twitter/subject", s.client.authURL)

	req := map[string]string{"twitter_subject": subject}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByTwitterUsername retrieves a user by their Twitter username.
func (s *UsersService) GetByTwitterUsername(ctx context.Context, username string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/twitter/username", s.client.authURL)

	req := map[string]string{"twitter_username": username}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByDiscordUsername retrieves a user by their Discord username.
func (s *UsersService) GetByDiscordUsername(ctx context.Context, username string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/discord/username", s.client.authURL)

	req := map[string]string{"discord_username": username}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByTelegramUserID retrieves a user by their Telegram user ID.
func (s *UsersService) GetByTelegramUserID(ctx context.Context, telegramUserID string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/telegram/telegram_user_id", s.client.authURL)

	req := map[string]string{"telegram_user_id": telegramUserID}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByTelegramUsername retrieves a user by their Telegram username.
func (s *UsersService) GetByTelegramUsername(ctx context.Context, username string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/telegram/username", s.client.authURL)

	req := map[string]string{"telegram_username": username}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByGithubUsername retrieves a user by their GitHub username.
func (s *UsersService) GetByGithubUsername(ctx context.Context, username string) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/github/username", s.client.authURL)

	req := map[string]string{"github_username": username}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateMetadata updates the custom metadata for a user.
func (s *UsersService) UpdateMetadata(ctx context.Context, userID string, metadata map[string]any) (*User, error) {
	if s == nil || s.client == nil {
		return nil, ErrNilClient
	}
	u := fmt.Sprintf("%s/users/%s/custom_metadata", s.client.authURL, userID)

	req := &UpdateUserMetadataRequest{CustomMetadata: metadata}
	var user User
	if err := s.client.doRequest(ctx, "POST", u, req, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
