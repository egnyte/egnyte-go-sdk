package egnyte

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// AuditService generates audit reports (v1, asynchronous jobs) and
// streams near-real-time audit events (v2).
//
// API documentation:
//   - v1 reports: https://developers.egnyte.com/integration/cfs/api-docs/audit-reporting-api/v1
//   - v2 stream: https://developers.egnyte.com/integration/cfs/api-docs/audit-reporting-api/v2
//
// OAuth scope: Egnyte.audit. The authenticated user must be an administrator
// or a power user allowed to run reports.
type AuditService service

// AuditReportRequest carries the fields common to every v1 report:
// Format ("csv" or "json"), DateStart and DateEnd (YYYY-MM-DD or
// ISO 8601) are required.
type AuditReportRequest struct {
	Format    string `json:"format"`
	DateStart string `json:"date_start"`
	DateEnd   string `json:"date_end"`
	// SuppressEmails suppresses the completion notification email.
	SuppressEmails *bool `json:"suppress_emails,omitempty"`
}

func (r AuditReportRequest) validate() error {
	if r.Format == "" || r.DateStart == "" || r.DateEnd == "" {
		return fmt.Errorf("egnyte: audit report format, date_start and date_end are required")
	}
	return nil
}

// LoginAuditReportRequest requests a login activity report. Events is
// required (logins, logouts, account_lockouts, password_resets,
// failed_attempts).
type LoginAuditReportRequest struct {
	AuditReportRequest
	Events       []string `json:"events"`
	AccessPoints []string `json:"access_points,omitempty"` // Web, FTP, Mobile
	Users        []string `json:"users,omitempty"`
}

// FileAuditReportRequest requests a file activity report. At least one
// of Folders or File must be set.
type FileAuditReportRequest struct {
	AuditReportRequest
	Folders []string `json:"folders,omitempty"`
	// File is a specific file path; supports a trailing * wildcard.
	File  string   `json:"file,omitempty"`
	Users []string `json:"users,omitempty"`
	// TransactionType filters to specific transactions (upload,
	// download, preview, delete, copy, move, create_folder, ...).
	TransactionType []string `json:"transaction_type,omitempty"`
}

// PermissionsAuditReportRequest requests a permission-change report.
type PermissionsAuditReportRequest struct {
	AuditReportRequest
	Folders        []string `json:"folders,omitempty"`
	Assigners      []string `json:"assigners,omitempty"`
	AssigneeUsers  []string `json:"assignee_users,omitempty"`
	AssigneeGroups []string `json:"assignee_groups,omitempty"`
}

// UserAuditReportRequest requests a user provisioning report.
type UserAuditReportRequest struct {
	AuditReportRequest
	// ActionType filters to CREATE, UPDATE, DISABLE, ENABLE, DELETE,
	// PASSWORD_RESET, PASSWORD_CHANGE.
	ActionType           []string `json:"action_type,omitempty"`
	PerformedBy          []string `json:"performed_by,omitempty"`
	IncludeSystemActions *bool    `json:"include_system_actions,omitempty"`
	// Subject filters to the users that were modified.
	Subject []string `json:"subject,omitempty"`
}

// GroupAuditReportRequest requests a group provisioning report.
type GroupAuditReportRequest struct {
	AuditReportRequest
	// ActionType filters to CREATE, ADD_USERS, REMOVE_USERS, RENAME, DELETE.
	ActionType           []string `json:"action_type,omitempty"`
	Users                []string `json:"users,omitempty"`
	IncludeSystemActions *bool    `json:"include_system_actions,omitempty"`
	Groups               []string `json:"groups,omitempty"`
}

// WorkgroupSettingsAuditReportRequest requests a configuration settings
// report.
type WorkgroupSettingsAuditReportRequest struct {
	AuditReportRequest
	Users []string `json:"users,omitempty"`
}

// WorkflowAuditReportRequest requests a workflow activity report.
type WorkflowAuditReportRequest struct {
	AuditReportRequest
	Users []string `json:"users,omitempty"`
	// File is a search pattern filtering workflows by name (supports *).
	File            string   `json:"file,omitempty"`
	WorkflowActions []string `json:"workflow_actions,omitempty"`
	WorkflowTypes   []string `json:"workflow_types,omitempty"`
}

// WorkflowTemplatesAuditReportRequest requests a workflow template
// changes report.
type WorkflowTemplatesAuditReportRequest struct {
	AuditReportRequest
	Users                   []string `json:"users,omitempty"`
	WorkflowTemplateActions []string `json:"workflow_template_actions,omitempty"`
}

