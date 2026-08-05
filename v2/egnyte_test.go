package egnyte

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// setup returns a test client wired to a local test server, the mux to
// register handlers on, and a teardown function. This is the shared
// fixture all service tests build on.
func setup(t *testing.T) (*Client, *http.ServeMux) {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := NewClient("test", WithToken("test-token"), WithBaseURL(server.URL+"/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, mux
}

func testMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Errorf("request method = %q, want %q", r.Method, want)
	}
}

func testHeader(t *testing.T, r *http.Request, header, want string) {
	t.Helper()
	if got := r.Header.Get(header); got != want {
		t.Errorf("header %s = %q, want %q", header, got, want)
	}
}

func TestDomainBaseURL(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"acme", "https://acme.egnyte.com/"},
		{"acme.egnyte.com", "https://acme.egnyte.com/"},
		{"https://acme.egnyte.com", "https://acme.egnyte.com/"},
		{"https://acme.egnyte.com/some/path", "https://acme.egnyte.com/"},
		// Gov cloud domains work by passing the full host.
		{"acme.egnytegov.com", "https://acme.egnytegov.com/"},
		{"https://acme.egnytegov.com", "https://acme.egnytegov.com/"},
	}
	for _, tt := range tests {
		u, err := domainBaseURL(tt.domain)
		if err != nil {
			t.Errorf("domainBaseURL(%q): %v", tt.domain, err)
			continue
		}
		if u.String() != tt.want {
			t.Errorf("domainBaseURL(%q) = %q, want %q", tt.domain, u, tt.want)
		}
	}
	// Tokens must never travel over plaintext HTTP, embedded credentials
	// are an injection hazard, and a schemeful-but-hostless URL would
	// silently produce requests to nowhere.
	rejected := []string{
		"",
		"http://acme.egnyte.com",
		"ftp://acme.egnyte.com",
		"https://user:pass@acme.egnyte.com",
		"https://",
	}
	for _, domain := range rejected {
		if _, err := domainBaseURL(domain); err == nil {
			t.Errorf("domainBaseURL(%q) should fail", domain)
		}
	}
}

