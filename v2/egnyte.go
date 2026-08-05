package egnyte

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

const (
	// Version is the SDK version, sent in the User-Agent header.
	Version = "2.0.0"

	defaultUserAgent = "egnyte-go-sdk/" + Version

	// apiBasePath is the common prefix of all Public API endpoints.
	apiBasePath = "/pubapi"

	// defaultMaxResponseBytes bounds successful non-streaming responses.
	// Generous enough for the largest documented report page, small
	// enough that a runaway response cannot exhaust memory. Change it
	// with WithMaxResponseBytes.
	defaultMaxResponseBytes = 64 << 20 // 64 MiB
)

// Client manages communication with the Egnyte Public API.
type Client struct {
	httpClient *http.Client

	// baseURL is the root of the Egnyte domain, e.g. https://acme.egnyte.com/.
	baseURL   *url.URL
	userAgent string

	// tokenMu guards token so that SetToken/RequestToken can rotate it
	// while other goroutines issue requests on the same client.
	tokenMu sync.RWMutex
	token   string

	// actAs / actAsEmail set the X-Egnyte-Act-As(-Email) impersonation
	// header on every request. Requires an administrator token.
	actAs      string
	actAsEmail string

	// maxResponseBytes caps decoded/drained successful responses; <= 0
	// means unlimited. See WithMaxResponseBytes.
	maxResponseBytes int64

	common service // reused single allocation for all services

	// Services used for talking to the different parts of the API.
	AI                  *AIService
	Agents              *AgentsService
	Audit               *AuditService
	Bookmarks           *BookmarksService
	Comments            *CommentsService
	ControlledDocs      *ControlledDocsService
	DocumentPortal      *DocumentPortalService
	ETMF                *ETMFService
	Events              *EventsService
	FileSystem          *FileSystemService
	FileSystemContent   *FileSystemContentService
	Groups              *GroupsService
	Insights            *InsightsService
	Links               *LinksService
	Metadata            *MetadataService
	MSP                 *MSPService
	Navigate            *NavigateService
	Permissions         *PermissionsService
	Procore             *ProcoreService
	ProjectCustomFields *ProjectCustomFieldsService
	ProjectFolders      *ProjectFoldersService
	Salesforce          *SalesforceService
	Search              *SearchService
	Sign                *SignService
	Tokens              *TokensService
	Trash               *TrashService
	UploadRequests      *UploadRequestsService
	Users               *UsersService
	Webhooks            *WebhooksService
	Workflows           *WorkflowsService
}

type service struct {
	client *Client
}

// ClientOption configures a Client.
type ClientOption func(*Client) error

// WithToken sets the OAuth bearer token used to authenticate requests.
func WithToken(token string) ClientOption {
	return func(c *Client) error {
		c.SetToken(token)
		return nil
	}
}

// WithHTTPClient sets the *http.Client used for requests.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) error {
		if hc == nil {
			return fmt.Errorf("egnyte: http client must not be nil")
		}
		c.httpClient = hc
		return nil
	}
}

// WithBaseURL overrides the domain-derived base URL. Useful for testing
// (it accepts plain-HTTP test servers, unlike NewClient's domain
// argument) and for non-standard deployments. The URL should have a
// trailing slash. Do not pass values derived from untrusted input:
// requests to the base URL carry credentials.
func WithBaseURL(base string) ClientOption {
	return func(c *Client) error {
		u, err := url.Parse(base)
		if err != nil {
			return fmt.Errorf("egnyte: invalid base URL: %w", err)
		}
		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("egnyte: base URL must be absolute with a host: %q", base)
		}
		// Userinfo would be a credential the client cannot redact, and
		// query/fragment components would be silently dropped per
		// request rather than honoured.
		if u.User != nil {
			return fmt.Errorf("egnyte: base URL must not contain userinfo")
		}
		if u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("egnyte: base URL must not contain a query or fragment")
		}
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
		c.baseURL = u
		return nil
	}
}

// WithMaxResponseBytes caps how many bytes of a successful response the
// client will decode or drain, guarding against unexpectedly huge
// replies from a compromised or misbehaving server (relevant when the
// domain comes from untrusted input). A value <= 0 disables the cap.
// Streaming reads — DoRaw and io.Writer destinations, i.e. file
// downloads — are never capped.
func WithMaxResponseBytes(n int64) ClientOption {
	return func(c *Client) error {
		c.maxResponseBytes = n
		return nil
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) error {
		c.userAgent = ua
		return nil
	}
}

