package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestTrashService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/trash", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("sort_by") != "name" || q.Get("count") != "25" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"count":1,"offset":0,"total_count":2,"items":[
			{"id":"trash1","type":"file","path":"/Private/asmith/photo.jpg",
			 "name":"photo.jpg","size":1733,"deleted_by":"Ashley Smith",
			 "delete_date":"2016-04-18T16:11:38Z","purge_date":"2016-05-19T00:00:00Z"}]}`))
	})

	list, _, err := client.Trash.List(context.Background(), &TrashListOptions{SortBy: "name", Count: 25})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.TotalCount != 2 || len(list.Items) != 1 || list.Items[0].DeletedBy != "Ashley Smith" {
		t.Errorf("list = %+v", list)
	}
}

func TestTrashService_ListV2(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/fs/trash", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("deletedBy") != "Ashley" || q.Get("startDate") != "2016-04-01T00:00:00Z" ||
			q.Get("endDate") != "2016-05-01T00:00:00Z" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"count":1,"offset":0,"has_more":true,"items":[
			{"id":"trash1","type":"file","name":"photo.jpg",
			 "owner":{"id":9,"username":"asmith","email":"a@x.com"},
			 "data_retention_info":{"retention":"GRACE_PERIOD","is_retained":true}}]}`))
	})

	list, _, err := client.Trash.ListV2(context.Background(), &TrashListV2Options{
		DeletedBy: "Ashley",
		StartDate: "2016-04-01T00:00:00Z",
		EndDate:   "2016-05-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("ListV2: %v", err)
	}
	if !list.HasMore || len(list.Items) != 1 {
		t.Fatalf("list = %+v", list)
	}
	item := list.Items[0]
	if item.Owner == nil || item.Owner.Username != "asmith" {
		t.Errorf("owner = %+v", item.Owner)
	}
	if item.DataRetentionInfo == nil || !item.DataRetentionInfo.IsRetained {
		t.Errorf("retention = %+v", item.DataRetentionInfo)
	}
}

func TestTrashService_TotalCount(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/fs/trash/totalcount", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"totalCount":42}`))
	})

	count, _, err := client.Trash.TotalCount(context.Background())
	if err != nil {
		t.Fatalf("TotalCount: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d", count)
	}
}

func TestTrashService_Restore(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/trash", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		ids, _ := body["ids"].([]any)
		if body["action"] != "RESTORE" || len(ids) != 1 {
			t.Errorf("body = %v", body)
		}
	})

	result, _, err := client.Trash.Restore(context.Background(), []string{"trash1"})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if len(result.Resources) != 0 {
		t.Errorf("resources = %+v, want empty on full success", result.Resources)
	}
}

func TestTrashService_Purge_multiStatus(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/trash", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["action"] != "PURGE" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusMultiStatus)
		w.Write([]byte(`{"resources":[
			{"id":"trash1","code":"200"},
			{"id":"trash2","code":"404","descriptions":"Item is not in the trash"}]}`))
	})

	result, resp, err := client.Trash.Purge(context.Background(), []string{"trash1", "trash2"})
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("status = %d, want 207", resp.StatusCode)
	}
	if len(result.Resources) != 2 || result.Resources[1].Code != "404" ||
		result.Resources[1].Descriptions != "Item is not in the trash" {
		t.Errorf("resources = %+v", result.Resources)
	}
}

func TestTrashService_action_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.Trash.Restore(ctx, nil); err == nil {
		t.Error("restore with no ids should fail before sending")
	}
	ids := make([]string, 11)
	for i := range ids {
		ids[i] = "id"
	}
	if _, _, err := client.Trash.Purge(ctx, ids); err == nil {
		t.Error("purge with more than 10 ids should fail before sending")
	}
}

func TestTrashService_List_forbidden(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/trash", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"statusCode":403,"errorMessage":"Trash access requires admin privileges"}`))
	})

	_, _, err := client.Trash.List(context.Background(), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError = %+v", apiErr)
	}
}
