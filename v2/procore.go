package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ProcoreService manages syncs between Egnyte folders and Procore
// projects. Requires the Egnyte.integrations scope (sync creation also
// needs Egnyte.launchwebsession), and Egnyte's DMSA key must be verified
// once per company via the Egnyte WebUI.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/third-party-integrations#egnyte-for-procore-integration
// OAuth scope: Egnyte.integrations; creating a sync additionally requires
// Egnyte.launchwebsession.
type ProcoreService service

// ProcoreSkippedFile is a file the sync skipped due to an error.
type ProcoreSkippedFile struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Source    string `json:"source"`
	ErrorCode string `json:"errorCode"`
	Version   string `json:"version"`
	Ctime     int64  `json:"ctime"`
	Checksum  string `json:"checksum"`
}

// ProcoreSyncedProject is one Egnyte↔Procore sync as listed.
type ProcoreSyncedProject struct {
	ID                 string   `json:"id"`
	JobID              string   `json:"jobId"`
	Folder             string   `json:"folder"` // Egnyte folder path
	ProcoreFolderPaths []string `json:"procoreFolderPaths"`
	Project            string   `json:"project"` // Procore project name
	ProcoreProjectID   int64    `json:"procoreProjectId"`
	ServiceAccountName string   `json:"serviceAccountName"`
	SyncType           string   `json:"syncType"` // "2-way sync" or "no sync"
	LastSuccessTime    string   `json:"lastSuccessTime"`
	IsHealthy          bool     `json:"isHealthy"`
	// IsHealthyNote describes the failure when IsHealthy is false.
	IsHealthyNote string               `json:"isHealthyNote"`
	Skipped       []ProcoreSkippedFile `json:"skipped"`
}

// ProcoreSync is a sync as returned by create and update calls.
type ProcoreSync struct {
	ID                      string   `json:"id"`
	JobID                   string   `json:"jobId"`
	EgnyteDomain            string   `json:"egnyteDomain"`
	EgnyteFolderPath        string   `json:"egnyteFolderPath"`
	ProcoreFolderPaths      []string `json:"procoreFolderPaths"`
	ProcoreServiceAccountID string   `json:"procoreServiceAccountId"`
	ProcoreProjectID        int64    `json:"procoreProjectId"`
	ProcoreProjectName      string   `json:"procoreProjectName"`
	SyncType                string   `json:"syncType"`
	CompanyID               int64    `json:"companyId"`
}

// ProcoreProject is a Procore project available for syncing.
type ProcoreProject struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	IsSynced  bool   `json:"isSynced"`
	CompanyID int64  `json:"companyId"`
}

// ListSyncedProjects returns the domain's Egnyte↔Procore syncs.
//
// GET /pubapi/v1/procore/sync
func (s *ProcoreService) ListSyncedProjects(ctx context.Context) ([]ProcoreSyncedProject, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/procore/sync", nil)
	if err != nil {
		return nil, nil, err
	}
	var projects []ProcoreSyncedProject
	resp, err := s.client.Do(req, &projects)
	if err != nil {
		return nil, resp, err
	}
	return projects, resp, nil
}

// CreateSync connects an Egnyte folder to a Procore project.
// procoreFolderPaths optionally limits the sync to specific Procore
// folders; empty syncs the entire project.
//
// POST /pubapi/v1/procore/sync
func (s *ProcoreService) CreateSync(ctx context.Context, folderPath, companyID, procoreProjectID string, procoreFolderPaths []string) (*ProcoreSync, *Response, error) {
	if folderPath == "" || companyID == "" || procoreProjectID == "" {
		return nil, nil, fmt.Errorf("egnyte: folderPath, companyId and procoreProjectId are required")
	}
	form := url.Values{}
	form.Set("folderPath", folderPath)
	form.Set("companyId", companyID)
	form.Set("procoreProjectId", procoreProjectID)
	if len(procoreFolderPaths) > 0 {
		form.Set("procoreFolderPaths", strings.Join(procoreFolderPaths, ","))
	}
	req, err := s.client.NewFormRequest(ctx, http.MethodPost, "v1/procore/sync", form)
	if err != nil {
		return nil, nil, err
	}
	sync := new(ProcoreSync)
	resp, err := s.client.Do(req, sync)
	if err != nil {
		return nil, resp, err
	}
	return sync, resp, nil
}

// GetSync returns the details and health of one sync.
//
// GET /pubapi/v1/procore/sync/{syncId}
func (s *ProcoreService) GetSync(ctx context.Context, syncID string) (*ProcoreSyncedProject, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/procore/sync/"+url.PathEscape(syncID), nil)
	if err != nil {
		return nil, nil, err
	}
	project := new(ProcoreSyncedProject)
	resp, err := s.client.Do(req, project)
	if err != nil {
		return nil, resp, err
	}
	return project, resp, nil
}

// UpdateSync changes a sync's option ("2-way sync" or "no sync") and
// optionally its Procore folder scope.
//
// PATCH /pubapi/v1/procore/sync/{syncId}
func (s *ProcoreService) UpdateSync(ctx context.Context, syncID, syncOption string, procoreFolderPaths []string) (*ProcoreSync, *Response, error) {
	if syncOption == "" {
		return nil, nil, fmt.Errorf("egnyte: syncOption is required")
	}
	form := url.Values{}
	form.Set("syncOption", syncOption)
	if len(procoreFolderPaths) > 0 {
		form.Set("procoreFolderPaths", strings.Join(procoreFolderPaths, ","))
	}
	req, err := s.client.NewFormRequest(ctx, http.MethodPatch, "v1/procore/sync/"+url.PathEscape(syncID), form)
	if err != nil {
		return nil, nil, err
	}
	sync := new(ProcoreSync)
	resp, err := s.client.Do(req, sync)
	if err != nil {
		return nil, resp, err
	}
	return sync, resp, nil
}

// DeleteSync removes a sync.
//
// DELETE /pubapi/v1/procore/sync/{syncId}
func (s *ProcoreService) DeleteSync(ctx context.Context, syncID string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v1/procore/sync/"+url.PathEscape(syncID), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// ListProjects returns the Procore projects of a company account.
//
// GET /pubapi/v1/procore/company/{companyId}/projects
func (s *ProcoreService) ListProjects(ctx context.Context, companyID string) ([]ProcoreProject, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/procore/company/"+url.PathEscape(companyID)+"/projects", nil)
	if err != nil {
		return nil, nil, err
	}
	var projects []ProcoreProject
	resp, err := s.client.Do(req, &projects)
	if err != nil {
		return nil, resp, err
	}
	return projects, resp, nil
}
