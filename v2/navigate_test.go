package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestNavigateService_Navigate(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/navigate", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["embedded"] != true || body["path"] != "/Shared/Projects/Project Alpha" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"redirect":"https://test.egnyte.com/navigate/temp/2c7e1b8b"}`))
	})

	redirect, _, err := client.Navigate.Navigate(context.Background(), true, "/Shared/Projects/Project Alpha")
	if err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	if redirect != "https://test.egnyte.com/navigate/temp/2c7e1b8b" {
		t.Errorf("redirect = %q", redirect)
	}
}

func TestNavigateService_Navigate_defaultPath(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/navigate", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["embedded"] != false {
			t.Errorf("embedded = %v", body["embedded"])
		}
		if _, ok := body["path"]; ok {
			t.Error("empty path should be omitted")
		}
		w.Write([]byte(`{"redirect":"https://test.egnyte.com/navigate/temp/xyz"}`))
	})

	if _, _, err := client.Navigate.Navigate(context.Background(), false, ""); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
}

func TestNavigateService_NavigateV1_redirect(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/navigate/embedded/folder/Shared/test", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		w.Header().Set("Location", "https://test.egnyte.com/navigate/temp/abc123")
		w.WriteHeader(http.StatusSeeOther)
	})
	mux.HandleFunc("/navigate/temp/abc123", func(w http.ResponseWriter, r *http.Request) {
		t.Error("NavigateV1 must not follow the 303 redirect")
	})

	redirect, resp, err := client.Navigate.NavigateV1(context.Background(), "folder/Shared/test")
	if err != nil {
		t.Fatalf("NavigateV1: %v", err)
	}
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("status = %d, want 303", resp.StatusCode)
	}
	if redirect != "https://test.egnyte.com/navigate/temp/abc123" {
		t.Errorf("redirect = %q", redirect)
	}
}

func TestNavigateService_Navigate_missingScope(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/navigate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Token missing Egnyte.launchwebsession scope"}`))
	})

	_, _, err := client.Navigate.Navigate(context.Background(), false, "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError = %+v", apiErr)
	}
}
