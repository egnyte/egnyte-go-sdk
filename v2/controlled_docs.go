package egnyte

import (
	"context"
	"fmt"
	"net/http"
)

// ControlledDocsService imports documents and training assignments into
// the Controlled Document Management app. Requires an Admin or Category
// Manager role.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/controlled-document-management-api
// OAuth scope: Egnyte.controlleddocs.
type ControlledDocsService service

// ImportControlledDocumentRequest imports a document or a new version of
// an existing document. When IsDraft is true the version must be below
// 1.0 and the date fields must be empty; when false, EffectiveFromDate
// and ApprovedOn are required.
type ImportControlledDocumentRequest struct {
	// DocID is the identifier within Controlled Document Management,
	// e.g. "SOP-1".
	DocID   string `json:"docId"`
	Name    string `json:"name"`
	IsDraft bool   `json:"isDraft"`
	Version string `json:"version"`
	// EntryID is the file version to import.
	EntryID               string `json:"entryId"`
	ResponsibleDepartment string `json:"responsibleDepartment,omitempty"`
	// EffectiveFromDate / EffectiveToDate / ApprovedOn are YYYY-MM-DD.
	EffectiveFromDate string `json:"effectiveFromDate,omitempty"`
	EffectiveToDate   string `json:"effectiveToDate,omitempty"`
	ApprovedOn        string `json:"approvedOn,omitempty"`
}

// ImportTrainingAssignmentRequest imports a training assignment record.
// The status is derived by the server from DueDate, CompletedDate and
// CanceledDate.
type ImportTrainingAssignmentRequest struct {
	AssigneeID int    `json:"assigneeId"`
	DocID      string `json:"docId"`
	Version    string `json:"version"`
	// AssignedDate (ISO-8601) defaults to now when empty.
	AssignedDate string `json:"assignedDate,omitempty"`
	// DueDate is YYYY-MM-DD.
	DueDate string `json:"dueDate"`
	// CompletedDate / CanceledDate (ISO-8601) mark finished assignments.
	CompletedDate string `json:"completedDate,omitempty"`
	CanceledDate  string `json:"canceledDate,omitempty"`
	// AssignedByID defaults to the requesting user when zero.
	AssignedByID int `json:"assignedById,omitempty"`
}

// ImportDocument imports a document and returns the created resource id.
//
// POST /pubapi/v1/controlled-docs/documents/import
func (s *ControlledDocsService) ImportDocument(ctx context.Context, document ImportControlledDocumentRequest) (string, *Response, error) {
	if document.DocID == "" || document.Name == "" || document.Version == "" || document.EntryID == "" {
		return "", nil, fmt.Errorf("egnyte: docId, name, version and entryId are required")
	}
	return s.importResource(ctx, "v1/controlled-docs/documents/import", document)
}

// ImportTrainingAssignment imports a training assignment and returns the
// created resource id.
//
// POST /pubapi/v1/controlled-docs/assignments/import
func (s *ControlledDocsService) ImportTrainingAssignment(ctx context.Context, assignment ImportTrainingAssignmentRequest) (string, *Response, error) {
	if assignment.AssigneeID == 0 || assignment.DocID == "" || assignment.Version == "" || assignment.DueDate == "" {
		return "", nil, fmt.Errorf("egnyte: assigneeId, docId, version and dueDate are required")
	}
	return s.importResource(ctx, "v1/controlled-docs/assignments/import", assignment)
}

func (s *ControlledDocsService) importResource(ctx context.Context, urlPath string, body any) (string, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, urlPath, body)
	if err != nil {
		return "", nil, err
	}
	var created struct {
		ID string `json:"id"`
	}
	resp, err := s.client.Do(req, &created)
	if err != nil {
		return "", resp, err
	}
	return created.ID, resp, nil
}
