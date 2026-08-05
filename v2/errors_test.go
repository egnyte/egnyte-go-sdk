package egnyte

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAPIError_errorsEnvelope(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"Errors":[{"description":"Link does not exist.","code":"404"}]}`))
	})

	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	_, err := client.Do(req, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if len(apiErr.Errors) != 1 || apiErr.Errors[0].Description != "Link does not exist." {
		t.Errorf("Errors = %+v, want one entry with description", apiErr.Errors)
	}
	if !strings.Contains(apiErr.Error(), "Link does not exist.") {
		t.Errorf("Error() = %q, should contain description", apiErr.Error())
	}
}

func TestAPIError_messageAndRetryAfter(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "17")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"Developer Over Qps"}`))
	})

	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	_, err := client.Do(req, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Message != "Developer Over Qps" {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if apiErr.RetryAfter != 17*time.Second {
		t.Errorf("RetryAfter = %v, want 17s", apiErr.RetryAfter)
	}
	if !apiErr.IsRateLimit() {
		t.Error("IsRateLimit() = false, want true for 429")
	}
}

func TestAPIError_lowercaseErrorsEnvelope(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors":[{"msg":"Invalid date range specified.","code":"INVALID_DATE_RANGE"}]}`))
	})

	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	_, err := client.Do(req, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if len(apiErr.Errors) != 1 ||
		apiErr.Errors[0].Description != "Invalid date range specified." ||
		apiErr.Errors[0].Code != "INVALID_DATE_RANGE" {
		t.Errorf("Errors = %+v", apiErr.Errors)
	}
	if !strings.Contains(apiErr.Error(), "Invalid date range specified.") {
		t.Errorf("Error() = %q, should contain description", apiErr.Error())
	}
}

func TestAPIError_inputErrorsEnvelope(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"inputErrors":{"helpText":[{"msg":"This field is required.","code":"MISSING_REQUIRED_FIELD"}]}}`))
	})

	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	_, err := client.Do(req, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if len(apiErr.Errors) != 1 ||
		apiErr.Errors[0].Description != "helpText: This field is required." ||
		apiErr.Errors[0].Code != "MISSING_REQUIRED_FIELD" {
		t.Errorf("Errors = %+v", apiErr.Errors)
	}
}

// The OAuth token endpoint signals throttling with 409; everywhere else
// 409 is an ordinary conflict and must not be reported as a rate limit
// even if the response happens to carry Retry-After.
func TestAPIError_conflictClassification(t *testing.T) {
	client, mux := setup(t)
	conflict := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message":"conflict"}`))
	}
	mux.HandleFunc("/puboauth/token", conflict)
	mux.HandleFunc("/pubapi/v1/fs/Shared/dup", conflict)
	ctx := context.Background()

	_, _, err := client.RequestToken(ctx, PasswordCredentials{ClientID: "k", Username: "u", Password: "p"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("oauth error type = %T", err)
	}
	if !apiErr.IsRateLimit() {
		t.Error("409 on the OAuth endpoint should count as a rate limit")
	}
	if apiErr.RetryAfter != 30*time.Second {
		t.Errorf("RetryAfter = %v", apiErr.RetryAfter)
	}

	// Same status and header, ordinary endpoint: a real conflict.
	_, err = client.FileSystem.CreateFolder(ctx, "/Shared/dup")
	if !errors.As(err, &apiErr) {
		t.Fatalf("fs error type = %T", err)
	}
	if apiErr.IsRateLimit() {
		t.Error("409 on a normal endpoint must not be classified as a rate limit")
	}
}

func TestAPIError_nonJSONBody(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("gateway exploded"))
	})

	req, _ := client.NewRequest(context.Background(), http.MethodGet, "v1/userinfo", nil)
	_, err := client.Do(req, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if string(apiErr.Body) != "gateway exploded" {
		t.Errorf("Body = %q, want raw body preserved", apiErr.Body)
	}
}
