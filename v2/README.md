# Egnyte Go SDK v2

A Go client library for the [Egnyte Public API](https://developers.egnyte.com/integration/cfs/api-docs/overview), covering the Content & File Services (CFS) API surface — around 150 endpoints across 30 services.

**v2 is dependency-free:** it uses only the Go standard library and has no
third-party runtime dependencies. Its `go.mod` declares the v2 module path and
minimum Go version; it does not inherit the legacy SDK's dependencies from the
repository-root module.

```bash
go get github.com/egnyte/egnyte-go-sdk/v2
```

Requires Go 1.24+.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	ctx := context.Background()

	client, err := egnyte.NewClient("acme", egnyte.WithToken("YOUR_ACCESS_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	folder, _, err := client.FileSystem.Get(ctx, "/Shared", &egnyte.FileSystemGetOptions{
		ListContent: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range folder.Folders {
		fmt.Println("folder:", f.Path)
	}
	for _, f := range folder.Files {
		fmt.Printf("file:   %s (%d bytes)\n", f.Path, f.Size)
	}
}
```

The domain accepts a bare subdomain (`"acme"`), a host (`"acme.egnyte.com"`), or a full URL. **Egnyte Gov Cloud** works by passing the full host: `egnyte.NewClient("acme.egnytegov.com", ...)`.

The default HTTP client has no timeout; long-running programs should bring their own:

```go
client, err := egnyte.NewClient("acme",
	egnyte.WithToken(token),
	egnyte.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}))
```

(Skip the timeout — or use a generous one — for large streaming uploads and downloads, since it covers the whole transfer. Per-call deadlines via `context.WithTimeout` work too.)

## Authentication

All requests use an OAuth 2.0 bearer token. Bring your own token, or obtain one
through the client. A complete runnable internal-application flow is available
in [examples/authentication/](examples/authentication/):

```go
// Resource Owner Password flow (internal applications):
client, _ := egnyte.NewClient("acme")
token, _, err := client.RequestToken(ctx, egnyte.PasswordCredentials{
	ClientID:     apiKey,    // from developers.egnyte.com
	ClientSecret: apiSecret,
	Username:     username,
	Password:     password,
})
// token.AccessToken is now set on the client automatically.

// Authorization Code flow (public applications):
url := client.AuthorizationURL(apiKey, redirectURI, "code", state,
	[]string{"Egnyte.filesystem", "Egnyte.link"})
// ...redirect the user, receive ?code=..., then:
token, _, err = client.RequestToken(ctx, egnyte.AuthorizationCode{
	ClientID: apiKey, ClientSecret: apiSecret,
	RedirectURI: redirectURI, Code: code,
})

// Refresh flow:
token, _, err = client.RequestToken(ctx, egnyte.RefreshGrant{
	ClientID: apiKey, ClientSecret: apiSecret, RefreshToken: refreshToken,
})
```

Administrators can impersonate users on every call with `egnyte.WithActAs(username)` or `egnyte.WithActAsEmail(email)`. See [docs/authentication.md](docs/authentication.md) for details and scopes.

## Common operations

```go
// Upload (streams; files are never buffered in memory):
f, _ := os.Open("report.pdf")
defer f.Close()
uploaded, _, err := client.FileSystemContent.Upload(ctx, "/Shared/Reports/report.pdf", f, nil)

// Download:
body, _, err := client.FileSystemContent.Download(ctx, "/Shared/Reports/report.pdf", nil)
defer body.Close()
io.Copy(dst, body)

// Share link:
link, _, err := client.Links.Create(ctx, egnyte.CreateLinkRequest{
	Path: "/Shared/Reports/report.pdf", Type: "file", Accessibility: "domain",
})

// Search:
results, _, err := client.Search.Search(ctx, "quarterly revenue", nil)

