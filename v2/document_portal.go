package egnyte

import (
	"context"
	"fmt"
	"net/http"
)

// DocumentPortalService manages Document Portal workspaces. Requires the
// Egnyte.documentportal scope.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/document-portal-api
// OAuth scope: Egnyte.documentportal.
type DocumentPortalService service

// WorkspaceCreator identifies the user who created a workspace.
type WorkspaceCreator struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// Workspace is one Document Portal workspace.
type Workspace struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`   // e.g. Client, Vendor
	Status    string            `json:"status"` // ACTIVE or INACTIVE
	TaskCount int               `json:"taskCount"`
	Creator   *WorkspaceCreator `json:"creator"`
}

// WorkspaceList is the workspace listing response.
type WorkspaceList struct {
	Count   int         `json:"count"`
	Results []Workspace `json:"results"`
}

// WorkspaceTemplateVariable assigns user ids to a permission template
// variable.
type WorkspaceTemplateVariable struct {
	Name  string `json:"name"`
	Value []int  `json:"value"` // user ids
}

// CreateWorkspaceRequest creates a workspace in exactly one mode:
//   - from an existing folder: set WorkspaceFolderID
//   - blank: set WorkspaceParentFolderID only
//   - from a template: set WorkspaceParentFolderID and
//     WorkspaceTemplateID (optionally Variables)
type CreateWorkspaceRequest struct {
	Name string `json:"name"`
	// Status is ACTIVE or INACTIVE.
	Status string `json:"status"`
	// Type must match one of the types from Settings.
	Type                    string                      `json:"type"`
	WorkspaceFolderID       string                      `json:"workspaceFolderId,omitempty"`
	WorkspaceParentFolderID string                      `json:"workspaceParentFolderId,omitempty"`
	WorkspaceTemplateID     string                      `json:"workspaceTemplateId,omitempty"`
	Variables               []WorkspaceTemplateVariable `json:"variables,omitempty"`
}

// DocumentPortalSettings is the domain's Document Portal configuration.
type DocumentPortalSettings struct {
	WorkspaceTypes []string `json:"workspaceTypes"`
}

// ListWorkspaces returns all workspaces in the domain.
//
// GET /pubapi/v1/document-portal/workspaces
func (s *DocumentPortalService) ListWorkspaces(ctx context.Context) (*WorkspaceList, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/document-portal/workspaces", nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(WorkspaceList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// CreateWorkspace creates a workspace and returns its id.
//
// POST /pubapi/v1/document-portal/workspaces
func (s *DocumentPortalService) CreateWorkspace(ctx context.Context, workspace CreateWorkspaceRequest) (string, *Response, error) {
	if workspace.Name == "" || workspace.Status == "" || workspace.Type == "" {
		return "", nil, fmt.Errorf("egnyte: workspace name, status and type are required")
	}
	if (workspace.WorkspaceFolderID != "") == (workspace.WorkspaceParentFolderID != "") {
		return "", nil, fmt.Errorf("egnyte: set exactly one of workspaceFolderId or workspaceParentFolderId")
	}
	if workspace.WorkspaceTemplateID != "" && workspace.WorkspaceParentFolderID == "" {
		return "", nil, fmt.Errorf("egnyte: workspaceTemplateId requires workspaceParentFolderId")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/document-portal/workspaces", workspace)
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

// Settings returns the Document Portal configuration, including the
// available workspace types.
//
// GET /pubapi/v1/document-portal/settings
func (s *DocumentPortalService) Settings(ctx context.Context) (*DocumentPortalSettings, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/document-portal/settings", nil)
	if err != nil {
		return nil, nil, err
	}
	settings := new(DocumentPortalSettings)
	resp, err := s.client.Do(req, settings)
	if err != nil {
		return nil, resp, err
	}
	return settings, resp, nil
}