// TestRouteContainment_dotSegments proves that dot segments cannot move
// a request out of its API route family. Before this was enforced,
// relative-reference resolution turned FileSystem.Delete("/../users/1")
// into an authenticated DELETE /pubapi/v1/users/1.
func TestRouteContainment_dotSegments(t *testing.T) {
	client, mux := setup(t)
	var reached []string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reached = append(reached, r.Method+" "+r.URL.EscapedPath())
		w.Write([]byte(`{}`))
	})
	ctx := context.Background()

	// Every builder that takes caller-supplied path or ID input.
	calls := []struct {
		name string
		call func(string) error
	}{
		{"FileSystem.Get", func(p string) error { _, _, err := client.FileSystem.Get(ctx, p, nil); return err }},
		{"FileSystem.GetV2", func(p string) error { _, _, err := client.FileSystem.GetV2(ctx, p, nil); return err }},
		{"FileSystem.Delete", func(p string) error { _, err := client.FileSystem.Delete(ctx, p, nil); return err }},
		{"FileSystem.CreateFolder", func(p string) error { _, err := client.FileSystem.CreateFolder(ctx, p); return err }},
		{"FileSystem.Lock", func(p string) error { _, _, err := client.FileSystem.Lock(ctx, p); return err }},
		{"FileSystem.GetFileByID", func(p string) error { _, _, err := client.FileSystem.GetFileByID(ctx, p, nil); return err }},
		{"FileSystem.GetFolderByID", func(p string) error { _, _, err := client.FileSystem.GetFolderByID(ctx, p, nil); return err }},
		{"FileSystem.FolderStatsByID", func(p string) error { _, _, err := client.FileSystem.FolderStatsByID(ctx, p); return err }},
		{"FileSystemContent.Download", func(p string) error { _, _, err := client.FileSystemContent.Download(ctx, p, nil); return err }},
		{"FileSystemContent.DownloadByID", func(p string) error { _, _, err := client.FileSystemContent.DownloadByID(ctx, p, nil); return err }},
		{"FileSystemContent.Upload", func(p string) error {
			_, _, err := client.FileSystemContent.Upload(ctx, p, strings.NewReader("x"), nil)
			return err
		}},
		{"Permissions.Get", func(p string) error { _, _, err := client.Permissions.Get(ctx, p); return err }},
		{"Permissions.GetEffective", func(p string) error { _, _, err := client.Permissions.GetEffective(ctx, p, "/Shared"); return err }},
		{"Links.Get", func(p string) error { _, _, err := client.Links.Get(ctx, p); return err }},
		{"Links.Delete", func(p string) error { _, err := client.Links.Delete(ctx, p); return err }},
		{"Metadata.DeleteNamespace", func(p string) error { _, err := client.Metadata.DeleteNamespace(ctx, p, false); return err }},
	}
	// Literal and percent-encoded dot segments, at the start, in the
	// middle, and as a bare ID. The invariant under test is containment,
	// not rejection: a payload is safe either because it is refused, or
	// because it was escaped into a single literal path segment. What
	// must never happen is a request landing outside its route family.
	payloads := []string{
		"/../users/123",
		"/Shared/../../v1/users/123",
		"..",
		".",
		"/%2e%2e/users/123",
		"/Shared/%2E%2E/%2e%2e/v1/users/123",
		"../../../puboauth/token",
	}
	for _, c := range calls {
		// Establish the static route prefix from a known-benign call by
		// locating where the caller-supplied part begins.
		reached = reached[:0]
		if err := c.call("BENIGNID"); err != nil {
			t.Fatalf("%s(benign): %v", c.name, err)
		}
		if len(reached) != 1 {
			t.Fatalf("%s(benign) reached %v", c.name, reached)
		}
		benign := reached[0][strings.Index(reached[0], " ")+1:]
		i := strings.Index(benign, "BENIGNID")
		if i < 0 {
			t.Fatalf("%s(benign) path %q does not contain the marker", c.name, benign)
		}
		prefix := benign[:i]

		for _, p := range payloads {
			reached = reached[:0]
			err := c.call(p)
			for _, got := range reached {
				path := got[strings.Index(got, " ")+1:]
				if !strings.HasPrefix(path, prefix) {
					t.Errorf("%s(%q) escaped its route family %q: %s (err=%v)", c.name, p, prefix, got, err)
					continue
				}
				// A dot segment must never survive to the wire.
				for _, seg := range strings.Split(path, "/") {
					if dec, _ := url.PathUnescape(seg); dec == "." || dec == ".." {
						t.Errorf("%s(%q) sent a dot segment: %s", c.name, p, got)
					}
				}
			}
		}
	}
	reached = reached[:0]

	// Names that merely contain dots stay legal.
	for _, p := range []string{"/Shared/..hidden/f.txt", "/Shared/a.b.c", "/Shared/...", "/Shared/.env"} {
		if _, _, err := client.FileSystem.Get(ctx, p, nil); err != nil {
			t.Errorf("legitimate path %q rejected: %v", p, err)
		}
	}
	if len(reached) != 4 {
		t.Errorf("legitimate paths reached = %v, want 4", reached)
	}
	for _, got := range reached {
		if !strings.HasPrefix(got, "GET /pubapi/v1/fs/Shared/") {
			t.Errorf("request left the fs route family: %q", got)
		}
	}
}

