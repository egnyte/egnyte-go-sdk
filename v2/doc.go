// Package egnyte provides a Go client for the Egnyte Public API,
// covering the Content & File Services (CFS) API surface: file
// system operations, uploads and downloads, links, permissions, users,
// groups, search, events, webhooks, audit reporting, metadata, trash,
// bookmarks, comments, workflows, project folders, e-signatures, AI, and
// more.
//
// # Getting started
//
// Construct a client for your domain and access each API family through
// the client's service fields:
//
//	client, err := egnyte.NewClient("acme", egnyte.WithToken(token))
//	if err != nil {
//		log.Fatal(err)
//	}
//	info, _, err := client.Tokens.UserInfo(ctx)
//
// The domain may be a bare subdomain ("acme"), a host
// ("acme.egnyte.com"), or a full URL. Egnyte Gov Cloud domains are
// supported by passing the full host ("acme.egnytegov.com").
//
// # Authentication
//
// Every API call sends the OAuth bearer token configured with WithToken
// (or SetToken). Tokens can also be obtained through the client itself:
//
//	client, _ := egnyte.NewClient("acme")
//	token, _, err := client.RequestToken(ctx, egnyte.PasswordCredentials{
//		ClientID:     apiKey,
//		ClientSecret: apiSecret,
//		Username:     username,
//		Password:     password,
//	})
//
// RequestToken supports the Resource Owner Password flow (internal
// apps), the Authorization Code exchange, and the Refresh Token flow.
// AuthorizationURL builds the redirect URL for browser-based flows.
//
// # Errors and rate limits
//
// Responses with status >= 400 return an *APIError carrying the status
// code, the API's error payload, and any Retry-After duration. Rate
// limit state from the X-Accesstoken-* headers is available on every
// *Response via its Rate field.
//
// # Method conventions
//
// Methods take a context.Context first and return the decoded result,
// the *Response wrapper, and an error. Optional parameters are grouped
// into *Options structs whose zero values are omitted from requests.
// File system paths are percent-encoded automatically (see EncodePath).
package egnyte
