# Migrating to version 2

Version 2 is a new Go module and a ground-up SDK design. The legacy package is
still present in this repository so migration can happen deliberately, one
application at a time.

## Change the dependency and import

```bash
go get github.com/egnyte/egnyte-go-sdk/v2
```

Replace:

```go
import "github.com/egnyte/egnyte-go-sdk/egnyte"
```

with:

```go
import "github.com/egnyte/egnyte-go-sdk/v2"
```

The package name is still `egnyte`; only its import path changes.

## Construct a client

Legacy:

```go
client, err := egnyte.NewClient(ctx, rootURL, token, httpClient)
```

Version 2:

```go
client, err := egnyte.NewClient("acme",
	egnyte.WithToken(token),
	egnyte.WithHTTPClient(httpClient),
)
```

The v2 client accepts a bare subdomain, an Egnyte host, or a full URL. Client
configuration uses functional options.

## Use services

Version 2 groups operations by API service and consistently accepts a
`context.Context` as the first method argument:

```go
folder, response, err := client.FileSystem.Get(ctx, "/Shared", nil)
body, response, err := client.FileSystemContent.Download(ctx, "/Shared/report.pdf", nil)
users, response, err := client.Users.List(ctx, nil)
```

Methods commonly return the decoded result, a response wrapper containing rate
limit information, and an error. HTTP failures can be inspected as
`*egnyte.APIError`.

## Migrate incrementally

Both versions may be imported by the same application during a staged
migration. Aliases make the distinction explicit:

```go
import (
	legacy "github.com/egnyte/egnyte-go-sdk/egnyte"
	egnytev2 "github.com/egnyte/egnyte-go-sdk/v2"
)
```

The two clients and their types are independent; values are not generally
interchangeable without conversion.

For authentication, error handling, service coverage, and complete examples,
see the [v2 README](v2/README.md).