// A URL-looking API path must never send the bearer token to another
// host; it stays a literal path under the client's own base URL.
func TestNewRequest_urlLikePathStaysOnHost(t *testing.T) {
	client, err := NewClient("acme", WithToken("tok"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()
	for _, p := range []string{
		"https://evil.example.com/steal",
		"//evil.example.com/steal",
		"v1/fs/../../../../evil.example.com",
	} {
		req, err := client.NewRequest(ctx, http.MethodGet, p, nil)
		if err != nil {
			continue // rejected outright is also fine
		}
		if req.URL.Host != "acme.egnyte.com" {
			t.Errorf("NewRequest(%q) targets host %q, want acme.egnyte.com", p, req.URL.Host)
		}
		if !strings.HasPrefix(req.URL.EscapedPath(), "/pubapi/") {
			t.Errorf("NewRequest(%q) left /pubapi: %s", p, req.URL.EscapedPath())
		}
	}
	// The internal absolute-path builder rejects full URLs outright.
	if _, err := client.newRequest(ctx, http.MethodGet, "https://evil.example.com/x", nil); err == nil {
		t.Error("newRequest should reject an absolute URL")
	}
	if _, err := client.newRequest(ctx, http.MethodGet, "relative/path", nil); err == nil {
		t.Error("newRequest should reject a non-absolute path")
	}
}

func TestNewClient_optionValidation(t *testing.T) {
	if _, err := NewClient("acme", nil); err == nil {
		t.Error("nil ClientOption should fail")
	}
	if _, err := NewClient("acme", WithHTTPClient(nil)); err == nil {
		t.Error("WithHTTPClient(nil) should fail")
	}
}

func TestNewRequest_encodeErrorWrapped(t *testing.T) {
	client, err := NewClient("acme", WithToken("tok"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.NewRequest(context.Background(), http.MethodPost, "v1/links", make(chan int))
	if err == nil {
		t.Fatal("marshalling a channel should fail")
	}
	if !strings.Contains(err.Error(), "encode POST v1/links request") {
		t.Errorf("error = %q, want method and path context", err)
	}
}

func TestDo_decodeErrorWrapped(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"username": 42}`))
	})

	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	var out struct {
		Username string `json:"username"`
	}
	_, err := client.Do(req, &out)
	if err == nil {
		t.Fatal("decoding a number into a string should fail")
	}
	if !strings.Contains(err.Error(), "decode GET /pubapi/v1/userinfo response") {
		t.Errorf("error = %q, want method and path context", err)
	}
}

func TestDo_responseSizeLimit(t *testing.T) {
	client, mux := setup(t)
	var payload []byte
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	})
	// A JSON string of n bytes total on the wire.
	body := func(total int) []byte {
		b := make([]byte, total)
		for i := range b {
			b[i] = 'a'
		}
		b[0], b[total-1] = '"', '"'
		return b
	}

	call := func(limit int64, size int) error {
		client.maxResponseBytes = limit
		payload = body(size)
		req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
		var out string
		_, err := client.Do(req, &out)
		return err
	}

	if err := call(100, 100); err != nil {
		t.Errorf("exactly-at-limit response should decode: %v", err)
	}
	if err := call(100, 99); err != nil {
		t.Errorf("under-limit response should decode: %v", err)
	}
	err := call(100, 101)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Errorf("over-limit error = %v, want ErrResponseTooLarge", err)
	}
	if err := call(0, 5000); err != nil {
		t.Errorf("limit disabled should decode any size: %v", err)
	}
}

func TestDo_rejectsTrailingJSON(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		// One valid value followed by a second — a protocol violation
		// that previously decoded silently as just the first value.
		w.Write([]byte(`{"username":"a"}{"username":"b"}`))
	})
	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	var out struct {
		Username string `json:"username"`
	}
	if _, err := client.Do(req, &out); err == nil {
		t.Error("trailing JSON value should be rejected")
	} else if !strings.Contains(err.Error(), "trailing data") {
		t.Errorf("error = %v, want trailing-data error", err)
	}
}

func TestDo_emptyAndIgnoredBodies(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/empty", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/pubapi/v1/big", func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte("x"), 1<<20))
	})
	ctx := context.Background()

	// 204 with no body decodes to an untouched value, not an error.
	req, _ := client.NewRequest(ctx, http.MethodGet, "v1/empty", nil)
	var out struct {
		Username string `json:"username"`
	}
	if _, err := client.Do(req, &out); err != nil {
		t.Errorf("empty 204 body: %v", err)
	}

	// An ignored oversized body is abandoned, not drained forever.
	client.maxResponseBytes = 1024
	req, _ = client.NewRequest(ctx, http.MethodGet, "v1/big", nil)
	if _, err := client.Do(req, nil); err != nil {
		t.Errorf("ignored large body: %v", err)
	}
}

func TestDo_nilRequest(t *testing.T) {
	client, _ := setup(t)
	if _, err := client.Do(nil, nil); err == nil {
		t.Error("Do(nil) should fail rather than panic")
	}
	if _, _, err := client.DoRaw(nil); err == nil {
		t.Error("DoRaw(nil) should fail rather than panic")
	}
}

func TestDo_cancellationDuringDecode(t *testing.T) {
	client, mux := setup(t)
	release := make(chan struct{})
	mux.HandleFunc("/pubapi/v1/slow", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"a":`))
		w.(http.Flusher).Flush()
		<-release
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := client.NewRequest(ctx, http.MethodGet, "v1/slow", nil)
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	var out map[string]any
	if _, err := client.Do(req, &out); err == nil {
		t.Error("cancelled decode should return an error")
	}
}

func TestSetToken_concurrent(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	// Rotate the token while requests are in flight; run with -race this
	// verifies SetToken and request building don't race on the token.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			client.SetToken(fmt.Sprintf("token-%d", i))
		}(i)
		go func() {
			defer wg.Done()
			req, err := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
			if err != nil {
				t.Errorf("NewRequest: %v", err)
				return
			}
			if _, err := client.Do(req, nil); err != nil {
				t.Errorf("Do: %v", err)
			}
		}()
	}
	wg.Wait()
}

