package egnyte

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// OAuth Scopes
const (
	FilesystemScope       = "Egnyte.filesystem"
	UserScope             = "Egnyte.user"
	GroupScope            = "Egnyte.group"
	PermissionScope       = "Egnyte.permission"
	LaunchWebSessionScope = "Egnyte.launchwebsession"
)

// OAuthEndpoint constructs an `oauth2.Endpoint` for the given domain
// This returns nil if invalid domain is provided
func OAuthEndpoint(domain string) oauth2.Endpoint {
	if domain == "" {
		return oauth2.Endpoint{}
	}
	authUrl := fmt.Sprintf("https://%s%s", domain, URI_OAUTH)
	tokenUrl := fmt.Sprintf("https://%s%s", domain, URI_OAUTH)
	return oauth2.Endpoint{AuthURL: authUrl, TokenURL: tokenUrl, AuthStyle: oauth2.AuthStyleInParams}
}

// GetAccessToken return auth token with grant type password
// This returns err if invalid details is provided else return auth token
func GetAccessToken(ctx context.Context, config map[string]string) (*oauth2.Token, error) {
	endpoint := OAuthEndpoint(config["domain"])
	oauthConfig := oauth2.Config{ClientID: config["api_key"], Endpoint: endpoint}
	return oauthConfig.PasswordCredentialsToken(ctx, config["username"], config["password"])
}
