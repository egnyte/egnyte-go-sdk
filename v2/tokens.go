package egnyte

import (
	"context"
	"net/http"
	"net/url"
)

// TokensService handles inspection and revocation of OAuth tokens.
//
// API documentation:
//   - user info: https://developers.egnyte.com/integration/cfs/api-docs/authentication#get-user-info-for-an-oauth-token
//   - revocation: https://developers.egnyte.com/integration/cfs/api-docs/authentication#revoke-an-oauth-token
//
// These operations do not require a dedicated resource scope.
type TokensService service

// UserInfo describes the user an OAuth token belongs to.
type UserInfo struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	// UserType is admin, power or standard.
	UserType string `json:"user_type"`
	Email    string `json:"email"`
}

// UserInfo returns basic information about the user associated with the
// client's access token.
//
// GET /pubapi/v1/userinfo
func (s *TokensService) UserInfo(ctx context.Context) (*UserInfo, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/userinfo", nil)
	if err != nil {
		return nil, nil, err
	}
	info := new(UserInfo)
	resp, err := s.client.Do(req, info)
	if err != nil {
		return nil, resp, err
	}
	return info, resp, nil
}

// Revoke invalidates the given access or refresh token. Revoking an
// access token also revokes its associated refresh token. clientSecret is
// the application's API secret.
//
// POST /pubapi/v1/tokens/revoke
func (s *TokensService) Revoke(ctx context.Context, token, clientSecret string) (*Response, error) {
	form := url.Values{}
	form.Set("token", token)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	req, err := s.client.NewFormRequest(ctx, http.MethodPost, "v1/tokens/revoke", form)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