// TestRedaction_allSecretTypes covers every exported type carrying a
// credential: routine %v/%+v/%#v logging must never print the secret,
// while JSON encoding must still carry it for the wire.
func TestRedaction_allSecretTypes(t *testing.T) {
	const secret = "s3cr3t-value"
	cases := []struct {
		name      string
		value     any
		jsonField string // must still appear in JSON output
	}{
		{"Token.AccessToken", Token{AccessToken: secret, TokenType: "bearer"}, "access_token"},
		{"Token.RefreshToken", Token{RefreshToken: secret}, "refresh_token"},
		{"PasswordCredentials", PasswordCredentials{ClientID: "k", ClientSecret: secret, Password: secret}, ""},
		{"AuthorizationCode", AuthorizationCode{ClientID: "k", ClientSecret: secret, Code: secret}, ""},
		{"RefreshGrant", RefreshGrant{ClientID: "k", ClientSecret: secret, RefreshToken: secret}, ""},
		{"CreateLinkRequest.Password", CreateLinkRequest{Path: "/Shared/f", Type: "file", Password: secret}, "password"},
		{"FileLock.LockToken", FileLock{LockToken: secret, Timeout: 3599}, "lock_token"},
		{"WebhookRequest.AuthHeaderValue", WebhookRequest{URL: "https://h/x", AuthHeaderValue: secret}, "authHeaderValue"},
		{"MSPWebhookRequest.AuthHeaderValue", MSPWebhookRequest{WebhookURL: "https://h/x", AuthHeaderValue: secret}, "auth_header_value"},
		{"MSPWebhookDetails.AuthHeaderValue", MSPWebhookDetails{WebhookID: "w1", AuthHeaderValue: secret}, "auth_header_value"},
		{"MSPCreateTrialRequest.Password", MSPCreateTrialRequest{Domain: "d", Password: secret}, "password"},
	}
	for _, tc := range cases {
		for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
			if got := fmt.Sprintf(format, tc.value); strings.Contains(got, secret) {
				t.Errorf("%s: fmt.Sprintf(%q, ...) leaks the secret: %s", tc.name, format, got)
			}
		}
		// Also check the pointer form, which is what callers often log.
		ptr := reflect.New(reflect.TypeOf(tc.value))
		ptr.Elem().Set(reflect.ValueOf(tc.value))
		for _, format := range []string{"%v", "%+v", "%s"} {
			if got := fmt.Sprintf(format, ptr.Interface()); strings.Contains(got, secret) {
				t.Errorf("%s: pointer fmt.Sprintf(%q, ...) leaks the secret: %s", tc.name, format, got)
			}
		}
		if tc.jsonField == "" {
			continue
		}
		data, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatalf("%s: Marshal: %v", tc.name, err)
		}
		if !strings.Contains(string(data), secret) {
			t.Errorf("%s: JSON must still carry the real value, got %s", tc.name, data)
		}
		if !strings.Contains(string(data), tc.jsonField) {
			t.Errorf("%s: JSON missing field %q: %s", tc.name, tc.jsonField, data)
		}
	}
}