// Permissions:
_, err = client.Permissions.Set(ctx, "/Shared/Reports", egnyte.SetFolderPermissions{
	UserPerms: map[string]egnyte.PermissionLevel{"jsmith": egnyte.PermissionEditor},
})
```

Files larger than 100 MB should use the chunked upload protocol (`FileSystemContent.UploadChunk`).

Runnable programs live in [examples/](examples/):

| Example | Shows |
|---|---|
| [authentication](examples/authentication/) | Obtaining, using and restoring an OAuth token without printing it |
| [quickstart](examples/quickstart/) | Authentication, folder listing, rate-limit readout |
| [upload-download](examples/upload-download/) | Streaming upload, share link, download round-trip |
| [events-watch](examples/events-watch/) | Tailing the domain event stream with a cursor |
| [audit-report](examples/audit-report/) | Async report jobs: create → poll → fetch results |
| [ai-ask](examples/ai-ask/) | AI Assistant Q&A with folder scoping and citations |

## Services

| Client field | API | Client field | API |
|---|---|---|---|
| `AI` | AI (Q&A, assistant, KBs, hybrid search) | `Metadata` | Custom metadata namespaces & values |
| `Agents` | Custom AI agents | `MSP` | Reseller/MSP management |
| `Audit` | Audit reports (v1) & event stream (v2) | `Navigate` | One-time Web UI URLs |
| `Bookmarks` | Folder bookmarks | `Permissions` | Folder permissions |
| `Comments` | File/folder comments (notes) | `Procore` | Procore integration |
| `ControlledDocs` | Controlled document imports | `ProjectCustomFields` | Project metadata fields |
| `DocumentPortal` | Document Portal workspaces | `ProjectFolders` | Project folders |
| `ETMF` | electronic Trial Master File | `Salesforce` | Salesforce folder mapping |
| `Events` | Domain event polling | `Search` | Keyword & metadata search |
| `FileSystem` | Files/folders (+ folder options) | `Sign` | E-signature requests |
| `FileSystemContent` | Upload/download (incl. chunked) | `Tokens` | Token introspection/revocation |
| `Groups` | Group management (SCIM) | `Trash` | Trash listing/restore/purge |
| `Insights` | Recently accessed files | `UploadRequests` | Document collection requests |
| `Links` | Share links | `Users` | User management (SCIM) |
| `Workflows` | Review/approval workflows & tasks | `Webhooks` | Webhook registrations |

## Errors and rate limits

Every response with status 400 or higher is returned as an `*egnyte.APIError` with the status code, the API's error payload, and any `Retry-After` hint:

```go
_, _, err := client.FileSystem.Get(ctx, "/Shared/missing", nil)
var apiErr *egnyte.APIError
if errors.As(err, &apiErr) {
	if apiErr.IsRateLimit() {
		time.Sleep(apiErr.RetryAfter)
	}
	fmt.Println(apiErr.StatusCode, apiErr.Message)
}
```

Per-token rate limit state (2 calls/second, 1,000/day by default) is parsed from every response:

```go
_, resp, _ := client.Tokens.UserInfo(ctx)
fmt.Printf("used %d of %d calls today\n", resp.Rate.QuotaCurrent, resp.Rate.QuotaAllotted)
```

See [docs/errors-and-rate-limits.md](docs/errors-and-rate-limits.md).

## Testing

```bash
go test ./...                                    # unit tests, no credentials needed
cp .env.integration.example .env.integration    # one-time live-test setup
# Fill .env.integration, then load it as described in docs/integration-testing.md
go test -tags integration -run TestIntegration  # against a real domain
```

The integration suite runs five scenarios (full round-trip, read-only sweep across 13 API families, file lifecycle, live-contract checks, chunked upload) inside self-cleaning scratch folders. Configure it with `EGNYTE_DOMAIN` plus either `EGNYTE_TOKEN` or `EGNYTE_KEY`/`EGNYTE_SECRET`/`EGNYTE_USERNAME`/`EGNYTE_PASSWORD` — see [docs/integration-testing.md](docs/integration-testing.md).

## Contributing

The SDK is organized by API family, with `httptest`-based unit tests covering
request and response behavior. See [CONTRIBUTING.md](CONTRIBUTING.md) for the
development workflow and API conventions.

## License

[MIT](../LICENSE.md)
