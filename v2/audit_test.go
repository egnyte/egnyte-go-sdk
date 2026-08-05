package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestAuditService_CreateLoginsReport(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/logins", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["format"] != "json" || body["date_start"] != "2019-05-01" || body["date_end"] != "2019-05-20" {
			t.Errorf("base fields = %v", body)
		}
		events, _ := body["events"].([]any)
		if len(events) != 2 || events[0] != "logins" {
			t.Errorf("events = %v", body["events"])
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"12345678"}`))
	})

	jobID, resp, err := client.Audit.CreateLoginsReport(context.Background(), LoginAuditReportRequest{
		AuditReportRequest: AuditReportRequest{Format: "json", DateStart: "2019-05-01", DateEnd: "2019-05-20"},
		Events:             []string{"logins", "failed_attempts"},
		AccessPoints:       []string{"Web"},
	})
	if err != nil {
		t.Fatalf("CreateLoginsReport: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted || jobID != "12345678" {
		t.Errorf("jobID = %q, status = %d", jobID, resp.StatusCode)
	}
}

func TestAuditService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	base := AuditReportRequest{Format: "json", DateStart: "2019-05-01", DateEnd: "2019-05-20"}

	if _, _, err := client.Audit.CreateLoginsReport(ctx, LoginAuditReportRequest{AuditReportRequest: base}); err == nil {
		t.Error("logins report without events should fail before sending")
	}
	if _, _, err := client.Audit.CreateFilesReport(ctx, FileAuditReportRequest{AuditReportRequest: base}); err == nil {
		t.Error("files report without folders/file should fail before sending")
	}
	if _, _, err := client.Audit.CreatePermissionsReport(ctx, PermissionsAuditReportRequest{}); err == nil {
		t.Error("report without base fields should fail before sending")
	}
}

func TestAuditService_CreateFilesReport(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/files", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		folders, _ := body["folders"].([]any)
		if len(folders) != 1 || folders[0] != "/Shared/Marketing" {
			t.Errorf("folders = %v", body["folders"])
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"id":"87654321"}`))
	})

	jobID, _, err := client.Audit.CreateFilesReport(context.Background(), FileAuditReportRequest{
		AuditReportRequest: AuditReportRequest{Format: "csv", DateStart: "2019-05-01", DateEnd: "2019-05-20"},
		Folders:            []string{"/Shared/Marketing"},
		TransactionType:    []string{"download", "upload"},
	})
	if err != nil {
		t.Fatalf("CreateFilesReport: %v", err)
	}
	if jobID != "87654321" {
		t.Errorf("jobID = %q", jobID)
	}
}

func TestAuditService_JobStatus_running(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/jobs/12345678", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"status":"running"}`))
	})

	status, _, err := client.Audit.JobStatus(context.Background(), "12345678")
	if err != nil {
		t.Fatalf("JobStatus: %v", err)
	}
	if status.Status != "running" || status.ReportURL != "" {
		t.Errorf("status = %+v", status)
	}
}

func TestAuditService_JobStatus_completedRedirect(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/jobs/12345678", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/pubapi/v1/audit/logins/12345678")
		w.WriteHeader(http.StatusSeeOther)
	})
	mux.HandleFunc("/pubapi/v1/audit/logins/12345678", func(w http.ResponseWriter, r *http.Request) {
		t.Error("JobStatus must not follow the 303 redirect")
	})

	status, resp, err := client.Audit.JobStatus(context.Background(), "12345678")
	if err != nil {
		t.Fatalf("JobStatus: %v", err)
	}
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("status code = %d, want 303", resp.StatusCode)
	}
	if status.Status != "completed" || status.ReportURL != "/pubapi/v1/audit/logins/12345678" {
		t.Errorf("status = %+v", status)
	}
}

func TestAuditService_GetReport(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/logins/12345678", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("offset") != "10" || q.Get("count") != "100" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"events":[{"user":"jsmith","action":"login","access_point":"Web"}],"totalCount":42}`))
	})

	report, _, err := client.Audit.GetReport(context.Background(), "logins", "12345678",
		&AuditReportPageOptions{Offset: 10, Count: 100})
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if report.TotalCount != 42 || len(report.Events) != 1 || report.Events[0]["user"] != "jsmith" {
		t.Errorf("report = %+v", report)
	}
}

func TestAuditService_DownloadReport_csv(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/files/87654321", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Header().Set("Content-Type", "text/csv")
		w.Write([]byte("user,action\njsmith,download\n"))
	})

	body, _, err := client.Audit.DownloadReport(context.Background(), "files", "87654321")
	if err != nil {
		t.Fatalf("DownloadReport: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "user,action\njsmith,download\n" {
		t.Errorf("csv = %q", data)
	}
}

func TestAuditService_DeleteReport(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/logins/12345678", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Audit.DeleteReport(context.Background(), "logins", "12345678"); err != nil {
		t.Fatalf("DeleteReport: %v", err)
	}
}

func TestAuditService_Stream(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/audit/stream", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["startDate"] != "2021-12-08" {
			t.Errorf("body = %v", body)
		}
		types, _ := body["auditType"].([]any)
		if len(types) != 2 {
			t.Errorf("auditType = %v", body["auditType"])
		}
		w.Write([]byte(`{"nextCursor":"QmlnVGFibGVLZXk=","moreEvents":true,
			"events":[{"date":1638921600000,"auditSource":"FILE_AUDIT",
			"sourcePath":"/Shared/document.pdf","action":"Download"}]}`))
	})

	stream, _, err := client.Audit.Stream(context.Background(), AuditStreamRequest{
		StartDate: "2021-12-08",
		EndDate:   "2021-12-10",
		AuditType: []string{"FILE_AUDIT", "LOGIN_AUDIT"},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if stream.NextCursor != "QmlnVGFibGVLZXk=" || !stream.MoreEvents || len(stream.Events) != 1 {
		t.Errorf("stream = %+v", stream)
	}
}

func TestAuditService_StreamGet(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/audit/stream", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("nextCursor") != "abc123" {
			t.Errorf("query = %v", q)
		}
		if q.Has("startDate") {
			t.Error("empty startDate should be omitted")
		}
		w.Write([]byte(`{"nextCursor":"def456","moreEvents":false,"events":[]}`))
	})

	stream, _, err := client.Audit.StreamGet(context.Background(), AuditStreamRequest{NextCursor: "abc123"})
	if err != nil {
		t.Fatalf("StreamGet: %v", err)
	}
	if stream.NextCursor != "def456" || stream.MoreEvents {
		t.Errorf("stream = %+v", stream)
	}
}

func TestAuditService_Stream_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.Audit.Stream(ctx, AuditStreamRequest{}); err == nil {
		t.Error("stream without startDate or nextCursor should fail before sending")
	}
	if _, _, err := client.Audit.Stream(ctx, AuditStreamRequest{StartDate: "2021-12-08", NextCursor: "abc"}); err == nil {
		t.Error("stream with both startDate and nextCursor should fail before sending")
	}
}

func TestAuditService_GetReport_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/audit/logins/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Report not found"}`))
	})

	_, _, err := client.Audit.GetReport(context.Background(), "logins", "999", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Message != "Report not found" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
