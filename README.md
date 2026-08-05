# Egnyte Go SDK

The official Go client libraries for the [Egnyte Public API](https://developers.egnyte.com).

## Version 2

Version 2 is the actively maintained SDK and is strongly recommended for all
new and existing integrations. It covers the Content & File Services API
surface with around 150 endpoints across 30 services.

**v2 is dependency-free:** it uses only the Go standard library and has no
third-party runtime dependencies. The dependencies listed in the repository's
root `go.mod` belong only to the unsupported legacy SDK; v2 is an independent
module defined by `v2/go.mod`.

```bash
go get github.com/egnyte/egnyte-go-sdk/v2
```

```go
package main

import (
	"context"
	"log"

	"github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	client, err := egnyte.NewClient("acme", egnyte.WithToken("YOUR_ACCESS_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	folder, _, err := client.FileSystem.Get(context.Background(), "/Shared", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("found %d files", len(folder.Files))
}
```

The import path ends in `/v2`, but the package name remains `egnyte`, so SDK
symbols are used as `egnyte.NewClient`, `egnyte.WithToken`, and so on.

See the [v2 documentation](v2/README.md), [migration guide](MIGRATING_TO_V2.md),
and [v2 package documentation](https://pkg.go.dev/github.com/egnyte/egnyte-go-sdk/v2).

## Legacy SDK

The original SDK remains available at
`github.com/egnyte/egnyte-go-sdk/egnyte` so existing applications continue to
build. It is no longer supported or actively maintained. Users should migrate
to v2, especially for new development, bug fixes, and access to the broader API
surface.

See the [migration guide](MIGRATING_TO_V2.md) for the principal API changes.

## Support and contributions

Please open a GitHub issue for bug reports and feature requests. Contributions
to v2 should follow the [v2 contribution guide](v2/CONTRIBUTING.md).

## License

[MIT](LICENSE.md)
