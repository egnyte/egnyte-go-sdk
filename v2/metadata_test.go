package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestMetadataService_ListNamespaces(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("includeFolderAssociations"); got != "true" {
			t.Errorf("includeFolderAssociations = %q", got)
		}
		w.Write([]byte(`[{"name":"ns1","displayName":"My namespace","scope":"public",
			"inheritable":false,"metadataScopeType":"GLOBAL",
			"keys":{"string-key":{"type":"string","displayName":"string-key","priority":5}}}]`))
	})

	namespaces, _, err := client.Metadata.ListNamespaces(context.Background(), true)
	if err != nil {
		t.Fatalf("ListNamespaces: %v", err)
	}
	if len(namespaces) != 1 || namespaces[0].Name != "ns1" {
		t.Fatalf("namespaces = %+v", namespaces)
	}
	if key, ok := namespaces[0].Keys["string-key"]; !ok || key.Type != "string" || key.Priority != 5 {
		t.Errorf("keys = %+v", namespaces[0].Keys)
	}
}

func TestMetadataService_CreateNamespace(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["name"] != "namespace1" || body["scope"] != "public" {
			t.Errorf("body = %v", body)
		}
		keys, _ := body["keys"].(map[string]any)
		enumKey, _ := keys["enum-key"].(map[string]any)
		if enumKey["type"] != "enum" {
			t.Errorf("keys = %v", keys)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Metadata.CreateNamespace(context.Background(), CreateNamespaceRequest{
		Name:  "namespace1",
		Scope: "public",
		Keys: map[string]MetadataKey{
			"int-key":  {Type: "integer", Priority: 5, DisplayName: "My integer key"},
			"enum-key": {Type: "enum", Data: []string{"red", "green", "blue"}, Priority: 3},
		},
	})
	if err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
}

func TestMetadataService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, err := client.Metadata.CreateNamespace(ctx, CreateNamespaceRequest{Name: "x", Scope: "public"}); err == nil {
		t.Error("namespace without keys should fail before sending")
	}
	if _, err := client.Metadata.CreateKey(ctx, "ns", CreateMetadataKeyRequest{Key: "k"}); err == nil {
		t.Error("key without type should fail before sending")
	}
	if _, _, err := client.Metadata.UpdateNamespace(ctx, "ns", UpdateNamespaceRequest{
		FolderAssociation:       &NamespaceFolderAssociationUpdate{AssociateFolderIDs: []string{"f1"}},
		DocumentTypeAssociation: &NamespaceDocumentTypeAssociationUpdate{AssociateDocumentTypes: []string{"PDF"}},
	}); err == nil {
		t.Error("combined folder and document type association should fail before sending")
	}
	if _, _, err := client.Metadata.Search(ctx, MetadataSearchRequest{Type: "ALL"}); err == nil {
		t.Error("metadata search without criteria should fail before sending")
	}
}

func TestMetadataService_GetAndUpdateNamespace(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace/ns1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"name":"ns1","scope":"public","keys":{"k":{"type":"string"}}}`))
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			priorities, _ := body["priorities"].(map[string]any)
			if body["displayName"] != "Updated" || priorities["k"] != float64(10) {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(`{"name":"ns1","displayName":"Updated","scope":"public"}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	ns, _, err := client.Metadata.GetNamespace(ctx, "ns1")
	if err != nil {
		t.Fatalf("GetNamespace: %v", err)
	}
	if ns.Name != "ns1" || ns.Keys["k"].Type != "string" {
		t.Errorf("namespace = %+v", ns)
	}

	updated, _, err := client.Metadata.UpdateNamespace(ctx, "ns1", UpdateNamespaceRequest{
		DisplayName: "Updated",
		Priorities:  map[string]int{"k": 10},
	})
	if err != nil {
		t.Fatalf("UpdateNamespace: %v", err)
	}
	if updated.DisplayName != "Updated" {
		t.Errorf("updated = %+v", updated)
	}
}

func TestMetadataService_DeleteNamespace_force(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace/ns1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
		testHeader(t, r, "X-Egnyte-Force-Delete", "Yes")
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Metadata.DeleteNamespace(context.Background(), "ns1", true); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}
}

