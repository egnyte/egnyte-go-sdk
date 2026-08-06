package egnyte

import (
	"context"
	"fmt"
	"net/http"
)

// UploadRequestsService sends document collection requests to internal
// or external users. Requires the Egnyte.uploadrequests scope.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/upload-requests-api
// OAuth scope: Egnyte.uploadrequests.
type UploadRequestsService service

// UploadRequestTemplate is a reusable upload request definition.
type UploadRequestTemplate struct {
	TemplateID           string `json:"templateId"`
	Version              string `json:"version"`
	Name                 string `json:"name"`
	Status               string `json:"status"` // ACTIVE
	AssigneeInstructions string `json:"assigneeInstructions"`
}

// UploadRequestAssignee is one recipient of an upload request. Exactly
// one of UserID (internal user) or ExternalEmail (external user) must be
// set — use InternalAssignee or ExternalAssignee to construct.
type UploadRequestAssignee struct {
	UserID        int    `json:"userId,omitempty"`
	ExternalEmail string `json:"externalEmail,omitempty"`
}

// InternalAssignee addresses an upload request to an internal user.
func InternalAssignee(userID int) UploadRequestAssignee {
	return UploadRequestAssignee{UserID: userID}
}

// ExternalAssignee addresses an upload request to an external email.
func ExternalAssignee(email string) UploadRequestAssignee {
	return UploadRequestAssignee{ExternalEmail: email}
}

// CreateUploadRequest is the body for creating an upload request. All
// fields are required.
type CreateUploadRequest struct {
	Name string `json:"name"`
	// DueDate is an ISO 8601 timestamp.
	DueDate string `json:"dueDate"`
	// FolderID is where uploaded files are stored.
	FolderID string `json:"folderId"`
	// TemplateID comes from ListTemplates.
	TemplateID           string                  `json:"templateId"`
	Assignees            []UploadRequestAssignee `json:"assignees"`
	AssigneeInstructions string                  `json:"assigneeInstructions"`
}

// ListTemplates returns the available upload request templates.
//
// GET /pubapi/v1/upload-requests/templates
func (s *UploadRequestsService) ListTemplates(ctx context.Context) ([]UploadRequestTemplate, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/upload-requests/templates", nil)
	if err != nil {
		return nil, nil, err
	}
	var templates []UploadRequestTemplate
	resp, err := s.client.Do(req, &templates)
	if err != nil {
		return nil, resp, err
	}
	return templates, resp, nil
}

// Create sends a new upload request to the given assignees and returns
// its id.
//
// POST /pubapi/v1/upload-requests
func (s *UploadRequestsService) Create(ctx context.Context, request CreateUploadRequest) (string, *Response, error) {
	switch {
	case request.Name == "", request.DueDate == "", request.FolderID == "",
		request.TemplateID == "", len(request.Assignees) == 0, request.AssigneeInstructions == "":
		return "", nil, fmt.Errorf("egnyte: upload request name, dueDate, folderId, templateId, assignees and assigneeInstructions are required")
	}
	for _, a := range request.Assignees {
		if (a.UserID == 0) == (a.ExternalEmail == "") {
			return "", nil, fmt.Errorf("egnyte: each assignee requires exactly one of userId or externalEmail")
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/upload-requests", request)
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
