package egnyte

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ProjectFoldersService manages folders designated as projects, which
// carry additional metadata (status, dates, location, custom fields) and
// can create dynamic groups.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/project-folder-api
// OAuth scope: Egnyte.projectfolders. Project functionality must also be
// enabled for the domain.
type ProjectFoldersService service

// ProjectLocation is the physical location of a project.
type ProjectLocation struct {
	StreetAddress1 string `json:"streetAddress1,omitempty"`
	StreetAddress2 string `json:"streetAddress2,omitempty"`
	City           string `json:"city,omitempty"`
	State          string `json:"state,omitempty"`
	Country        string `json:"country,omitempty"`
	PostalCode     string `json:"postalCode,omitempty"`
}

// Project is a project folder's metadata.
type Project struct {
	ID           string `json:"id"`
	RootFolderID string `json:"rootFolderId"`
	Name         string `json:"name"`
	// ProjectID is the custom project identifier (e.g. "ABC-123").
	ProjectID    string           `json:"projectId"`
	CustomerName string           `json:"customerName"`
	Description  string           `json:"description"`
	Location     *ProjectLocation `json:"location"`
	// Status is pending, in-progress, completed, on-hold, canceled, or
	// a custom status string.
	Status         string `json:"status"`
	StartDate      string `json:"startDate"`
	CompletionDate string `json:"completionDate"`
	CreatedBy      int    `json:"createdBy"`
	LastUpdatedBy  int    `json:"lastUpdatedBy"`
	CreationTime   string `json:"creationTime"`
	LastModified   string `json:"lastModifiedTime"`
	// CustomFieldsWithValues is present only when the custom fields
	// feature is enabled.
	CustomFieldsWithValues map[string]string `json:"customFieldsWithValues"`
}

// ProjectList is a page of projects.
type ProjectList struct {
	ItemsPerPage int       `json:"itemsPerPage"`
	TotalResults int       `json:"totalResults"`
	StartIndex   int       `json:"startIndex"`
	Resources    []Project `json:"resources"`
}

// ProjectListOptions paginate project listings.
type ProjectListOptions struct {
	// StartIndex is the 1-based index of the first result.
	StartIndex int
	// Count is the page size (max 100).
	Count int
}

// MarkFolderAsProjectRequest converts an existing folder into a project.
type MarkFolderAsProjectRequest struct {
	RootFolderID   string `json:"rootFolderId"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Status         string `json:"status"`
	StartDate      string `json:"startDate,omitempty"`
	CompletionDate string `json:"completionDate,omitempty"`
}

// CreateProjectFromTemplateRequest creates a project from a project
// folder template. Requires the project folder templates feature.
type CreateProjectFromTemplateRequest struct {
	ParentFolderID   string `json:"parentFolderId"`
	TemplateFolderID string `json:"templateFolderId"`
	FolderName       string `json:"folderName"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	// ProjectID is required if used in dynamic group names.
	ProjectID              string            `json:"projectId,omitempty"`
	CustomerName           string            `json:"customerName,omitempty"`
	Location               *ProjectLocation  `json:"location,omitempty"`
	Status                 string            `json:"status"`
	StartDate              string            `json:"startDate,omitempty"`
	CompletionDate         string            `json:"completionDate,omitempty"`
	CustomFieldsWithValues map[string]string `json:"customFieldsWithValues,omitempty"`
}

// CreatedProjectGroup is a dynamic group created from a template.
type CreatedProjectGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpdateProjectRequest partially updates a project; only provided fields
// change.
type UpdateProjectRequest struct {
	Name           string           `json:"name,omitempty"`
	Description    string           `json:"description,omitempty"`
	ProjectID      string           `json:"projectId,omitempty"`
	CustomerName   string           `json:"customerName,omitempty"`
	Location       *ProjectLocation `json:"location,omitempty"`
	Status         string           `json:"status,omitempty"`
	StartDate      string           `json:"startDate,omitempty"`
	CompletionDate string           `json:"completionDate,omitempty"`
	// CustomFieldsWithValues adds or updates the given keys only.
	CustomFieldsWithValues map[string]string `json:"customFieldsWithValues,omitempty"`
	// DeleteCustomFields removes keys, processed after
	// CustomFieldsWithValues.
	DeleteCustomFields []string `json:"deleteCustomFields,omitempty"`
}

// ProjectCleanupRequest configures cleanup of a completed or canceled
// project. The operation runs asynchronously (202).
type ProjectCleanupRequest struct {
	// DeleteLinks deletes all active links in the project.
	DeleteLinks bool `json:"deleteLinks"`
	// UsersToDelete / UsersToDisable list user ids to remove or disable.
	UsersToDelete  []int `json:"usersToDelete,omitempty"`
	UsersToDisable []int `json:"usersToDisable,omitempty"`
}

