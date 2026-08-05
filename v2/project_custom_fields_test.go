package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestProjectCustomFieldsService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/project-tag/custom-fields", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"keys":{
			"budget_code":{"name":"budget_code","displayName":"Budget Code","type":"string",
				"helpText":"Enter the project budget code","priority":100},
			"project_category":{"name":"project_category","displayName":"Project Category",
				"type":"enum","data":["Infrastructure","Development"],"priority":150}}}`))
	})

	fields, _, err := client.ProjectCustomFields.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(fields) != 2 || fields["budget_code"].Type != "string" {
		t.Errorf("fields = %+v", fields)
	}
	if cat := fields["project_category"]; len(cat.Data) != 2 || cat.Priority != 150 {
		t.Errorf("category = %+v", cat)
	}
}

func TestProjectCustomFieldsService_Create(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/project-tag/custom-fields", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		data, _ := body["data"].([]any)
		if body["displayName"] != "Project Tags" || body["type"] != "labels" ||
			len(data) != 3 || body["priority"] != float64(200) {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusCreated)
	})

	_, err := client.ProjectCustomFields.Create(context.Background(), CreateProjectCustomFieldRequest{
		DisplayName: "Project Tags",
		Type:        "labels",
		Data:        []string{"infrastructure", "customer-facing", "internal"},
		HelpText:    "Add relevant tags",
		Priority:    Int(200),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestProjectCustomFieldsService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, err := client.ProjectCustomFields.Create(ctx, CreateProjectCustomFieldRequest{
		DisplayName: "x", Type: "enum", HelpText: "h",
	}); err == nil {
		t.Error("enum without data should fail before sending")
	}
	if _, err := client.ProjectCustomFields.Create(ctx, CreateProjectCustomFieldRequest{
		DisplayName: "x", Type: "string", HelpText: "h", Data: []string{"a"},
	}); err == nil {
		t.Error("primitive type with data should fail before sending")
	}
	if _, err := client.ProjectCustomFields.Create(ctx, CreateProjectCustomFieldRequest{
		DisplayName: "x", Type: "string",
	}); err == nil {
		t.Error("missing helpText should fail before sending")
	}
}

func TestProjectCustomFieldsService_Update(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/project-tag/custom-fields/project_tags", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		data, _ := body["data"].([]any)
		if body["displayName"] != "Project Tags (Updated)" || len(data) != 1 {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.ProjectCustomFields.Update(context.Background(), "project_tags",
		UpdateProjectCustomFieldRequest{
			DisplayName: "Project Tags (Updated)",
			Data:        []string{"architecture"},
		})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestProjectCustomFieldsService_DeleteAndDeleteData(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/project-tag/custom-fields/project_tags", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
		testHeader(t, r, "X-Egnyte-Force-Delete", "Yes")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/pubapi/v1/properties/project-tag/custom-fields/project_tags/data", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		if got := r.Header.Get("X-Egnyte-Force-Delete"); got != "" {
			t.Errorf("force header should be absent, got %q", got)
		}
		var values []string
		if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(values) != 2 || values[0] != "infrastructure" {
			t.Errorf("values = %v", values)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.Background()
	if _, err := client.ProjectCustomFields.Delete(ctx, "project_tags", true); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := client.ProjectCustomFields.DeleteData(ctx, "project_tags",
		[]string{"infrastructure", "customer-facing"}, false); err != nil {
		t.Fatalf("DeleteData: %v", err)
	}
}

func TestProjectCustomFieldsService_Create_featureDisabled(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/properties/project-tag/custom-fields", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errorMessage":"Feature is disabled for this domain","errorCode":"FEATURE_DISABLED"}`))
	})

	_, err := client.ProjectCustomFields.Create(context.Background(), CreateProjectCustomFieldRequest{
		DisplayName: "x", Type: "string", HelpText: "h",
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError = %+v", apiErr)
	}
}
