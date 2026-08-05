package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestControlledDocsService_ImportDocument(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/controlled-docs/documents/import", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["docId"] != "SOP-1" || body["version"] != "1.0" ||
			body["entryId"] != "f450227d" || body["isDraft"] != false ||
			body["effectiveFromDate"] != "2023-01-22" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"f450227d"}`))
	})

	id, resp, err := client.ControlledDocs.ImportDocument(context.Background(), ImportControlledDocumentRequest{
		DocID:                 "SOP-1",
		Name:                  "Change Management SOP",
		Version:               "1.0",
		EntryID:               "f450227d",
		ResponsibleDepartment: "Clinical",
		EffectiveFromDate:     "2023-01-22",
		EffectiveToDate:       "2024-01-21",
		ApprovedOn:            "2022-12-17",
	})
	if err != nil {
		t.Fatalf("ImportDocument: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || id != "f450227d" {
		t.Errorf("id = %q, status = %d", id, resp.StatusCode)
	}
}

func TestControlledDocsService_ImportTrainingAssignment(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/controlled-docs/assignments/import", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["assigneeId"] != float64(11) || body["docId"] != "SOP-1" ||
			body["dueDate"] != "2023-02-22" || body["assignedById"] != float64(22) {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"e250227d"}`))
	})

	id, _, err := client.ControlledDocs.ImportTrainingAssignment(context.Background(), ImportTrainingAssignmentRequest{
		AssigneeID:    11,
		DocID:         "SOP-1",
		Version:       "1.0",
		AssignedDate:  "2023-01-22T14:00:00Z",
		DueDate:       "2023-02-22",
		CompletedDate: "2023-01-27T16:22:07Z",
		AssignedByID:  22,
	})
	if err != nil {
		t.Fatalf("ImportTrainingAssignment: %v", err)
	}
	if id != "e250227d" {
		t.Errorf("id = %q", id)
	}
}

func TestControlledDocsService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.ControlledDocs.ImportDocument(ctx, ImportControlledDocumentRequest{
		DocID: "SOP-1", Name: "x", Version: "1.0",
	}); err == nil {
		t.Error("missing entryId should fail before sending")
	}
	if _, _, err := client.ControlledDocs.ImportTrainingAssignment(ctx, ImportTrainingAssignmentRequest{
		AssigneeID: 11, DocID: "SOP-1", Version: "1.0",
	}); err == nil {
		t.Error("missing dueDate should fail before sending")
	}
}

func TestControlledDocsService_ImportDocument_conflict(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/controlled-docs/documents/import", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"msg":"Document version already exists for this document"}`))
	})

	_, _, err := client.ControlledDocs.ImportDocument(context.Background(), ImportControlledDocumentRequest{
		DocID: "SOP-1", Name: "x", Version: "1.0", EntryID: "e1",
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("APIError = %+v", apiErr)
	}
}
