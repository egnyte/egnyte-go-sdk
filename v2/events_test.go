package egnyte

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestEventsService_Cursor(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/events/cursor", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"latest_event_id":16342,"oldest_event_id":16321,
			"timestamp":"2025-05-28T11:41:12.000Z"}`))
	})

	cursor, _, err := client.Events.Cursor(context.Background())
	if err != nil {
		t.Fatalf("Cursor: %v", err)
	}
	if cursor.LatestEventID != 16342 || cursor.OldestEventID != 16321 {
		t.Errorf("cursor = %+v", cursor)
	}
}

func TestEventsService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/events", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("id") != "16341" || q.Get("folder") != "/Shared" ||
			q.Get("suppress") != "app" || q.Get("type") != "file_system|note" ||
			q.Get("count") != "10" || q.Get("reverse") != "1" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"latest_id":16342,"oldest_id":16342,"count":1,"events":[
			{"id":16342,"timestamp":"2025-05-28T11:41:12.000Z","action_source":"WebUI",
			 "actor":1,"type":"file_system","action":"copy",
			 "data":{"target_path":"/Shared/Documents/My Contract.docx",
			         "target_id":"89b1e9d9","source_path":"/Shared/Contracts/My Contract.docx",
			         "is_folder":false}}]}`))
	})

	list, _, err := client.Events.List(context.Background(), 16341, &EventListOptions{
		Folder:   "/Shared",
		Suppress: "app",
		Types:    []string{"file_system", "note"},
		Count:    10,
		Reverse:  true,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.LatestID != 16342 || len(list.Events) != 1 {
		t.Fatalf("list = %+v", list)
	}
	e := list.Events[0]
	if e.Action != "copy" || e.Data.TargetPath != "/Shared/Documents/My Contract.docx" || e.Data.IsFolder {
		t.Errorf("event = %+v", e)
	}
}

func TestEventsService_List_noContent(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	list, resp, err := client.Events.List(context.Background(), 16342, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
	if len(list.Events) != 0 {
		t.Errorf("events = %+v, want empty", list.Events)
	}
}

func TestEventsService_ListV2_permissionChange(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/events", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("id"); got != "16320" {
			t.Errorf("id = %q", got)
		}
		w.Write([]byte(`{"latest_id":16321,"oldest_id":16321,"count":1,"events":[
			{"id":16321,"timestamp":"2024-10-21T05:10:53.000Z","action_source":"WebUI",
			 "actor":1,"type":"permission_change","action":"permission_change",
			 "data":{"target_path":"/Shared/Permission Test/Viewer","is_folder":false},
			 "eventTypeSpecificAttributes":{
			   "changeGroupEvents":[],
			   "changePermissionEvent":{"changedPermissions":[
			     {"action":"ADD","entry":{"subject":"/user/6","priv":"READ"}},
			     {"action":"DELETE","entry":{"subject":"/user/6","priv":"NONE"}}]},
			   "changeContext":null,
			   "targetFolderId":"dd3b7523","groupChange":false}}]}`))
	})

	list, _, err := client.Events.ListV2(context.Background(), 16320, nil)
	if err != nil {
		t.Fatalf("ListV2: %v", err)
	}
	e := list.Events[0]
	if e.Type != "permission_change" || e.EventTypeSpecificAttributes == nil {
		t.Fatalf("event = %+v", e)
	}
	attrs := e.EventTypeSpecificAttributes
	if attrs.TargetFolderID != "dd3b7523" || attrs.GroupChange {
		t.Errorf("attributes = %+v", attrs)
	}
	changed := attrs.ChangePermissionEvent.ChangedPermissions
	if len(changed) != 2 || changed[0].Action != "ADD" || changed[0].Entry.Priv != "READ" {
		t.Errorf("changedPermissions = %+v", changed)
	}
}

func TestEventsService_List_invalidCursor(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Invalid cursor"}`))
	})

	_, _, err := client.Events.List(context.Background(), 1, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Message != "Invalid cursor" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
