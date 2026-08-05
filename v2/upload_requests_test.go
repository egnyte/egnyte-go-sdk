package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestUploadRequestsService_ListTemplates(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/upload-requests/templates", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`[{"templateId":"74403afb","version":"1.0","name":"Account Onboarding",
			"status":"ACTIVE","assigneeInstructions":"Provide the following documents"}]`))
	})

	templates, _, err := client.UploadRequests.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates) != 1 || templates[0].TemplateID != "74403afb" || templates[0].Status != "ACTIVE" {
		t.Errorf("templates = %+v", templates)
	}
}

func TestUploadRequestsService_Create(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/upload-requests", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["name"] != "Account Onboarding" || body["folderId"] != "6e4465e9" {
			t.Errorf("body = %v", body)
		}
		assignees, _ := body["assignees"].([]any)
		if len(assignees) != 2 {
			t.Fatalf("assignees = %v", body["assignees"])
		}
		internal, _ := assignees[0].(map[string]any)
		external, _ := assignees[1].(map[string]any)
		if internal["userId"] != float64(47) || external["externalEmail"] != "bob@example.com" {
			t.Errorf("assignees = %v", assignees)
		}
		if _, ok := internal["externalEmail"]; ok {
			t.Error("internal assignee must not carry externalEmail")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"e250227d"}`))
	})

	id, resp, err := client.UploadRequests.Create(context.Background(), CreateUploadRequest{
		Name:                 "Account Onboarding",
		DueDate:              "2026-12-31T23:59:59.000Z",
		FolderID:             "6e4465e9",
		TemplateID:           "0af99f3d",
		Assignees:            []UploadRequestAssignee{InternalAssignee(47), ExternalAssignee("bob@example.com")},
		AssigneeInstructions: "Provide the following documents",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || id != "e250227d" {
		t.Errorf("id = %q, status = %d", id, resp.StatusCode)
	}
}

func TestUploadRequestsService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	valid := CreateUploadRequest{
		Name: "x", DueDate: "2026-12-31T23:59:59.000Z", FolderID: "f", TemplateID: "t",
		Assignees:            []UploadRequestAssignee{InternalAssignee(1)},
		AssigneeInstructions: "i",
	}

	missing := valid
	missing.TemplateID = ""
	if _, _, err := client.UploadRequests.Create(ctx, missing); err == nil {
		t.Error("missing templateId should fail before sending")
	}

	both := valid
	both.Assignees = []UploadRequestAssignee{{UserID: 1, ExternalEmail: "a@b.com"}}
	if _, _, err := client.UploadRequests.Create(ctx, both); err == nil {
		t.Error("assignee with both userId and externalEmail should fail before sending")
	}

	neither := valid
	neither.Assignees = []UploadRequestAssignee{{}}
	if _, _, err := client.UploadRequests.Create(ctx, neither); err == nil {
		t.Error("assignee with neither userId nor externalEmail should fail before sending")
	}
}

func TestUploadRequestsService_Create_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/upload-requests", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Template not found"}`))
	})

	_, _, err := client.UploadRequests.Create(context.Background(), CreateUploadRequest{
		Name: "x", DueDate: "d", FolderID: "f", TemplateID: "missing",
		Assignees:            []UploadRequestAssignee{InternalAssignee(1)},
		AssigneeInstructions: "i",
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Message != "Template not found" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
