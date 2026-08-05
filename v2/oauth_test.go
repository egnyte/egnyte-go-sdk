package egnyte

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRequestToken_passwordGrant(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/puboauth/token", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "Content-Type", "application/x-www-form-urlencoded")
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization header leaked to token endpoint: %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		want := url.Values{
			"grant_type":    {"password"},
			"client_id":     {"key123"},
			"client_secret": {"sec456"},
			"username":      {"jsmith"},
			"password":      {"hunter2"},
			"scope":         {"Egnyte.filesystem Egnyte.link"},
		}
		for k, v := range want {
			if got := r.PostForm.Get(k); got != v[0] {
				t.Errorf("form[%s] = %q, want %q", k, got, v[0])
			}
		}
		w.Write([]byte(`{"access_token":"at1","refresh_token":"rt1","token_type":"bearer","expires_in":2592000}`))
	})

	tok, _, err := client.RequestToken(context.Background(), PasswordCredentials{
		ClientID:     "key123",
		ClientSecret: "sec456",
		Username:     "jsmith",
		Password:     "hunter2",
		Scopes:       []string{"Egnyte.filesystem", "Egnyte.link"},
	})
	if err != nil {
		t.Fatalf("RequestToken: %v", err)
	}
	if tok.AccessToken != "at1" || tok.RefreshToken != "rt1" || tok.ExpiresIn != 2592000 {
		t.Errorf("Token = %+v", tok)
	}
	// The client should now authenticate with the new token.
	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	testHeader(t, req, "Authorization", "Bearer at1")
}

func TestRequestToken_grantForms(t *testing.T) {
	auth := AuthorizationCode{
		ClientID:    "key",
		RedirectURI: "https://app.example.com/oauth",
		Code:        "code1",
	}.values()
	if auth.Get("grant_type") != "authorization_code" || auth.Get("code") != "code1" {
		t.Errorf("AuthorizationCode values = %v", auth)
	}
	if auth.Has("client_secret") {
		t.Error("empty client_secret should be omitted")
	}

	refresh := RefreshGrant{ClientID: "key", ClientSecret: "sec", RefreshToken: "rt1"}.values()
	if refresh.Get("grant_type") != "refresh_token" || refresh.Get("refresh_token") != "rt1" {
		t.Errorf("RefreshGrant values = %v", refresh)
	}
}

func TestRequestToken_nilGrant(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.RequestToken(context.Background(), nil); err == nil {
		t.Error("nil grant should fail before sending")
	}
}

func TestOAuth_redactsSecrets(t *testing.T) {
	values := []any{
		Token{AccessToken: "at-secret", RefreshToken: "rt-secret", TokenType: "bearer"},
		PasswordCredentials{ClientID: "key", ClientSecret: "cs-secret", Username: "jsmith", Password: "pw-secret"},
		AuthorizationCode{ClientID: "key", ClientSecret: "cs-secret", Code: "code-secret"},
		RefreshGrant{ClientID: "key", ClientSecret: "cs-secret", RefreshToken: "rt-secret"},
	}
	for _, v := range values {
		for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
			got := fmt.Sprintf(format, v)
			for _, secret := range []string{"at-secret", "rt-secret", "cs-secret", "pw-secret", "code-secret"} {
				if strings.Contains(got, secret) {
					t.Errorf("fmt.Sprintf(%q, %T) leaks %q: %s", format, v, secret, got)
				}
			}
		}
	}
	// JSON round-tripping must still carry the real token so callers can
	// cache it.
	data, err := json.Marshal(Token{AccessToken: "at1", RefreshToken: "rt1"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), "at1") || !strings.Contains(string(data), "rt1") {
		t.Errorf("Token JSON = %s, want real values preserved", data)
	}
}

func TestAuthorizationURL(t *testing.T) {
	client, err := NewClient("acme")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got := client.AuthorizationURL("key123", "https://app.example.com/oauth", "code", "st8",
		[]string{"Egnyte.filesystem", "Egnyte.link"})

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("Parse(%q): %v", got, err)
	}
	if !strings.HasPrefix(got, "https://acme.egnyte.com/puboauth/token?") {
		t.Errorf("URL = %q, want acme.egnyte.com/puboauth/token prefix", got)
	}
	q := u.Query()
	if q.Get("response_type") != "code" || q.Get("state") != "st8" ||
		q.Get("scope") != "Egnyte.filesystem Egnyte.link" {
		t.Errorf("query = %v", q)
	}
}