// QualityDocsCategoryFilter narrows Quality Docs reports to categories
// and optional subcategories.
type QualityDocsCategoryFilter struct {
	ID             string   `json:"id"`
	SubcategoryIDs []string `json:"subcategoryIds,omitempty"`
}

// QualityDocsAuditReportRequest requests a Quality Docs activity report.
type QualityDocsAuditReportRequest struct {
	AuditReportRequest
	Users                 []string                    `json:"users,omitempty"`
	QualityDocumentID     string                      `json:"quality_document_id,omitempty"`
	QualityDocsCategories []QualityDocsCategoryFilter `json:"quality_docs_categories,omitempty"`
	QualityDocsActions    []string                    `json:"quality_docs_actions,omitempty"`
	QualityDocsStatuses   []string                    `json:"quality_docs_statuses,omitempty"`
}

// QualityDocsCategoriesAuditReportRequest requests a Quality Docs
// category changes report.
type QualityDocsCategoriesAuditReportRequest struct {
	AuditReportRequest
	Users                        []string                    `json:"users,omitempty"`
	QualityDocsCategories        []QualityDocsCategoryFilter `json:"quality_docs_categories,omitempty"`
	QualityDocsCategoriesActions []string                    `json:"quality_docs_categories_actions,omitempty"`
}

// QualityDocsCoursesAuditReportRequest requests a Quality Docs training
// course report.
type QualityDocsCoursesAuditReportRequest struct {
	AuditReportRequest
	Users                     []string `json:"users,omitempty"`
	QualityDocsCoursesIDs     []string `json:"quality_docs_courses_ids,omitempty"`
	QualityDocsCoursesActions []string `json:"quality_docs_courses_actions,omitempty"`
	QualityDocsCoursesParams  []string `json:"quality_docs_courses_parameters,omitempty"`
}

// ETMFAuditReportRequest requests an eTMF activity report.
type ETMFAuditReportRequest struct {
	AuditReportRequest
	Users              []string `json:"users,omitempty"`
	ETMFActions        []string `json:"etmf_actions,omitempty"`
	ETMFStudyIDs       []string `json:"etmf_study_ids,omitempty"`
	ETMFArtifactNumber string   `json:"etmf_artifact_number,omitempty"`
}

// SnapshotRestoreAuditReportRequest requests a snapshot restore report.
type SnapshotRestoreAuditReportRequest struct {
	AuditReportRequest
	Users                []string `json:"users,omitempty"`
	IncludeSystemActions *bool    `json:"include_system_actions,omitempty"`
}

// UploadRequestsAuditReportRequest requests an upload request activity
// report.
type UploadRequestsAuditReportRequest struct {
	AuditReportRequest
	Users                 []string `json:"users,omitempty"`
	UploadRequestsActions []string `json:"upload_requests_actions,omitempty"`
}

// AuditJobStatus is the state of a report generation job. When Status is
// "completed", ReportURL carries the Location the API redirected to.
type AuditJobStatus struct {
	Status    string `json:"status"` // running or completed
	ReportURL string `json:"-"`
}

// AuditReport is a JSON-format report page.
type AuditReport struct {
	Events     []map[string]any `json:"events"`
	TotalCount int              `json:"totalCount"`
}

// AuditReportPageOptions paginate JSON-format report retrieval.
type AuditReportPageOptions struct {
	Offset int
	Count  int
}

