package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// WorkflowsService orchestrates review and approval processes on files.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/workflow-api
// OAuth scope: Egnyte.filesystem. File permissions and workflow role rules
// still apply.
type WorkflowsService service

// WorkflowUser identifies a workflow participant.
type WorkflowUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

// WorkflowFileRef identifies the file a workflow runs on.
type WorkflowFileRef struct {
	GroupID string `json:"groupId"`
	EntryID string `json:"entryId,omitempty"`
}

// WorkflowStepOptions configure one step of a new workflow.
type WorkflowStepOptions struct {
	// Assignees lists the numeric user ids the step is assigned to.
	Assignees []int `json:"assignees"`
	// MinMustComplete is how many assignees must complete the step.
	MinMustComplete int    `json:"minMustComplete,omitempty"`
	DueDate         string `json:"dueDate,omitempty"` // ISO-8601
	// SignatureRequired requires a digital signature (approval steps).
	SignatureRequired *bool `json:"signatureRequired,omitempty"`
}

// CreateWorkflowStep defines one step of a new workflow.
type CreateWorkflowStep struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Type is TODO, REVIEW or APPROVAL.
	Type        string              `json:"type"`
	StepOptions WorkflowStepOptions `json:"stepOptions"`
}

// CreateWorkflowRequest is the body for creating a workflow.
type CreateWorkflowRequest struct {
	Name            string          `json:"name"`
	File            WorkflowFileRef `json:"file"`
	ReasonForChange string          `json:"reasonForChange,omitempty"`
	// WorkflowType is TODO, APPROVAL or REVIEW_APPROVE.
	WorkflowType string               `json:"workflowType,omitempty"`
	Steps        []CreateWorkflowStep `json:"steps"`
}

// Workflow is a workflow summary as returned by List.
type Workflow struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	ReasonForChange     string          `json:"reasonForChange"`
	CreationDate        string          `json:"creationDate"`
	DueDate             string          `json:"dueDate"`
	CompletionDate      string          `json:"completionDate"`
	WorkflowDisplayID   int             `json:"workflowDisplayId"`
	Type                string          `json:"type"`
	Status              string          `json:"status"`           // NOT_STARTED, IN_PROGRESS, ...
	CompletionStatus    string          `json:"completionStatus"` // APPROVED or REJECTED once completed
	Creator             WorkflowUser    `json:"creator"`
	CurrentStepNum      int             `json:"currentStepNum"`
	TotalSteps          int             `json:"totalSteps"`
	CurrentStepProgress int             `json:"currentStepProgress"`
	Assignees           []WorkflowUser  `json:"assignees"`
	File                WorkflowFileRef `json:"file"`
}

// WorkflowStepTask is one task inside a workflow step.
type WorkflowStepTask struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// WorkflowStep is one step of a workflow's details.
type WorkflowStep struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	Type              string             `json:"type"`
	Status            string             `json:"status"`
	DueDate           string             `json:"dueDate"`
	StartDate         string             `json:"startDate"`
	CompletionDate    string             `json:"completionDate"`
	SignatureRequired *bool              `json:"signatureRequired"`
	MinMustComplete   *int               `json:"minMustComplete"`
	Tasks             []WorkflowStepTask `json:"tasks"`
}

// WorkflowDetails is the full state of one workflow.
type WorkflowDetails struct {
	ID                       string         `json:"id"`
	Name                     string         `json:"name"`
	ReasonForChange          string         `json:"reasonForChange"`
	Status                   string         `json:"status"`
	WorkflowCompletionStatus string         `json:"workflowCompletionStatus"`
	CreationDate             string         `json:"creationDate"`
	CompletionDate           string         `json:"completionDate"`
	Creator                  WorkflowUser   `json:"creator"`
	Steps                    []WorkflowStep `json:"steps"`
}

// WorkflowList is a page of workflows.
type WorkflowList struct {
	Results    []Workflow `json:"results"`
	TotalCount int        `json:"totalCount"`
}

// WorkflowListOptions filter workflow listings. Zero values are omitted.
type WorkflowListOptions struct {
	// AssignerID returns workflows created by a user; AssigneeID
	// returns workflows assigned to a user.
	AssignerID int
	AssigneeID int
	// GroupID returns workflows on a specific file.
	GroupID string
	// Status is NOT_STARTED, IN_PROGRESS, BEING_CANCELLED, CANCELLED,
	// SUSPENDED or COMPLETED.
	Status string
	// CompletionStatus filters completed workflows: APPROVED or REJECTED.
	CompletionStatus string
	// Limit is the page size (1-25, default 25).
	Limit  int
	Offset int
	// SortBy is NAME, CREATION_DATE, DUE_DATE, COMPLETION_DATE,
	// WORKFLOW_STATUS or WORKFLOW_COMPLETION_STATUS.
	SortBy string
	// SortDirection is ASC or DESC.
	SortDirection string
}

