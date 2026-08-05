package egnyte

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestTokensService_UserInfo(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		testHeader(t, r, "Authorization", "Bearer test-token")
		w.Write([]byte(`{"id":123,"first_name":"Test","last_name":"User","username":"test",
			"user_type":"admin","email":"test@example.com"}`))
	})

	info, _, err := client.Tokens.UserInfo(context.Background())
	if err != nil {
		t.Fatalf("UserInfo: %v", err)
	}
	want := UserInfo{ID: 123, FirstName: "Test", LastName: "User", Username: "test",
		UserType: "admin", Email: "test@example.com"}
	if *info != want {
		t.Errorf("UserInfo = %+v, want %+v", *info, want)
	}
}

func TestTokensService_Revoke(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/tokens/revoke", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "Content-Type", "application/x-www-form-urlencoded")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.PostForm.Get("token"); got != "tok-to-revoke" {
			t.Errorf("form[token] = %q", got)
		}
		if got := r.PostForm.Get("client_secret"); got != "sec456" {
			t.Errorf("form[client_secret] = %q", got)
		}
	})

	if _, err := client.Tokens.Revoke(context.Background(), "tok-to-revoke", "sec456"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}

func TestTokensService_UserInfo_unauthorized(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Invalid token"}`))
	})

	_, _, err := client.Tokens.UserInfo(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Message != "Invalid token" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
