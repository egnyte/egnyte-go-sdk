package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestSearchService_Search(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("query") != "cloud storage" || q.Get("folder") != "/Shared/Documents" ||
			q.Get("type") != "FILE" || q.Get("snippet_requested") != "false" ||
			q.Get("count") != "5" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"results":[{"name":"LocalCloudPress.doc",
			"path":"/Shared/Documents/Sales/LocalCloudPress.doc",
			"type":"application/msword","size":28672,
			"snippet":"Egnyte brings its storage cloud closer...",
			"entry_id":"2c8e1083","group_id":"61ed8373",
			"last_modified":"2020-01-14T22:19:29Z","uploaded_by":"David Pfeffer",
			"uploaded_by_username":"david","num_versions":1,"is_folder":false}],
			"total_count":20,"offset":0,"count":1}`))
	})

	results, _, err := client.Search.Search(context.Background(), "cloud storage", &SearchOptions{
		Folder:           "/Shared/Documents",
		Type:             "FILE",
		SnippetRequested: Bool(false),
		Count:            5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if results.TotalCount != 20 || len(results.Results) != 1 {
		t.Fatalf("results = %+v", results)
	}
	r0 := results.Results[0]
	if r0.Name != "LocalCloudPress.doc" || r0.Size != 28672 || r0.UploadedByUsername != "david" {
		t.Errorf("result = %+v", r0)
	}
}

func TestSearchService_Search_validation(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	ctx := context.Background()
	if _, _, err := client.Search.Search(ctx, "ab", nil); err == nil {
		t.Error("query under 3 chars should fail before sending")
	}
	// Bounds are counted in characters, not bytes, so multi-byte queries
	// are measured the way the API measures them.
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"2 ascii", "ab", true},
		{"3 ascii", "abc", false},
		{"100 ascii", strings.Repeat("a", 100), false},
		{"101 ascii", strings.Repeat("a", 101), true},
		{"2 multibyte", "ąę", true},
		{"3 multibyte", "ąęó", false},
		{"100 multibyte", strings.Repeat("ą", 100), false},
		{"101 multibyte", strings.Repeat("ą", 101), true},
	}
	for _, tt := range tests {
		_, _, err := client.Search.Search(ctx, tt.query, nil)
		if gotErr := err != nil; gotErr != tt.wantErr {
			t.Errorf("Search(%s): error = %v, wantErr = %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestSearchService_SearchV2(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		filters, _ := body["custom_metadata"].([]any)
		if len(filters) != 2 {
			t.Fatalf("custom_metadata = %v", body["custom_metadata"])
		}
		f0, _ := filters[0].(map[string]any)
		if f0["operator"] != "EQUALS" || f0["value"] != "active" {
			t.Errorf("filter[0] = %v", f0)
		}
		f1, _ := filters[1].(map[string]any)
		rng, _ := f1["range"].(map[string]any)
		if f1["operator"] != "BETWEEN" || rng["start"] != float64(1704085200000) {
			t.Errorf("filter[1] = %v", f1)
		}
		if _, ok := f1["value"]; ok {
			t.Error("unset value should be omitted for BETWEEN filter")
		}
		w.Write([]byte(`{"count":1,"offset":0,"total_count":1,"hasMore":false,
			"results":[{"name":"cat.txt","path":"/Shared/test/cat.txt","type":"text/plain",
			"entry_id":"b1be1ef9","score":1.5,
			"custom_properties":[{"namespace":"ns1","key":"status","value":"active"}]}]}`))
	})

	results, _, err := client.Search.SearchV2(context.Background(), SearchRequestV2{
		CustomMetadata: []MetadataFilter{
			{Namespace: "ns1", Key: "status", Operator: "EQUALS", Value: "active"},
			{Namespace: "ns1", Key: "review_date", Operator: "BETWEEN",
				Range: &MetadataRange{Start: 1704085200000, End: 1735621200000}},
		},
		Namespaces: []string{"ns1"},
	})
	if err != nil {
		t.Fatalf("SearchV2: %v", err)
	}
	if results.TotalCount != 1 || results.HasMore {
		t.Fatalf("results = %+v", results)
	}
	r0 := results.Results[0]
	if r0.Score != 1.5 || len(r0.CustomProperties) != 1 || r0.CustomProperties[0].Value != "active" {
		t.Errorf("result = %+v", r0)
	}
}

func TestSearchService_SearchV2_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Search.SearchV2(context.Background(), SearchRequestV2{}); err == nil {
		t.Error("empty v2 search should fail before sending")
	}
}

func TestSearchService_Search_forbidden(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/search", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Access denied"}`))
	})

	_, _, err := client.Search.Search(context.Background(), "secret plans", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError = %+v", apiErr)
	}
}
