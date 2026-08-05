package egnyte

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestInsightsService_RecentFiles(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/insights/files", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("folder_path"); got != "/Shared" {
			t.Errorf("folder_path = %q", got)
		}
		w.Write([]byte(`{"recentFiles":[{"name":"Outline.docx",
			"path":"/Private/johndoe/Drafts/Outline.docx","size":307951,
			"entry_id":"cba6286d","group_id":"84792f80","uploaded_by":"John Doe",
			"num_versions":2,"last_modified":1542833068000,"last_accessed":1542945701862,
			"recommendation_type":"recent"}]}`))
	})

	files, _, err := client.Insights.RecentFiles(context.Background(), "/Shared")
	if err != nil {
		t.Fatalf("RecentFiles: %v", err)
	}
	if len(files) != 1 || files[0].Name != "Outline.docx" || files[0].LastAccessed != 1542945701862 {
		t.Errorf("files = %+v", files)
	}
}

func TestInsightsService_RecentFiles_forbidden(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/insights/files", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("folder_path") {
			t.Error("empty folder_path should be omitted")
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"statusCode":403,"errorMessage":"Insights not available"}`))
	})

	_, _, err := client.Insights.RecentFiles(context.Background(), "")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError = %+v", apiErr)
	}
}
