package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestCommentsService_Add(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/notes", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["path"] != "/Shared/Documents/form.docx" || body["body"] != "Test comment" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"id":"4fbba2ef","message":"Test comment","username":"asmith",
			"can_delete":true,"creation_time":"2016-05-18T17:56:39.975+0000",
			"formatted_name":"Anna Smith","file_path":"/Shared/Documents/form.docx",
			"group_id":"628ceb16"}`))
	})

	comment, _, err := client.Comments.Add(context.Background(), "/Shared/Documents/form.docx", "Test comment")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if comment.ID != "4fbba2ef" || comment.FormattedName != "Anna Smith" || !comment.CanDelete {
		t.Errorf("comment = %+v", comment)
	}
}

func TestCommentsService_Add_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.Comments.Add(ctx, "", "text"); err == nil {
		t.Error("missing path should fail before sending")
	}
	if _, _, err := client.Comments.Add(ctx, "/Shared/x.txt", ""); err == nil {
		t.Error("missing body should fail before sending")
	}
}

func TestCommentsService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/notes", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("file") != "/Shared/Documents/testdoc.docx" || q.Get("count") != "50" ||
			q.Get("start_time") != "2016-05-01T00:00:00Z" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"total_results":2,"count":1,"offset":0,"notes":[
			{"id":"4fbba2ef","message":"Test comment","username":"asmith"}]}`))
	})

	list, _, err := client.Comments.List(context.Background(), &CommentListOptions{
		File:      "/Shared/Documents/testdoc.docx",
		StartTime: "2016-05-01T00:00:00Z",
		Count:     50,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.TotalResults != 2 || len(list.Notes) != 1 || list.Notes[0].Username != "asmith" {
		t.Errorf("list = %+v", list)
	}
}

func TestCommentsService_GetAndDelete(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/notes/4fbba2ef", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"id":"4fbba2ef","message":"Test comment 2","username":"asmith"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	comment, _, err := client.Comments.Get(ctx, "4fbba2ef")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if comment.Message != "Test comment 2" {
		t.Errorf("comment = %+v", comment)
	}
	if _, err := client.Comments.Delete(ctx, "4fbba2ef"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestCommentsService_Get_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/notes/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Comment not found"}`))
	})

	_, _, err := client.Comments.Get(context.Background(), "nope")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError = %+v", apiErr)
	}
}