func TestWithBaseURL_validation(t *testing.T) {
	rejected := []string{
		"https://user:pass@example.com/",
		"not-a-url",
		"/relative/only",
		"https://example.com/?a=b",
		"https://example.com/#frag",
	}
	for _, base := range rejected {
		if _, err := NewClient("acme", WithBaseURL(base)); err == nil {
			t.Errorf("WithBaseURL(%q) should be rejected", base)
		}
	}
	// Plain HTTP stays allowed: httptest servers depend on it.
	if _, err := NewClient("acme", WithBaseURL("http://127.0.0.1:8080/")); err != nil {
		t.Errorf("WithBaseURL(http) should be allowed: %v", err)
	}
}

func TestClient_redactsToken(t *testing.T) {
	client, err := NewClient("acme", WithToken("s3cret-token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		if got := fmt.Sprintf(format, client); strings.Contains(got, "s3cret-token") {
			t.Errorf("fmt.Sprintf(%q, client) leaks the token: %s", format, got)
		}
	}
}

func TestEncodePath(t *testing.T) {
	// Example straight from the best-practices doc: encode segments,
	// never the separating slashes.
	got := EncodePath("/Shared/example?path/$file.txt")
	want := "/Shared/example%3Fpath/%24file.txt"
	if got != want {
		t.Errorf("EncodePath = %q, want %q", got, want)
	}
}

func TestNewRequest_headers(t *testing.T) {
	client, err := NewClient("acme",
		WithToken("tok123"),
		WithActAs("jsmith"),
		WithActAsEmail("jsmith@example.com"),
		WithUserAgent("custom-agent"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	req, err := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if got, want := req.URL.String(), "https://acme.egnyte.com/pubapi/v1/userinfo"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
	testHeader(t, req, "Authorization", "Bearer tok123")
	testHeader(t, req, "X-Egnyte-Act-As", "jsmith")
	testHeader(t, req, "X-Egnyte-Act-As-Email", "jsmith@example.com")
	testHeader(t, req, "User-Agent", "custom-agent")
}

func TestDo_rateHeaders(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Accesstoken-Qps-Allotted", "2")
		w.Header().Set("X-Accesstoken-Qps-Current", "1")
		w.Header().Set("X-Accesstoken-Quota-Allotted", "1000")
		w.Header().Set("X-Accesstoken-Quota-Current", "42")
		w.Write([]byte(`{}`))
	})

	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	resp, err := client.Do(req, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	want := Rate{QPSAllotted: 2, QPSCurrent: 1, QuotaAllotted: 1000, QuotaCurrent: 42}
	if resp.Rate != want {
		t.Errorf("Rate = %+v, want %+v", resp.Rate, want)
	}
}
