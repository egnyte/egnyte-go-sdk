# Contributing to the v2 SDK

## Prerequisites

The v2 module requires Go 1.24 or later and has no external dependencies.

Run commands for v2 from the `v2/` directory:

```bash
gofmt -l .
go vet ./...
go test -race ./...
go build ./examples/...
```

The formatting command must print no file names. Unit tests do not require
credentials. The real-domain integration suite is optional and documented in
[docs/integration-testing.md](docs/integration-testing.md).

## API conventions

The published Egnyte API documentation and OpenAPI definitions are the API
contract. When adding or changing endpoints:

- Keep one `<api>.go` service file and `<api>_test.go` file per API family.
- Register services on `Client` in `egnyte.go` in alphabetical order.
- Accept `context.Context` first and return `(result, *Response, error)`, or
  `(*Response, error)` when an endpoint has no response body.
- Keep version-specific endpoint variants behind a shared unexported helper.
- Mirror API JSON names exactly, including inconsistencies such as
  `lastModified` versus `last_modified`.
- Use `omitempty` for optional request fields and pointers such as
  `egnyte.Bool` or `egnyte.Int` where false or zero must be sent explicitly.
- Put optional query parameters in an options struct with an unexported
  `values() url.Values` builder.
- Validate required input before sending a request.
- Encode file-system paths with `EncodePath`; do not use `url.PathEscape`,
  which leaves `$` unescaped.
- Return errors through the shared `*APIError` mapping. Library code must not
  panic or log.
- Test request method, path, query or body encoding, response decoding, input
  validation, and service-level error mapping.

## Pull requests

Keep changes focused, include tests for behavior changes, and update public
documentation when the API changes. Pull requests must keep both the legacy
and v2 CI jobs green, even when a change affects only v2.