// WithActAs makes every request impersonate the given username via the
// X-Egnyte-Act-As header. The token must belong to an administrator.
func WithActAs(username string) ClientOption {
	return func(c *Client) error {
		c.actAs = username
		return nil
	}
}

// WithActAsEmail makes every request impersonate the user with the given
// email via the X-Egnyte-Act-As-Email header. Ignored by the API if
// X-Egnyte-Act-As is also set.
func WithActAsEmail(email string) ClientOption {
	return func(c *Client) error {
		c.actAsEmail = email
		return nil
	}
}

// NewClient returns a client for the given Egnyte domain. The domain may
// be given as a bare subdomain ("acme"), a host ("acme.egnyte.com"), or a
// full URL ("https://acme.egnyte.com").
func NewClient(domain string, opts ...ClientOption) (*Client, error) {
	base, err := domainBaseURL(domain)
	if err != nil {
		return nil, err
	}
	c := &Client{
		httpClient:       &http.Client{},
		baseURL:          base,
		userAgent:        defaultUserAgent,
		maxResponseBytes: defaultMaxResponseBytes,
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("egnyte: nil ClientOption")
		}
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	c.common.client = c
	c.AI = (*AIService)(&c.common)
	c.Agents = (*AgentsService)(&c.common)
	c.Audit = (*AuditService)(&c.common)
	c.Bookmarks = (*BookmarksService)(&c.common)
	c.Comments = (*CommentsService)(&c.common)
	c.ControlledDocs = (*ControlledDocsService)(&c.common)
	c.DocumentPortal = (*DocumentPortalService)(&c.common)
	c.ETMF = (*ETMFService)(&c.common)
	c.Events = (*EventsService)(&c.common)
	c.FileSystem = (*FileSystemService)(&c.common)
	c.FileSystemContent = (*FileSystemContentService)(&c.common)
	c.Groups = (*GroupsService)(&c.common)
	c.Insights = (*InsightsService)(&c.common)
	c.Links = (*LinksService)(&c.common)
	c.Metadata = (*MetadataService)(&c.common)
	c.MSP = (*MSPService)(&c.common)
	c.Navigate = (*NavigateService)(&c.common)
	c.Permissions = (*PermissionsService)(&c.common)
	c.Procore = (*ProcoreService)(&c.common)
	c.ProjectCustomFields = (*ProjectCustomFieldsService)(&c.common)
	c.ProjectFolders = (*ProjectFoldersService)(&c.common)
	c.Salesforce = (*SalesforceService)(&c.common)
	c.Search = (*SearchService)(&c.common)
	c.Sign = (*SignService)(&c.common)
	c.Tokens = (*TokensService)(&c.common)
	c.Trash = (*TrashService)(&c.common)
	c.UploadRequests = (*UploadRequestsService)(&c.common)
	c.Users = (*UsersService)(&c.common)
	c.Webhooks = (*WebhooksService)(&c.common)
	c.Workflows = (*WorkflowsService)(&c.common)
	return c, nil
}

func domainBaseURL(domain string) (*url.URL, error) {
	if domain == "" {
		return nil, fmt.Errorf("egnyte: domain must not be empty")
	}
	s := domain
	if !strings.Contains(s, "://") {
		if !strings.Contains(s, ".") {
			s += ".egnyte.com"
		}
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("egnyte: invalid domain %q: %w", domain, err)
	}
	// Requests carry bearer tokens and OAuth credentials — never allow
	// them over plaintext or to URLs smuggling userinfo. Test servers
	// use the WithBaseURL escape hatch instead.
	if u.Scheme != "https" {
		return nil, fmt.Errorf("egnyte: domain %q must use https", domain)
	}
	if u.User != nil {
		return nil, fmt.Errorf("egnyte: domain %q must not contain userinfo", domain)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("egnyte: domain %q has no host", domain)
	}
	u.Path = "/"
	return u, nil
}

// BaseURL returns a copy of the client's base URL.
func (c *Client) BaseURL() url.URL { return *c.baseURL }

// redacted replaces a secret in formatted output. Secrets are redacted
// only in fmt output — JSON encoding still carries the real values,
// since the API needs them on the wire.
const redacted = "REDACTED"

// String implements fmt.Stringer with the access token redacted, so
// logging the client (including with %+v) never leaks credentials.
func (c *Client) String() string {
	base := *c.baseURL
	// WithBaseURL rejects userinfo, but redact defensively so a client
	// built by other means cannot leak a password through its base URL.
	if base.User != nil {
		base.User = url.User(redacted)
	}
	return fmt.Sprintf("egnyte.Client{baseURL:%s, token:%s}", base.String(), redacted)
}

