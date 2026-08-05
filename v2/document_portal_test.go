package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestDocumentPortalService_ListWorkspaces(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/document-portal/workspaces", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"count":1,"results":[{"id":"ae898c78","name":"Acme Enterprises",
			"type":"Client","status":"ACTIVE","taskCount":0,
			"creator":{"id":1,"username":"jsmith","email":"jsmith@example.com"}}]}`))
	})

	list, _, err := client.DocumentPortal.ListWorkspaces(context.Background())
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if list.Count != 1 || list.Results[0].Creator.Username != "jsmith" {
		t.Errorf("list = %+v", list)
	}
}

func TestDocumentPortalService_CreateWorkspace_fromTemplate(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/document-portal/workspaces", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		variables, _ := body["variables"].([]any)
		if body["workspaceParentFolderId"] != "4e6173b7" ||
			body["workspaceTemplateId"] != "0ee1f7ff" || len(variables) != 1 {
			t.Errorf("body = %v", body)
		}
		if _, ok := body["workspaceFolderId"]; ok {
			t.Error("unset workspaceFolderId should be omitted")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"e250227d"}`))
	})

	id, resp, err := client.DocumentPortal.CreateWorkspace(context.Background(), CreateWorkspaceRequest{
		Name:                    "Acme Enterprises",
		Status:                  "ACTIVE",
		Type:                    "Client",
		WorkspaceParentFolderID: "4e6173b7",
		WorkspaceTemplateID:     "0ee1f7ff",
		Variables:               []WorkspaceTemplateVariable{{Name: "Account Manager", Value: []int{123}}},
	})
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || id != "e250227d" {
		t.Errorf("id = %q, status = %d", id, resp.StatusCode)
	}
}

func TestDocumentPortalService_CreateWorkspace_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	base := CreateWorkspaceRequest{Name: "x", Status: "ACTIVE", Type: "Client"}

	neither := base
	if _, _, err := client.DocumentPortal.CreateWorkspace(ctx, neither); err == nil {
		t.Error("neither folder id should fail before sending")
	}

	both := base
	both.WorkspaceFolderID = "f1"
	both.WorkspaceParentFolderID = "f2"
	if _, _, err := client.DocumentPortal.CreateWorkspace(ctx, both); err == nil {
		t.Error("both folder ids should fail before sending")
	}

	templateWithoutParent := base
	templateWithoutParent.WorkspaceFolderID = "f1"
	templateWithoutParent.WorkspaceTemplateID = "t1"
	if _, _, err := client.DocumentPortal.CreateWorkspace(ctx, templateWithoutParent); err == nil {
		t.Error("template without parent folder should fail before sending")
	}
}

func TestDocumentPortalService_Settings(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/document-portal/settings", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"workspaceTypes":["Client","Vendor","Partner"]}`))
	})

	settings, _, err := client.DocumentPortal.Settings(context.Background())
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if len(settings.WorkspaceTypes) != 3 || settings.WorkspaceTypes[0] != "Client" {
		t.Errorf("settings = %+v", settings)
	}
}

func TestDocumentPortalService_ListWorkspaces_forbidden(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/document-portal/workspaces", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Missing Egnyte.documentportal scope","code":"FORBIDDEN"}`))
	})

	_, _, err := client.DocumentPortal.ListWorkspaces(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError = %+v", apiErr)
	}
}
