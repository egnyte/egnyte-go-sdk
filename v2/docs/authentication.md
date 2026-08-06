# Authentication

The SDK authenticates every request with an OAuth 2.0 bearer token. You
need an **API key** (client id) from [developers.egnyte.com](https://developers.egnyte.com)
— keys issued after January 2015 also have a **client secret**.

## Bringing your own token

```go
client, err := egnyte.NewClient("acme", egnyte.WithToken(token))
// or later:
client.SetToken(token)
```

Access tokens are valid for 30 days and are revoked when the user
changes their password. Always cache tokens — the OAuth token endpoint
has its own rate limit (10 requests/user/hour for internal apps).

## Obtaining tokens through the SDK

`Client.RequestToken` posts to `/puboauth/token` and stores the returned
access token on the client. See the runnable
[internal-application authentication example](../examples/authentication/)
for environment validation, safe output, and cached-token restoration. Three
grants are supported:

```go
// Internal applications — Resource Owner Password flow:
token, _, err := client.RequestToken(ctx, egnyte.PasswordCredentials{
	ClientID: key, ClientSecret: secret,
	Username: user, Password: pass,
	Scopes: []string{"Egnyte.filesystem", "Egnyte.link"}, // optional
})

// Public applications — Authorization Code flow:
// 1) send the user to:
url := client.AuthorizationURL(key, redirectURI, "code", state, scopes)
// 2) exchange the code from the callback:
token, _, err = client.RequestToken(ctx, egnyte.AuthorizationCode{
	ClientID: key, ClientSecret: secret,
	RedirectURI: redirectURI, Code: code,
})

// Renewing — Refresh Token flow:
token, _, err = client.RequestToken(ctx, egnyte.RefreshGrant{
	ClientID: key, ClientSecret: secret, RefreshToken: token.RefreshToken,
})
```

Tokens can be revoked with `client.Tokens.Revoke(ctx, token, clientSecret)`;
revoking an access token also revokes its refresh token.

## Scopes

Scope tokens to only what the application needs (required for
production approval of third-party apps). See the
[developer portal OAuth scope matrix](https://developers.egnyte.com/integration/cfs/api-docs/authentication#oauth-scopes)
for the APIs covered by each scope: `Egnyte.filesystem`,
`Egnyte.link`, `Egnyte.permission`, `Egnyte.user`, `Egnyte.group`,
`Egnyte.audit`, `Egnyte.webhooks`, `Egnyte.ai`, `Egnyte.sign`,
`Egnyte.documentportal`, `Egnyte.uploadrequests`, `Egnyte.etmf`,
`Egnyte.controlleddocs`, `Egnyte.projectfolders`, `Egnyte.bookmark`,
`Egnyte.salesforce`, `Egnyte.integrations`, `Egnyte.launchwebsession`.

## Impersonation

Administrator tokens can act as another user on every request:

```go
client, _ := egnyte.NewClient("acme",
	egnyte.WithToken(adminToken),
	egnyte.WithActAs("jsmith")) // or WithActAsEmail("jsmith@example.com")
```

Impersonated calls run in the impersonated user's permission context and
are audited as such.

## Gov Cloud

Pass the full host — everything else (API paths, OAuth endpoints) is
derived from it:

```go
client, err := egnyte.NewClient("acme.egnytegov.com", egnyte.WithToken(token))
```
