package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// oauthTokenPath is the OAuth token endpoint. Unlike API endpoints it
// lives outside /pubapi and has its own rate limits (throttled requests
// get 409 with a Retry-After header).
const oauthTokenPath = "/puboauth/token"

// Token is an OAuth token pair returned by the token endpoint.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	// ExpiresIn is the access token lifetime in seconds (30 days).
	ExpiresIn int `json:"expires_in"`
}

// PasswordCredentials requests a token using the Resource Owner Password
// Credentials flow. Only for internal (customer-built) applications.
type PasswordCredentials struct {
	ClientID     string
	ClientSecret string // required for API keys issued after January 2015
	Username     string
	Password     string
	Scopes       []string
}

// AuthorizationCode exchanges a code obtained from the authorization
// redirect for a token (Authorization Code flow, step 3).
type AuthorizationCode struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string // must match the redirect URI used in step 1
	Code         string
	Scopes       []string // must match step 1 if provided
}

// RefreshGrant requests a new token pair using a refresh token.
type RefreshGrant struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

func (g PasswordCredentials) values() url.Values {
	v := url.Values{}
	v.Set("grant_type", "password")
	v.Set("client_id", g.ClientID)
	if g.ClientSecret != "" {
		v.Set("client_secret", g.ClientSecret)
	}
	v.Set("username", g.Username)
	v.Set("password", g.Password)
	if len(g.Scopes) > 0 {
		v.Set("scope", strings.Join(g.Scopes, " "))
	}
	return v
}

func (g AuthorizationCode) values() url.Values {
	v := url.Values{}
	v.Set("grant_type", "authorization_code")
	v.Set("client_id", g.ClientID)
	if g.ClientSecret != "" {
		v.Set("client_secret", g.ClientSecret)
	}
	v.Set("redirect_uri", g.RedirectURI)
	v.Set("code", g.Code)
	if len(g.Scopes) > 0 {
		v.Set("scope", strings.Join(g.Scopes, " "))
	}
	return v
}

func (g RefreshGrant) values() url.Values {
	v := url.Values{}
	v.Set("grant_type", "refresh_token")
	v.Set("client_id", g.ClientID)
	v.Set("client_secret", g.ClientSecret)
	v.Set("refresh_token", g.RefreshToken)
	return v
}

// Grant is an OAuth grant that can be exchanged for a Token.
type Grant interface {
	values() url.Values
}

// RequestToken exchanges the given grant for a token pair and stores the
// access token on the client for subsequent API calls.
func (c *Client) RequestToken(ctx context.Context, grant Grant) (*Token, *Response, error) {
	if grant == nil {
		return nil, nil, fmt.Errorf("egnyte: grant must not be nil")
	}
	req, err := c.newRequest(ctx, http.MethodPost, oauthTokenPath, strings.NewReader(grant.values().Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// The token endpoint is unauthenticated; a stale Authorization
	// header must not leak into it.
	req.Header.Del("Authorization")

	token := new(Token)
	resp, err := c.Do(req, token)
	if err != nil {
		return nil, resp, err
	}
	c.SetToken(token.AccessToken)
	return token, resp, nil
}

// AuthorizationURL builds the URL to redirect a user to for the
// Authorization Code flow (responseType "code") or Implicit Grant flow
// (responseType "token").
func (c *Client) AuthorizationURL(clientID, redirectURI, responseType, state string, scopes []string) string {
	u := *c.baseURL
	u.Path = oauthTokenPath
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", responseType)
	if state != "" {
		q.Set("state", state)
	}
	if len(scopes) > 0 {
		q.Set("scope", strings.Join(scopes, " "))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// SetToken replaces the OAuth access token used by the client, e.g. after
// restoring a cached token. It is safe to call concurrently with
// in-flight requests.
func (c *Client) SetToken(token string) {
	c.tokenMu.Lock()
	c.token = token
	c.tokenMu.Unlock()
}

func (c *Client) tokenValue() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.token
}

// String implements fmt.Stringer with credentials redacted, so that
// logging a Token (including with %+v) never leaks secrets. JSON
// marshalling is unaffected and still carries the real values.
func (t Token) String() string {
	return fmt.Sprintf("egnyte.Token{AccessToken:REDACTED, RefreshToken:REDACTED, TokenType:%s, ExpiresIn:%d}",
		t.TokenType, t.ExpiresIn)
}

// GoString implements fmt.GoStringer (%#v) with credentials redacted.
func (t Token) GoString() string { return t.String() }

// String implements fmt.Stringer with credentials redacted.
func (g PasswordCredentials) String() string {
	return fmt.Sprintf("egnyte.PasswordCredentials{ClientID:%s, ClientSecret:REDACTED, Username:%s, Password:REDACTED, Scopes:%v}",
		g.ClientID, g.Username, g.Scopes)
}

// GoString implements fmt.GoStringer (%#v) with credentials redacted.
func (g PasswordCredentials) GoString() string { return g.String() }

// String implements fmt.Stringer with credentials redacted.
func (g AuthorizationCode) String() string {
	return fmt.Sprintf("egnyte.AuthorizationCode{ClientID:%s, ClientSecret:REDACTED, RedirectURI:%s, Code:REDACTED, Scopes:%v}",
		g.ClientID, g.RedirectURI, g.Scopes)
}

// GoString implements fmt.GoStringer (%#v) with credentials redacted.
func (g AuthorizationCode) GoString() string { return g.String() }

// String implements fmt.Stringer with credentials redacted.
func (g RefreshGrant) String() string {
	return fmt.Sprintf("egnyte.RefreshGrant{ClientID:%s, ClientSecret:REDACTED, RefreshToken:REDACTED}", g.ClientID)
}

// GoString implements fmt.GoStringer (%#v) with credentials redacted.
func (g RefreshGrant) GoString() string { return g.String() }
