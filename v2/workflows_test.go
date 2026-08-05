package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestWorkflowsService_Create(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/workflows", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		file, _ := body["file"].(map[string]any)
		if body["name"] != "Demo Workflow" || body["workflowType"] != "REVIEW_APPROVE" ||
			file["groupId"] != "3d1d09d3" {
			t.Errorf("body = %v", body)
		}
		steps, _ := body["steps"].([]any)
		if len(steps) != 2 {
			t.Fatalf("steps = %v", body["steps"])
		}
		step0, _ := steps[0].(map[string]any)
		opts, _ := step0["stepOptions"].(map[string]any)
		assignees, _ := opts["assignees"].([]any)
		if step0["type"] != "REVIEW" || len(assignees) != 2 || opts["minMustComplete"] != float64(1) {
			t.Errorf("step[0] = %v", step0)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"workflowId":"403eeaaf"}`))
	})

	id, resp, err := client.Workflows.Create(context.Background(), CreateWorkflowRequest{
		Name:         "Demo Workflow",
		WorkflowType: "REVIEW_APPROVE",
		File:         WorkflowFileRef{GroupID: "3d1d09d3"},
		Steps: []CreateWorkflowStep{
			{Name: "Document Review", Type: "REVIEW", Description: "Please review",
				StepOptions: WorkflowStepOptions{Assignees: []int{1, 46}, MinMustComplete: 1,
					DueDate: "2022-02-24T18:00:00Z"}},
			{Name: "Document Approval", Type: "APPROVAL",
				StepOptions: WorkflowStepOptions{Assignees: []int{16}, SignatureRequired: Bool(false)}},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || id != "403eeaaf" {
		t.Errorf("id = %q, status = %d", id, resp.StatusCode)
	}
}

func TestWorkflowsService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.Workflows.Create(ctx, CreateWorkflowRequest{Name: "x"}); err == nil {
		t.Error("workflow without file/steps should fail before sending")
	}
	if _, _, err := client.Workflows.Create(ctx, CreateWorkflowRequest{
		Name: "x", File: WorkflowFileRef{GroupID: "g"},
		Steps: []CreateWorkflowStep{{Name: "s", Type: "REVIEW"}},
	}); err == nil {
		t.Error("step without assignees should fail before sending")
	}
}

func TestWorkflowsService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/workflows", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("status") != "IN_PROGRESS" || q.Get("assigneeId") != "46" || q.Get("limit") != "10" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"results":[{"id":"0ec5926c","name":"Demo Workflow",
			"workflowDisplayId":85,"type":"REVIEW_APPROVE","status":"IN_PROGRESS",
			"creator":{"id":1,"username":"jsmith"},"currentStepNum":1,"totalSteps":2,
			"assignees":[{"id":1,"username":"grandolph"}],
			"file":{"groupId":"3d1d09d3","entryId":"19efb714"}}],"totalCount":85}`))
	})

	list, _, err := client.Workflows.List(context.Background(), &WorkflowListOptions{
		AssigneeID: 46,
		Status:     "IN_PROGRESS",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.TotalCount != 85 || len(list.Results) != 1 {
		t.Fatalf("list = %+v", list)
	}
	wf := list.Results[0]
	if wf.WorkflowDisplayID != 85 || wf.Creator.Username != "jsmith" || wf.File.EntryID != "19efb714" {
		t.Errorf("workflow = %+v", wf)
	}
}

func TestWorkflowsService_Get(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/workflows/0ec5926c", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"id":"0ec5926c","name":"Demo Workflow","status":"IN_PROGRESS",
			"creator":{"id":1,"username":"jsmith"},
			"steps":[{"id":"85627761","name":"Document Review","type":"REVIEW",
			"status":"IN_PROGRESS","signatureRequired":null,"minMustComplete":null,
			"tasks":[{"id":"2b4a7375","type":"REVIEW","status":"IN_PROGRESS"}]}]}`))
	})

	details, _, err := client.Workflows.Get(context.Background(), "0ec5926c")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(details.Steps) != 1 || details.Steps[0].SignatureRequired != nil {
		t.Fatalf("details = %+v", details)
	}
	if len(details.Steps[0].Tasks) != 1 || details.Steps[0].Tasks[0].ID != "2b4a7375" {
		t.Errorf("tasks = %+v", details.Steps[0].Tasks)
	}
}

func TestWorkflowsService_Cancel(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/workflows/0ec5926c/cancel", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Workflows.Cancel(context.Background(), "0ec5926c"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
}

func TestWorkflowsService_ListTasks(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/workflows/tasks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("status") != "IN_PROGRESS" || q.Get("sortBy") != "DUE_DATE" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"results":[{"id":"37296067","name":"Review","status":"IN_PROGRESS",
			"taskType":"REVIEW","workflow":{"id":"f1ae9f50","name":"Policy Update"},
			"assigner":{"id":1,"username":"jsmith"},
			"file":{"groupId":"3d1d09d3"}}],"totalCount":28}`))
	})

	list, _, err := client.Workflows.ListTasks(context.Background(), &WorkflowTaskListOptions{
		Status: "IN_PROGRESS",
		SortBy: "DUE_DATE",
	})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if list.TotalCount != 28 || len(list.Results) != 1 {
		t.Fatalf("list = %+v", list)
	}
	task := list.Results[0]
	if task.TaskType != "REVIEW" || task.Workflow.Name != "Policy Update" {
		t.Errorf("task = %+v", task)
	}
}

func TestWorkflowsService_SendCompletionSignal(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/workflows/wf1/tasks/task1/completion-signal", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Workflows.SendCompletionSignal(context.Background(), "wf1", "task1"); err != nil {
		t.Fatalf("SendCompletionSignal: %v", err)
	}
}

func TestWorkflowsService_Get_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/workflows/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Workflow not found"}`))
	})

	_, _, err := client.Workflows.Get(context.Background(), "nope")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError = %+v", apiErr)
	}
}
