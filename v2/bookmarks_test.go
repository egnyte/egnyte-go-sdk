package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestBookmarksService_CreateByPath(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["path"] != "/Shared/Documents/MyPictures" {
			t.Errorf("body = %v", body)
		}
		if _, ok := body["folder_id"]; ok {
			t.Error("folder_id should be absent when creating by path")
		}
		w.Write([]byte(`{"id":5468,"path":"/Shared/Documents/MyPictures",
			"folder_id":"3edcb54a","creation_date":"2016-06-02T11:36:01.000+0000"}`))
	})

	bookmark, _, err := client.Bookmarks.CreateByPath(context.Background(), "/Shared/Documents/MyPictures")
	if err != nil {
		t.Fatalf("CreateByPath: %v", err)
	}
	if bookmark.ID != 5468 || bookmark.FolderID != "3edcb54a" {
		t.Errorf("bookmark = %+v", bookmark)
	}
}

func TestBookmarksService_CreateByFolderID(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["folder_id"] != "3edcb54a" {
			t.Errorf("body = %v", body)
		}
		if _, ok := body["path"]; ok {
			t.Error("path should be absent when creating by folder_id")
		}
		w.Write([]byte(`{"id":5469,"folder_id":"3edcb54a"}`))
	})

	bookmark, _, err := client.Bookmarks.CreateByFolderID(context.Background(), "3edcb54a")
	if err != nil {
		t.Fatalf("CreateByFolderID: %v", err)
	}
	if bookmark.ID != 5469 {
		t.Errorf("bookmark = %+v", bookmark)
	}
}

func TestBookmarksService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.Bookmarks.CreateByPath(ctx, ""); err == nil {
		t.Error("empty path should fail before sending")
	}
	if _, _, err := client.Bookmarks.CreateByFolderID(ctx, ""); err == nil {
		t.Error("empty folder_id should fail before sending")
	}
}

func TestBookmarksService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("offset") != "5" || q.Get("count") != "10" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"offset":5,"count":2,"bookmarks":[
			{"id":14455,"path":"/Shared/My Folder/Pictures","folder_id":"0344c35b"},
			{"id":14453,"path":"/Shared/MyDocuments","folder_id":"ff004fae"}]}`))
	})

	list, _, err := client.Bookmarks.List(context.Background(), &BookmarkListOptions{Offset: 5, Count: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Bookmarks) != 2 || list.Bookmarks[0].ID != 14455 {
		t.Errorf("list = %+v", list)
	}
}

func TestBookmarksService_GetAndDelete(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/bookmarks/14455", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"id":14455,"path":"/Shared/My Folder/Pictures","folder_id":"0344c35b"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	bookmark, _, err := client.Bookmarks.Get(ctx, 14455)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if bookmark.Path != "/Shared/My Folder/Pictures" {
		t.Errorf("bookmark = %+v", bookmark)
	}
	if _, err := client.Bookmarks.Delete(ctx, 14455); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestBookmarksService_Get_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/bookmarks/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Bookmark not found"}`))
	})

	_, _, err := client.Bookmarks.Get(context.Background(), 999)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Message != "Bookmark not found" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