// WorkflowTask is a task assigned to the current user.
type WorkflowTask struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	CreationDate   string `json:"creationDate"`
	DueDate        string `json:"dueDate"`
	CompletionDate string `json:"completionDate"`
	Status         string `json:"status"`
	TaskType       string `json:"taskType"`
	Workflow       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"workflow"`
	Assigner WorkflowUser    `json:"assigner"`
	File     WorkflowFileRef `json:"file"`
}

// WorkflowTaskList is a page of tasks.
type WorkflowTaskList struct {
	Results    []WorkflowTask `json:"results"`
	TotalCount int            `json:"totalCount"`
}

// WorkflowTaskListOptions filter task listings.
type WorkflowTaskListOptions struct {
	// Status is NOT_STARTED, IN_PROGRESS, BEING_CANCELLED, CANCELLED,
	// SUSPENDED, APPROVED, REJECTED or COMPLETED.
	Status string
	// Limit is the page size (1-50).
	Limit  int
	Offset int
	// SortBy is NAME, DUE_DATE, STATUS or TASK_TYPE.
	SortBy string
	// SortDirection is ASC or DESC.
	SortDirection string
}

// Create starts a new workflow on a file and returns the workflow id.
//
// POST /pubapi/v1/workflows
func (s *WorkflowsService) Create(ctx context.Context, workflow CreateWorkflowRequest) (string, *Response, error) {
	if workflow.Name == "" || workflow.File.GroupID == "" || len(workflow.Steps) == 0 {
		return "", nil, fmt.Errorf("egnyte: workflow name, file groupId and steps are required")
	}
	for _, step := range workflow.Steps {
		if step.Name == "" || step.Type == "" || len(step.StepOptions.Assignees) == 0 {
			return "", nil, fmt.Errorf("egnyte: each workflow step requires name, type and assignees")
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/workflows", workflow)
	if err != nil {
		return "", nil, err
	}
	var created struct {
		WorkflowID string `json:"workflowId"`
	}
	resp, err := s.client.Do(req, &created)
	if err != nil {
		return "", resp, err
	}
	return created.WorkflowID, resp, nil
}

// List returns workflows visible to the caller (admins see all, others
// only their own).
//
// GET /pubapi/v1/workflows
func (s *WorkflowsService) List(ctx context.Context, opts *WorkflowListOptions) (*WorkflowList, *Response, error) {
	urlPath := "v1/workflows"
	if opts != nil {
		q := url.Values{}
		if opts.AssignerID > 0 {
			q.Set("assignerId", strconv.Itoa(opts.AssignerID))
		}
		if opts.AssigneeID > 0 {
			q.Set("assigneeId", strconv.Itoa(opts.AssigneeID))
		}
		setString := func(name, val string) {
			if val != "" {
				q.Set(name, val)
			}
		}
		setString("groupId", opts.GroupID)
		setString("status", opts.Status)
		setString("completionStatus", opts.CompletionStatus)
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
		setString("sortBy", opts.SortBy)
		setString("sortDirection", opts.SortDirection)
		if enc := q.Encode(); enc != "" {
			urlPath += "?" + enc
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(WorkflowList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// Get returns the details of a workflow, including its steps and tasks.
//
// GET /pubapi/v1/workflows/{workflowId}
func (s *WorkflowsService) Get(ctx context.Context, id string) (*WorkflowDetails, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/workflows/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	details := new(WorkflowDetails)
	resp, err := s.client.Do(req, details)
	if err != nil {
		return nil, resp, err
	}
	return details, resp, nil
}

// Cancel cancels an in-progress workflow; cancelled workflows cannot be
// resumed.
//
// POST /pubapi/v1/workflows/{workflowId}/cancel
func (s *WorkflowsService) Cancel(ctx context.Context, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/workflows/"+url.PathEscape(id)+"/cancel", nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// ListTasks returns tasks assigned to the authenticated user.
//
// GET /pubapi/v1/workflows/tasks
func (s *WorkflowsService) ListTasks(ctx context.Context, opts *WorkflowTaskListOptions) (*WorkflowTaskList, *Response, error) {
	urlPath := "v1/workflows/tasks"
	if opts != nil {
		q := url.Values{}
		if opts.Status != "" {
			q.Set("status", opts.Status)
		}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
		if opts.SortBy != "" {
			q.Set("sortBy", opts.SortBy)
		}
		if opts.SortDirection != "" {
			q.Set("sortDirection", opts.SortDirection)
		}
		if enc := q.Encode(); enc != "" {
			urlPath += "?" + enc
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(WorkflowTaskList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// SendCompletionSignal marks an outbound-webhook task as completed.
// Invoke it after processing a webhook step's callback when the
// completion-signal option is enabled.
//
// POST /pubapi/v1/workflows/{workflowId}/tasks/{taskId}/completion-signal
func (s *WorkflowsService) SendCompletionSignal(ctx context.Context, workflowID, taskID string) (*Response, error) {
	urlPath := "v1/workflows/" + url.PathEscape(workflowID) + "/tasks/" + url.PathEscape(taskID) + "/completion-signal"
	req, err := s.client.NewRequest(ctx, http.MethodPost, urlPath, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
