package egnyte

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestProcoreService_ListSyncedProjects(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/procore/sync", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`[{"id":"11111111","jobId":"22222222","folder":"/Shared/test",
			"procoreFolderPaths":[],"project":"project_name","procoreProjectId":54321,
			"serviceAccountName":"serviceAccount","syncType":"2-way sync",
			"lastSuccessTime":"2024-01-15T10:00:00Z","isHealthy":true,
			"isHealthyNote":null,"skipped":[]}]`))
	})

	projects, _, err := client.Procore.ListSyncedProjects(context.Background())
	if err != nil {
		t.Fatalf("ListSyncedProjects: %v", err)
	}
	if len(projects) != 1 || projects[0].ProcoreProjectID != 54321 || !projects[0].IsHealthy {
		t.Errorf("projects = %+v", projects)
	}
}

func TestProcoreService_CreateSync(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/procore/sync", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "Content-Type", "application/x-www-form-urlencoded")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.PostForm.Get("folderPath") != "/Shared/test" ||
			r.PostForm.Get("companyId") != "12345" ||
			r.PostForm.Get("procoreProjectId") != "54321" ||
			r.PostForm.Get("procoreFolderPaths") != "/01 Design Files,/03 Safety/01 Inbound" {
			t.Errorf("form = %v", r.PostForm)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"11111111","egnyteFolderPath":"/Shared/test",
			"procoreProjectId":54321,"syncType":"2-way sync","companyId":12345}`))
	})

	sync, resp, err := client.Procore.CreateSync(context.Background(), "/Shared/test", "12345", "54321",
		[]string{"/01 Design Files", "/03 Safety/01 Inbound"})
	if err != nil {
		t.Fatalf("CreateSync: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || sync.CompanyID != 12345 {
		t.Errorf("sync = %+v, status = %d", sync, resp.StatusCode)
	}
}

func TestProcoreService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.Procore.CreateSync(ctx, "/Shared/test", "", "54321", nil); err == nil {
		t.Error("missing companyId should fail before sending")
	}
	if _, _, err := client.Procore.UpdateSync(ctx, "s1", "", nil); err == nil {
		t.Error("missing syncOption should fail before sending")
	}
}

func TestProcoreService_GetUpdateDeleteSync(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/procore/sync/11111111", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"id":"11111111","folder":"/Shared/test","isHealthy":false,
				"isHealthyNote":"Sync error","skipped":[{"path":"/Shared/test/big.bin",
				"size":123,"errorCode":"TOO_LARGE"}]}`))
		case http.MethodPatch:
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			if r.PostForm.Get("syncOption") != "no sync" {
				t.Errorf("form = %v", r.PostForm)
			}
			w.Write([]byte(`{"id":"11111111","syncType":"no sync"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	project, _, err := client.Procore.GetSync(ctx, "11111111")
	if err != nil {
		t.Fatalf("GetSync: %v", err)
	}
	if project.IsHealthy || project.IsHealthyNote != "Sync error" || len(project.Skipped) != 1 {
		t.Errorf("project = %+v", project)
	}

	sync, _, err := client.Procore.UpdateSync(ctx, "11111111", "no sync", nil)
	if err != nil {
		t.Fatalf("UpdateSync: %v", err)
	}
	if sync.SyncType != "no sync" {
		t.Errorf("sync = %+v", sync)
	}

	if _, err := client.Procore.DeleteSync(ctx, "11111111"); err != nil {
		t.Fatalf("DeleteSync: %v", err)
	}
}

func TestProcoreService_ListProjects(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/procore/company/12345/projects", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`[{"id":54321,"name":"My Procore Project","isSynced":false,"companyId":12345}]`))
	})

	projects, _, err := client.Procore.ListProjects(context.Background(), "12345")
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "My Procore Project" {
		t.Errorf("projects = %+v", projects)
	}
}

func TestProcoreService_GetSync_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/procore/sync/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"not_found","message":"Sync not found"}`))
	})

	_, _, err := client.Procore.GetSync(context.Background(), "nope")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError = %+v", apiErr)
	}
}