func TestMetadataService_KeyLifecycle(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace/ns1/keys", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "new-enum-key" || body["type"] != "enum" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/pubapi/v1/properties/namespace/ns1/keys/new-enum-key", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["priority"] != float64(10) {
				t.Errorf("body = %v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			if got := r.Header.Get("X-Egnyte-Force-Delete"); got != "" {
				t.Errorf("force header should be absent, got %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	if _, err := client.Metadata.CreateKey(ctx, "ns1", CreateMetadataKeyRequest{
		Key: "new-enum-key", Type: "enum", Data: []string{"red", "green"},
	}); err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	if _, err := client.Metadata.UpdateKey(ctx, "ns1", "new-enum-key", UpdateMetadataKeyRequest{
		Priority: Int(10),
	}); err != nil {
		t.Fatalf("UpdateKey: %v", err)
	}
	if _, err := client.Metadata.DeleteKey(ctx, "ns1", "new-enum-key", false); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
}

func TestMetadataService_DeleteKeyData(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace/ns1/keys/tags/data", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		testHeader(t, r, "X-Egnyte-Force-Delete", "Yes")
		var values []string
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(values) != 2 || values[0] != "urgent" {
			t.Errorf("values = %v", values)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Metadata.DeleteKeyData(context.Background(), "ns1", "tags",
		[]string{"urgent", "review"}, true); err != nil {
		t.Fatalf("DeleteKeyData: %v", err)
	}
}

func TestMetadataService_DeleteKeysData(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace/ns1/keys/data", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		var body struct {
			KeyDataDeletions []MetadataKeyDataDeletion `json:"keyDataDeletions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.KeyDataDeletions) != 2 || body.KeyDataDeletions[0].Key != "status" {
			t.Errorf("deletions = %+v", body.KeyDataDeletions)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Metadata.DeleteKeysData(context.Background(), "ns1", []MetadataKeyDataDeletion{
		{Key: "status", Data: []string{"completed"}},
		{Key: "priority", Data: []string{"low"}},
	}, false); err != nil {
		t.Fatalf("DeleteKeysData: %v", err)
	}
}

func TestMetadataService_FileAndFolderValues(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/ids/file/gid1/properties/ns1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["string-key"] != "abc" {
				t.Errorf("body = %v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			w.Write([]byte(`{"results":[{"ns1":{"string-key":"abc","int-key":10}}]}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	mux.HandleFunc("/pubapi/v1/fs/ids/folder/fid1/properties/ns1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"results":[{"ns1":{"project-code":"X123"}}]}`))
	})

	ctx := context.Background()
	if _, err := client.Metadata.SetFileMetadata(ctx, "gid1", "ns1", map[string]any{"string-key": "abc"}); err != nil {
		t.Fatalf("SetFileMetadata: %v", err)
	}
	values, _, err := client.Metadata.GetFileMetadata(ctx, "gid1", "ns1")
	if err != nil {
		t.Fatalf("GetFileMetadata: %v", err)
	}
	if len(values.Results) != 1 || values.Results[0]["ns1"]["string-key"] != "abc" {
		t.Errorf("values = %+v", values)
	}

	folderValues, _, err := client.Metadata.GetFolderMetadata(ctx, "fid1", "ns1")
	if err != nil {
		t.Fatalf("GetFolderMetadata: %v", err)
	}
	if folderValues.Results[0]["ns1"]["project-code"] != "X123" {
		t.Errorf("folder values = %+v", folderValues)
	}
}

func TestMetadataService_Search(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		kwv, _ := body["key_with_value"].([]any)
		if body["type"] != "ALL" || len(kwv) != 1 {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"results":[{"name":"tagged.txt","path":"/Shared/tagged.txt","is_folder":false}],
			"total_count":1,"offset":0,"count":1}`))
	})

	results, _, err := client.Metadata.Search(context.Background(), MetadataSearchRequest{
		Type:         "ALL",
		KeyWithValue: []MetadataKeyValueRef{{Namespace: "ns1", Key: "string-key", Value: "abc"}},
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if results.TotalCount != 1 || results.Results[0].Name != "tagged.txt" {
		t.Errorf("results = %+v", results)
	}
}

func TestMetadataService_CreateNamespace_conflict(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/namespace", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"statusCode":409,"errorMessage":"Namespace already exists"}`))
	})

	_, err := client.Metadata.CreateNamespace(context.Background(), CreateNamespaceRequest{
		Name: "ns1", Scope: "public", Keys: map[string]MetadataKey{"k": {Type: "string"}},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("APIError = %+v", apiErr)
	}
}
