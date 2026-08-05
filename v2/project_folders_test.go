package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestProjectFoldersService_ListAndCreateFromTemplate(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/project-folders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			q := r.URL.Query()
			if q.Get("startIndex") != "1" || q.Get("count") != "50" {
				t.Errorf("query = %v", q)
			}
			w.Write([]byte(`{"itemsPerPage":1,"totalResults":1,"startIndex":1,"resources":[
				{"id":"a69bd625","rootFolderId":"b133ce0d","name":"Acme Widgets HQ",
				 "projectId":"ABC-123","customerName":"Acme Widgets","status":"in-progress"}]}`))
		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			loc, _ := body["location"].(map[string]any)
			fields, _ := body["customFieldsWithValues"].(map[string]any)
			if body["parentFolderId"] != "7ab1234c" || body["templateFolderId"] != "9cd5678e" ||
				body["status"] != "pending" || loc["city"] != "Anytown" || fields["priority"] != "high" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(`{"groupsCreated":[
				{"id":"8b149a9e","name":"ABC123 - Project Team"},
				{"id":"a183d648","name":"ABC123 - Project Managers"}]}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	list, _, err := client.ProjectFolders.List(ctx, &ProjectListOptions{StartIndex: 1, Count: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.TotalResults != 1 || list.Resources[0].ProjectID != "ABC-123" {
		t.Errorf("list = %+v", list)
	}

	groups, _, err := client.ProjectFolders.CreateFromTemplate(ctx, CreateProjectFromTemplateRequest{
		ParentFolderID:         "7ab1234c",
		TemplateFolderID:       "9cd5678e",
		FolderName:             "ABC123 - 123 Main St",
		Name:                   "Acme Widgets HQ",
		Status:                 "pending",
		Location:               &ProjectLocation{City: "Anytown", State: "CA"},
		CustomFieldsWithValues: map[string]string{"priority": "high"},
	})
	if err != nil {
		t.Fatalf("CreateFromTemplate: %v", err)
	}
	if len(groups) != 2 || groups[0].Name != "ABC123 - Project Team" {
		t.Errorf("groups = %+v", groups)
	}
}

func TestProjectFoldersService_MarkFolderAsProject(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/project-folders", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["rootFolderId"] != "5bd7337c" || body["status"] != "pending" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"a69bd625","rootFolderId":"5bd7337c",
			"name":"Mountain View CA Project","status":"pending"}`))
	})

	project, resp, err := client.ProjectFolders.MarkFolderAsProject(context.Background(),
		MarkFolderAsProjectRequest{
			RootFolderID: "5bd7337c",
			Name:         "Mountain View CA Project",
			Status:       "pending",
		})
	if err != nil {
		t.Fatalf("MarkFolderAsProject: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || project.ID != "a69bd625" {
		t.Errorf("project = %+v, status = %d", project, resp.StatusCode)
	}
}

func TestProjectFoldersService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.ProjectFolders.MarkFolderAsProject(ctx, MarkFolderAsProjectRequest{Name: "x"}); err == nil {
		t.Error("missing rootFolderId/status should fail before sending")
	}
	if _, _, err := client.ProjectFolders.CreateFromTemplate(ctx, CreateProjectFromTemplateRequest{Name: "x"}); err == nil {
		t.Error("missing template fields should fail before sending")
	}
	if _, _, err := client.ProjectFolders.FindByRootFolder(ctx, ""); err == nil {
		t.Error("empty rootFolderId should fail before sending")
	}
}

func TestProjectFoldersService_GetUpdateDelete(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/project-folders/ABC-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"id":"a69bd625","projectId":"ABC-123","name":"Acme Widgets HQ Redesign",
				"status":"in-progress","customFieldsWithValues":{"priority":"high"}}`))
		case http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			del, _ := body["deleteCustomFields"].([]any)
			if body["status"] != "completed" || len(del) != 1 || del[0] != "region" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(`{"id":"a69bd625","projectId":"ABC-123","status":"completed"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	project, _, err := client.ProjectFolders.Get(ctx, "ABC-123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if project.CustomFieldsWithValues["priority"] != "high" {
		t.Errorf("project = %+v", project)
	}

	updated, _, err := client.ProjectFolders.Update(ctx, "ABC-123", UpdateProjectRequest{
		Status:             "completed",
		DeleteCustomFields: []string{"region"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Status != "completed" {
		t.Errorf("updated = %+v", updated)
	}

	if _, err := client.ProjectFolders.Delete(ctx, "ABC-123"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestProjectFoldersService_List_arrayShape(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/project-folders", func(w http.ResponseWriter, r *http.Request) {
		// The live API returns a bare array instead of the documented
		// paginated object.
		w.Write([]byte(`[{"id":"a69bd625","projectId":"ABC-123","name":"Acme HQ"}]`))
	})

	list, _, err := client.ProjectFolders.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.TotalResults != 1 || len(list.Resources) != 1 || list.Resources[0].ProjectID != "ABC-123" {
		t.Errorf("list = %+v", list)
	}
}

func TestProjectFoldersService_FindByRootFolder(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/project-folders/search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["rootFolderId"] != "b133ce0d" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`[{"id":"a69bd625","rootFolderId":"b133ce0d","projectId":"ABC-123"}]`))
	})

	projects, _, err := client.ProjectFolders.FindByRootFolder(context.Background(), "b133ce0d")
	if err != nil {
		t.Fatalf("FindByRootFolder: %v", err)
	}
	if len(projects) != 1 || projects[0].ProjectID != "ABC-123" {
		t.Errorf("projects = %+v", projects)
	}
}

func TestProjectFoldersService_Cleanup(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/project-folders/ABC-123/cleanup", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		toDelete, _ := body["usersToDelete"].([]any)
		if body["deleteLinks"] != true || len(toDelete) != 1 {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusAccepted)
	})

	resp, err := client.ProjectFolders.Cleanup(context.Background(), "ABC-123", ProjectCleanupRequest{
		DeleteLinks:   true,
		UsersToDelete: []int{12345},
	})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want 202", resp.StatusCode)
	}
}

func TestProjectFoldersService_Cleanup_conflict(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/project-folders/ABC-123/cleanup", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message":"Previous cleanup operation still in progress"}`))
	})

	_, err := client.ProjectFolders.Cleanup(context.Background(), "ABC-123", ProjectCleanupRequest{DeleteLinks: true})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("APIError = %+v", apiErr)
	}
}