// List returns all project folders in the domain.
//
// GET /pubapi/v2/project-folders
func (s *ProjectFoldersService) List(ctx context.Context, opts *ProjectListOptions) (*ProjectList, *Response, error) {
	urlPath := "v2/project-folders"
	if opts != nil {
		q := url.Values{}
		if opts.StartIndex > 0 {
			q.Set("startIndex", strconv.Itoa(opts.StartIndex))
		}
		if opts.Count > 0 {
			q.Set("count", strconv.Itoa(opts.Count))
		}
		if enc := q.Encode(); enc != "" {
			urlPath += "?" + enc
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	// The spec documents a paginated object, but the live API returns a
	// bare JSON array — accept both shapes.
	var raw json.RawMessage
	resp, err := s.client.Do(req, &raw)
	if err != nil {
		return nil, resp, err
	}
	list := new(ProjectList)
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 {
		if trimmed[0] == '[' {
			if err := json.Unmarshal(trimmed, &list.Resources); err != nil {
				return nil, resp, err
			}
			list.TotalResults = len(list.Resources)
		} else if err := json.Unmarshal(trimmed, list); err != nil {
			return nil, resp, err
		}
	}
	return list, resp, nil
}

// MarkFolderAsProject designates an existing folder as a project and
// returns the new project metadata.
//
// POST /pubapi/v1/project-folders
func (s *ProjectFoldersService) MarkFolderAsProject(ctx context.Context, project MarkFolderAsProjectRequest) (*Project, *Response, error) {
	if project.RootFolderID == "" || project.Name == "" || project.Status == "" {
		return nil, nil, fmt.Errorf("egnyte: project rootFolderId, name and status are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/project-folders", project)
	if err != nil {
		return nil, nil, err
	}
	created := new(Project)
	resp, err := s.client.Do(req, created)
	if err != nil {
		return nil, resp, err
	}
	return created, resp, nil
}

// CreateFromTemplate creates a new project from a template, including
// folder structure and permissions. Returns any dynamic groups created.
//
// POST /pubapi/v2/project-folders
func (s *ProjectFoldersService) CreateFromTemplate(ctx context.Context, project CreateProjectFromTemplateRequest) ([]CreatedProjectGroup, *Response, error) {
	switch {
	case project.ParentFolderID == "", project.TemplateFolderID == "",
		project.FolderName == "", project.Name == "", project.Status == "":
		return nil, nil, fmt.Errorf("egnyte: project parentFolderId, templateFolderId, folderName, name and status are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/project-folders", project)
	if err != nil {
		return nil, nil, err
	}
	var body struct {
		GroupsCreated []CreatedProjectGroup `json:"groupsCreated"`
	}
	resp, err := s.client.Do(req, &body)
	if err != nil {
		return nil, resp, err
	}
	return body.GroupsCreated, resp, nil
}

// Get returns the details of the project with the given project id.
//
// GET /pubapi/v2/project-folders/{projectId}
func (s *ProjectFoldersService) Get(ctx context.Context, projectID string) (*Project, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v2/project-folders/"+url.PathEscape(projectID), nil)
	if err != nil {
		return nil, nil, err
	}
	project := new(Project)
	resp, err := s.client.Do(req, project)
	if err != nil {
		return nil, resp, err
	}
	return project, resp, nil
}

// Update modifies project metadata and returns the updated project.
//
// PATCH /pubapi/v2/project-folders/{projectId}
func (s *ProjectFoldersService) Update(ctx context.Context, projectID string, update UpdateProjectRequest) (*Project, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPatch, "v2/project-folders/"+url.PathEscape(projectID), update)
	if err != nil {
		return nil, nil, err
	}
	project := new(Project)
	resp, err := s.client.Do(req, project)
	if err != nil {
		return nil, resp, err
	}
	return project, resp, nil
}

// Delete removes the project designation and metadata from a folder.
// The underlying folder is not deleted.
//
// DELETE /pubapi/v2/project-folders/{projectId}
func (s *ProjectFoldersService) Delete(ctx context.Context, projectID string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v2/project-folders/"+url.PathEscape(projectID), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// FindByRootFolder returns the projects whose root folder has the given
// folder id (typically one).
//
// POST /pubapi/v2/project-folders/search
func (s *ProjectFoldersService) FindByRootFolder(ctx context.Context, rootFolderID string) ([]Project, *Response, error) {
	if rootFolderID == "" {
		return nil, nil, fmt.Errorf("egnyte: rootFolderId is required")
	}
	body := struct {
		RootFolderID string `json:"rootFolderId"`
	}{rootFolderID}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/project-folders/search", body)
	if err != nil {
		return nil, nil, err
	}
	var projects []Project
	resp, err := s.client.Do(req, &projects)
	if err != nil {
		return nil, resp, err
	}
	return projects, resp, nil
}

// Cleanup starts an asynchronous cleanup of a completed or canceled
// project (deleting links, removing or disabling users). A 409 means a
// previous cleanup is still running.
//
// POST /pubapi/v1/project-folders/{projectId}/cleanup
func (s *ProjectFoldersService) Cleanup(ctx context.Context, projectID string, cleanup ProjectCleanupRequest) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/project-folders/"+url.PathEscape(projectID)+"/cleanup", cleanup)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