// AuditStreamRequest requests v2 audit events. Exactly one of StartDate
// (initial request, within the last 7 days) or NextCursor (subsequent
// requests) must be set.
type AuditStreamRequest struct {
	StartDate  string `json:"startDate,omitempty"`
	EndDate    string `json:"endDate,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
	// AuditType filters events (FILE_AUDIT, LOGIN_AUDIT, ...). Defaults
	// to ANY.
	AuditType []string `json:"auditType,omitempty"`
}

// AuditStream is one page of the v2 event stream (up to 5,000 events).
type AuditStream struct {
	NextCursor string           `json:"nextCursor"`
	Events     []map[string]any `json:"events"`
	MoreEvents bool             `json:"moreEvents"`
}

func (s *AuditService) createReport(ctx context.Context, urlPath string, base AuditReportRequest, body any) (string, *Response, error) {
	if err := base.validate(); err != nil {
		return "", nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, urlPath, body)
	if err != nil {
		return "", nil, err
	}
	var job struct {
		ID string `json:"id"`
	}
	resp, err := s.client.Do(req, &job)
	if err != nil {
		return "", resp, err
	}
	return job.ID, resp, nil
}

// CreateLoginsReport starts generation of a login audit report and
// returns the job id to poll with JobStatus.
//
// POST /pubapi/v1/audit/logins
func (s *AuditService) CreateLoginsReport(ctx context.Context, r LoginAuditReportRequest) (string, *Response, error) {
	if len(r.Events) == 0 {
		return "", nil, fmt.Errorf("egnyte: login audit report requires at least one event type")
	}
	return s.createReport(ctx, "v1/audit/logins", r.AuditReportRequest, r)
}

// CreateFilesReport starts generation of a file audit report.
//
// POST /pubapi/v1/audit/files
func (s *AuditService) CreateFilesReport(ctx context.Context, r FileAuditReportRequest) (string, *Response, error) {
	if len(r.Folders) == 0 && r.File == "" {
		return "", nil, fmt.Errorf("egnyte: file audit report requires folders or file")
	}
	return s.createReport(ctx, "v1/audit/files", r.AuditReportRequest, r)
}

// CreatePermissionsReport starts generation of a permissions audit report.
//
// POST /pubapi/v1/audit/permissions
func (s *AuditService) CreatePermissionsReport(ctx context.Context, r PermissionsAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/permissions", r.AuditReportRequest, r)
}

// CreateUsersReport starts generation of a user provisioning audit report.
//
// POST /pubapi/v1/audit/users
func (s *AuditService) CreateUsersReport(ctx context.Context, r UserAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/users", r.AuditReportRequest, r)
}

// CreateGroupsReport starts generation of a group provisioning audit report.
//
// POST /pubapi/v1/audit/groups
func (s *AuditService) CreateGroupsReport(ctx context.Context, r GroupAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/groups", r.AuditReportRequest, r)
}

// CreateWorkgroupSettingsReport starts generation of a configuration
// settings audit report.
//
// POST /pubapi/v1/audit/workgroup-settings
func (s *AuditService) CreateWorkgroupSettingsReport(ctx context.Context, r WorkgroupSettingsAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/workgroup-settings", r.AuditReportRequest, r)
}

// CreateWorkflowsReport starts generation of a workflow audit report.
//
// POST /pubapi/v1/audit/workflows
func (s *AuditService) CreateWorkflowsReport(ctx context.Context, r WorkflowAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/workflows", r.AuditReportRequest, r)
}

// CreateWorkflowTemplatesReport starts generation of a workflow template
// audit report.
//
// POST /pubapi/v1/audit/workflow-templates
func (s *AuditService) CreateWorkflowTemplatesReport(ctx context.Context, r WorkflowTemplatesAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/workflow-templates", r.AuditReportRequest, r)
}

// CreateQualityDocsReport starts generation of a Quality Docs audit report.
//
// POST /pubapi/v1/audit/quality-docs
func (s *AuditService) CreateQualityDocsReport(ctx context.Context, r QualityDocsAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/quality-docs", r.AuditReportRequest, r)
}

// CreateQualityDocsCategoriesReport starts generation of a Quality Docs
// categories audit report.
//
// POST /pubapi/v1/audit/quality-docs-categories
func (s *AuditService) CreateQualityDocsCategoriesReport(ctx context.Context, r QualityDocsCategoriesAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/quality-docs-categories", r.AuditReportRequest, r)
}

// CreateQualityDocsCoursesReport starts generation of a Quality Docs
// courses audit report.
//
// POST /pubapi/v1/audit/quality-docs-courses
func (s *AuditService) CreateQualityDocsCoursesReport(ctx context.Context, r QualityDocsCoursesAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/quality-docs-courses", r.AuditReportRequest, r)
}

// CreateETMFReport starts generation of an eTMF audit report.
//
// POST /pubapi/v1/audit/etmf
func (s *AuditService) CreateETMFReport(ctx context.Context, r ETMFAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/etmf", r.AuditReportRequest, r)
}

// CreateSnapshotRestoreReport starts generation of a snapshot restore
// audit report.
//
// POST /pubapi/v1/audit/snapshot-restore
func (s *AuditService) CreateSnapshotRestoreReport(ctx context.Context, r SnapshotRestoreAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/snapshot-restore", r.AuditReportRequest, r)
}

// CreateUploadRequestsReport starts generation of an upload requests
// audit report.
//
// POST /pubapi/v1/audit/upload-requests
func (s *AuditService) CreateUploadRequestsReport(ctx context.Context, r UploadRequestsAuditReportRequest) (string, *Response, error) {
	return s.createReport(ctx, "v1/audit/upload-requests", r.AuditReportRequest, r)
}

// JobStatus reports whether a report generation job is still running.
// When the job is complete the API responds 303 See Other; the returned
// status is "completed" and ReportURL carries the redirect target. Poll
// no more than once every two minutes.
//
// GET /pubapi/v1/audit/jobs/{id}
func (s *AuditService) JobStatus(ctx context.Context, jobID string) (*AuditJobStatus, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/audit/jobs/"+url.PathEscape(jobID), nil)
	if err != nil {
		return nil, nil, err
	}
	// The completion signal is a 303 redirect that must not be followed
	// automatically, so this request runs on a redirect-suppressed copy
	// of the client.
	hc := *s.client.httpClient
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	rawResp, err := hc.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer rawResp.Body.Close()
	resp := newResponse(rawResp)
	if rawResp.StatusCode >= 400 {
		return nil, resp, newAPIError(rawResp)
	}

	status := new(AuditJobStatus)
	if rawResp.StatusCode == http.StatusSeeOther {
		status.Status = "completed"
		status.ReportURL = rawResp.Header.Get("Location")
		return status, resp, nil
	}
	if err := decodeJSON(rawResp.Body, status, s.client.maxResponseBytes); err != nil {
		return nil, resp, fmt.Errorf("egnyte: decode %s %s response: %w",
			req.Method, req.URL.Path, err)
	}
	return status, resp, nil
}

// GetReport retrieves a completed JSON-format report. reportType is the
// path segment the report was created under (logins, files, permissions,
// users, groups, workgroup-settings, workflows, workflow-templates,
// quality-docs, quality-docs-categories, quality-docs-courses, etmf,
// snapshot-restore, upload-requests).
//
// GET /pubapi/v1/audit/{type}/{id}
func (s *AuditService) GetReport(ctx context.Context, reportType, id string, opts *AuditReportPageOptions) (*AuditReport, *Response, error) {
	urlPath := s.reportPath(reportType, id)
	if opts != nil {
		q := url.Values{}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
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
	report := new(AuditReport)
	resp, err := s.client.Do(req, report)
	if err != nil {
		return nil, resp, err
	}
	return report, resp, nil
}

// DownloadReport streams a completed report in its raw form — use this
// for CSV-format reports. The caller must close the returned ReadCloser.
//
// GET /pubapi/v1/audit/{type}/{id}
func (s *AuditService) DownloadReport(ctx context.Context, reportType, id string) (io.ReadCloser, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, s.reportPath(reportType, id), nil)
	if err != nil {
		return nil, nil, err
	}
	return s.client.DoRaw(req)
}

// DeleteReport deletes a completed report.
//
// DELETE /pubapi/v1/audit/{type}/{id}
func (s *AuditService) DeleteReport(ctx context.Context, reportType, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, s.reportPath(reportType, id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

func (s *AuditService) reportPath(reportType, id string) string {
	return "v1/audit/" + url.PathEscape(reportType) + "/" + url.PathEscape(id)
}

// Stream retrieves up to 5,000 audit events from the last seven days via
// the POST endpoint. Follow AuditStream.NextCursor for subsequent pages.
// Rate limited to 10 requests/minute and 100 requests/hour.
//
// POST /pubapi/v2/audit/stream
func (s *AuditService) Stream(ctx context.Context, r AuditStreamRequest) (*AuditStream, *Response, error) {
	if err := validateStreamRequest(r); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/audit/stream", r)
	if err != nil {
		return nil, nil, err
	}
	stream := new(AuditStream)
	resp, err := s.client.Do(req, stream)
	if err != nil {
		return nil, resp, err
	}
	return stream, resp, nil
}

// StreamGet is Stream via the GET endpoint, passing the same parameters
// as query strings.
//
// GET /pubapi/v2/audit/stream
func (s *AuditService) StreamGet(ctx context.Context, r AuditStreamRequest) (*AuditStream, *Response, error) {
	if err := validateStreamRequest(r); err != nil {
		return nil, nil, err
	}
	q := url.Values{}
	if r.StartDate != "" {
		q.Set("startDate", r.StartDate)
	}
	if r.EndDate != "" {
		q.Set("endDate", r.EndDate)
	}
	if r.NextCursor != "" {
		q.Set("nextCursor", r.NextCursor)
	}
	for _, t := range r.AuditType {
		q.Add("auditType", t)
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v2/audit/stream?"+q.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}
	stream := new(AuditStream)
	resp, err := s.client.Do(req, stream)
	if err != nil {
		return nil, resp, err
	}
	return stream, resp, nil
}

func validateStreamRequest(r AuditStreamRequest) error {
	if (r.StartDate == "") == (r.NextCursor == "") {
		return fmt.Errorf("egnyte: audit stream requires exactly one of startDate or nextCursor")
	}
	return nil
}