// GoString implements fmt.GoStringer (%#v) with the token redacted.
func (c *Client) GoString() string { return c.String() }

// EncodePath percent-encodes each segment of a file system path while
// preserving the separating slashes, as required by the File System and
// Permissions APIs. Encoding is stricter than RFC 3986 paths: every byte
// outside the unreserved set (ALPHA / DIGIT / - . _ ~) is escaped, since
// Egnyte requires characters like "$" and "?" encoded in path segments.
func EncodePath(path string) string {
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch {
		case c == '/',
			'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9',
			c == '-', c == '.', c == '_', c == '~':
			b.WriteByte(c)
		default:
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}

// NewRequest creates an API request against the Public API base path
// (/pubapi). urlPath is relative to it, e.g. "v1/userinfo". If body is
// non-nil it is JSON-encoded.
func (c *Client) NewRequest(ctx context.Context, method, urlPath string, body any) (*http.Request, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("egnyte: encode %s %s request: %w", method, urlPath, err)
		}
		buf = bytes.NewReader(b)
	}
	req, err := c.newRequest(ctx, method, apiBasePath+"/"+strings.TrimPrefix(urlPath, "/"), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// NewFormRequest creates an API request with a form-encoded body against
// the Public API base path (/pubapi).
func (c *Client) NewFormRequest(ctx context.Context, method, urlPath string, form url.Values) (*http.Request, error) {
	req, err := c.newRequest(ctx, method, apiBasePath+"/"+strings.TrimPrefix(urlPath, "/"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// resolvePath builds the absolute request URL for an API path under the
// client's base URL.
//
// It deliberately does NOT use (*url.URL).Parse / ResolveReference:
// those apply RFC 3986 dot-segment removal, so a path containing "." or
// ".." could climb out of its API route family — e.g. a caller passing
// an untrusted "/../users/123" to FileSystem.Delete would send an
// authenticated DELETE to /pubapi/v1/users/123. Dot segments are
// rejected instead, in both literal and percent-encoded form, and the
// path is joined onto the base path verbatim.
func (c *Client) resolvePath(absPath string) (*url.URL, error) {
	if !strings.HasPrefix(absPath, "/") {
		return nil, fmt.Errorf("egnyte: API path must be absolute: %q", absPath)
	}
	ref, err := url.ParseRequestURI(absPath)
	if err != nil {
		return nil, fmt.Errorf("egnyte: invalid API path %q: %w", absPath, err)
	}
	if ref.Scheme != "" || ref.Host != "" || ref.User != nil {
		return nil, fmt.Errorf("egnyte: API path must not be a full URL: %q", absPath)
	}
	for _, segment := range strings.Split(ref.EscapedPath(), "/") {
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			return nil, fmt.Errorf("egnyte: invalid API path segment %q: %w", segment, err)
		}
		if decoded == "." || decoded == ".." {
			return nil, fmt.Errorf("egnyte: dot path segment %q is not allowed in %q", decoded, absPath)
		}
	}

	u := *c.baseURL
	base := strings.TrimSuffix(u.Path, "/")
	u.Path = base + ref.Path
	if raw := ref.EscapedPath(); raw != ref.Path {
		u.RawPath = base + raw
	} else {
		u.RawPath = ""
	}
	u.RawQuery = ref.RawQuery
	u.Fragment, u.RawFragment = "", ""
	return &u, nil
}

// newRequest creates a request for an absolute path under the domain base
// URL (used directly for endpoints outside /pubapi, such as /puboauth).
func (c *Client) newRequest(ctx context.Context, method, absPath string, body io.Reader) (*http.Request, error) {
	u, err := c.resolvePath(absPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if token := c.tokenValue(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("User-Agent", c.userAgent)
	if c.actAs != "" {
		req.Header.Set("X-Egnyte-Act-As", c.actAs)
	}
	if c.actAsEmail != "" {
		req.Header.Set("X-Egnyte-Act-As-Email", c.actAsEmail)
	}
	return req, nil
}

// Rate reports the per-token rate limit state parsed from response headers.
type Rate struct {
	// QPSAllotted and QPSCurrent report the queries-per-second limit and
	// the queries consumed in the current second.
	QPSAllotted int
	QPSCurrent  int
	// QuotaAllotted and QuotaCurrent report the daily quota and the
	// queries consumed today (resets at 00:00 UTC).
	QuotaAllotted int
	QuotaCurrent  int
}

// Response wraps *http.Response and carries Egnyte rate limit information.
type Response struct {
	*http.Response
	Rate Rate
}

func newResponse(r *http.Response) *Response {
	resp := &Response{Response: r}
	resp.Rate = Rate{
		QPSAllotted:   headerInt(r, "X-Accesstoken-Qps-Allotted"),
		QPSCurrent:    headerInt(r, "X-Accesstoken-Qps-Current"),
		QuotaAllotted: headerInt(r, "X-Accesstoken-Quota-Allotted"),
		QuotaCurrent:  headerInt(r, "X-Accesstoken-Quota-Current"),
	}
	return resp
}

func headerInt(r *http.Response, name string) int {
	n, _ := strconv.Atoi(r.Header.Get(name))
	return n
}

// ErrResponseTooLarge is returned when a successful response exceeds the
// client's response size cap. See WithMaxResponseBytes.
var ErrResponseTooLarge = errors.New("egnyte: response body too large")

// maxBodyReader fails once more than limit bytes have been read, so an
// oversized response is reported instead of being buffered in full.
type maxBodyReader struct {
	r     io.Reader
	limit int64
	read  int64
}

func (m *maxBodyReader) Read(p []byte) (int, error) {
	if m.read > m.limit {
		return 0, ErrResponseTooLarge
	}
	// Allow one byte past the limit so an exactly-at-limit body still
	// decodes but an oversized one is detected.
	if room := m.limit + 1 - m.read; int64(len(p)) > room {
		p = p[:room]
	}
	n, err := m.r.Read(p)
	m.read += int64(n)
	if m.read > m.limit {
		return n, ErrResponseTooLarge
	}
	return n, err
}

// decodeJSON decodes a single JSON value from r into v, bounded by limit
// (<= 0 means unbounded), and rejects trailing data after that value.
func decodeJSON(r io.Reader, v any, limit int64) error {
	if limit > 0 {
		r = &maxBodyReader{r: r, limit: limit}
	}
	dec := json.NewDecoder(r)
	if err := dec.Decode(v); err != nil && err != io.EOF {
		return err
	}
	// A well-formed response carries exactly one JSON value; anything
	// after it means the body was not what it claimed to be.
	if _, err := dec.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("egnyte: unexpected trailing data after JSON value")
	}
	return nil
}

// drain discards up to the response cap so the connection can be reused.
// An oversized body is abandoned rather than drained indefinitely.
func (c *Client) drain(body io.Reader) {
	if c.maxResponseBytes > 0 {
		_, _ = io.CopyN(io.Discard, body, c.maxResponseBytes)
		return
	}
	_, _ = io.Copy(io.Discard, body)
}

// Do sends an API request. If v is non-nil the response body is JSON-decoded
// into it; if v is an io.Writer the raw body is copied instead. Responses
// with status >= 400 return an *APIError.
func (c *Client) Do(req *http.Request, v any) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("egnyte: request must not be nil")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response := newResponse(resp)
	if resp.StatusCode >= 400 {
		return response, newAPIError(resp)
	}
	if v == nil {
		c.drain(resp.Body)
		return response, nil
	}
	// Streaming destinations are the caller's to bound.
	if w, ok := v.(io.Writer); ok {
		_, err = io.Copy(w, resp.Body)
		return response, err
	}
	if err := decodeJSON(resp.Body, v, c.maxResponseBytes); err != nil {
		return response, fmt.Errorf("egnyte: decode %s %s response: %w",
			req.Method, req.URL.Path, err)
	}
	return response, nil
}

// DoRaw sends an API request and returns the raw response body for
// streaming, e.g. file downloads. The caller must close the returned
// ReadCloser. Responses with status >= 400 return an *APIError and no body.
func (c *Client) DoRaw(req *http.Request) (io.ReadCloser, *Response, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("egnyte: request must not be nil")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	response := newResponse(resp)
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, response, newAPIError(resp)
	}
	return resp.Body, response, nil
}

// Bool, Int and String return pointers to the given value. They are
// convenience helpers for filling optional request fields.
func Bool(v bool) *bool       { return &v }
func Int(v int) *int          { return &v }
func String(v string) *string { return &v }

// LenientBool is a bool that also decodes from the JSON string forms
// "true"/"false", which some live endpoints return where their spec
// declares a boolean (observed on the Users API).
type LenientBool bool

func (b *LenientBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "null" || s == "" {
		*b = false
		return nil
	}
	v, err := strconv.ParseBool(strings.ToLower(s))
	if err != nil {
		return fmt.Errorf("egnyte: cannot parse %s as bool", data)
	}
	*b = LenientBool(v)
	return nil
}
