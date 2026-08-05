# Errors and rate limits

## APIError

Every response with status >= 400 is returned as an `*egnyte.APIError`.
The Egnyte API uses several error envelopes depending on the endpoint —
`{"Errors": [{description, code}]}`, `{"message": ...}`,
`{"errorMessage": ...}`, `{"formErrors": [{code, msg}]}`,
`{"errors": [{msg, code}]}` and `{"inputErrors": {field: [{msg, code}]}}`
— and APIError captures all of them, plus the raw body:

```go
_, _, err := client.Links.Get(ctx, "nope")
var apiErr *egnyte.APIError
if errors.As(err, &apiErr) {
	apiErr.StatusCode // 404
	apiErr.Message    // flat message, if the API sent one
	apiErr.Errors     // []ErrorDetail{Description, Code}, if sent
	apiErr.Body       // raw response body
}
```

Client-side validation failures (missing required fields, path segments
that would leave the endpoint's route family) are returned as plain
errors *before* any request is sent.

## Response size cap

Successful JSON responses are decoded up to 64 MiB by default; a larger
body fails with `egnyte.ErrResponseTooLarge` rather than being buffered.
Streaming reads — `DoRaw` and `io.Writer` destinations, i.e. file
downloads — are never capped.

```go
client, _ := egnyte.NewClient("acme",
	egnyte.WithToken(token),
	egnyte.WithMaxResponseBytes(256<<20)) // raise it, or <= 0 to disable
```

## Rate limits

Limits are enforced per access token: by default 2 calls/second and
1,000 calls/day (AI endpoints are lower). Two signals to handle:

**Response headers** — parsed into `Response.Rate` on every call:

```go
_, resp, _ := client.Tokens.UserInfo(ctx)
resp.Rate.QPSCurrent    // calls this second
resp.Rate.QPSAllotted   // per-second allowance
resp.Rate.QuotaCurrent  // calls today (resets 00:00 UTC)
resp.Rate.QuotaAllotted // daily allowance
```

**Throttled responses** — `429` (and `409` on the OAuth endpoint) with a
`Retry-After` header:

```go
if apiErr.IsRateLimit() {
	time.Sleep(apiErr.RetryAfter) // zero if the server sent no hint
	// retry
}
```

The SDK does not retry automatically; wrap calls in your own retry
policy if needed.

## Redirect-based endpoints

Two endpoints use redirects as data, not as hops, and the SDK handles
them without following the redirect: `Audit.JobStatus` (303 = report
complete; `AuditJobStatus.ReportURL` carries the Location) and
`Navigate.NavigateV1` (303 Location is the one-time URL).
