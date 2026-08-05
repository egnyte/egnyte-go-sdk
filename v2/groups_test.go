package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestGroupsService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/groups", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("filter") != `displayName sw "Mar"` || q.Get("count") != "50" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"totalResults":1,"itemsPerPage":1,"startIndex":1,"resources":[
			{"id":"c8aa533b","displayName":"Marketing Team",
			 "members":[{"value":"17d2ea40","display":"alice"}]}]}`))
	})

	list, _, err := client.Groups.List(context.Background(), &GroupListOptions{
		Count:  50,
		Filter: `displayName sw "Mar"`,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.TotalResults != 1 || len(list.Resources) != 1 {
		t.Fatalf("list = %+v", list)
	}
	g := list.Resources[0]
	if g.DisplayName != "Marketing Team" || len(g.Members) != 1 || g.Members[0].Display != "alice" {
		t.Errorf("group = %+v", g)
	}
}

func TestGroupsService_Create(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/groups", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["displayName"] != "Project Contributors" {
			t.Errorf("body = %v", body)
		}
		members, _ := body["members"].([]any)
		if len(members) != 2 {
			t.Errorf("members = %v", body["members"])
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"5451f218","displayName":"Project Contributors","members":[]}`))
	})

	group, resp, err := client.Groups.Create(context.Background(), "Project Contributors",
		[]string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || group.ID != "5451f218" {
		t.Errorf("group = %+v, status = %d", group, resp.StatusCode)
	}
}

func TestGroupsService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Groups.Create(context.Background(), "", nil); err == nil {
		t.Error("missing displayName should fail before sending")
	}
	if _, _, err := client.Groups.Update(context.Background(), "gid", "", nil); err == nil {
		t.Error("Update with missing displayName should fail before sending")
	}
	if _, _, err := client.Groups.Patch(context.Background(), "gid", nil); err == nil {
		t.Error("Patch with no operations should fail before sending")
	}
}

func TestGroupsService_Get(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/groups/38d6b4c0", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"id":"38d6b4c0","displayName":"Engineering",
			"members":[{"value":"17d2ea40","display":"alice"}]}`))
	})

	group, _, err := client.Groups.Get(context.Background(), "38d6b4c0")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if group.DisplayName != "Engineering" || len(group.Members) != 1 {
		t.Errorf("group = %+v", group)
	}
}

func TestGroupsService_Update(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/groups/38d6b4c0", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPut)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["displayName"] != "Engineering Team" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"id":"38d6b4c0","displayName":"Engineering Team",
			"members":[{"value":"17d2ea40","display":"alice"}]}`))
	})

	group, _, err := client.Groups.Update(context.Background(), "38d6b4c0", "Engineering Team",
		[]string{"17d2ea40"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if group.DisplayName != "Engineering Team" {
		t.Errorf("group = %+v", group)
	}
}

func TestGroupsService_Patch(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/groups/38d6b4c0", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		var body struct {
			Schemas    []string              `json:"schemas"`
			Operations []GroupPatchOperation `json:"Operations"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.Schemas) != 1 || body.Schemas[0] != scimPatchOpSchema {
			t.Errorf("schemas = %v", body.Schemas)
		}
		if len(body.Operations) != 1 || body.Operations[0].Op != "replace" ||
			body.Operations[0].Path != "displayName" {
			t.Errorf("operations = %+v", body.Operations)
		}
		w.Write([]byte(`{"id":"38d6b4c0","displayName":"Engineering"}`))
	})

	group, _, err := client.Groups.Patch(context.Background(), "38d6b4c0", []GroupPatchOperation{
		{Op: "replace", Path: "displayName", Value: "Engineering"},
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if group.DisplayName != "Engineering" {
		t.Errorf("group = %+v", group)
	}
}

func TestGroupsService_Delete(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/groups/38d6b4c0", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Groups.Delete(context.Background(), "38d6b4c0"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestGroupsService_Create_conflict(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/groups", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message":"Group already exists"}`))
	})

	_, _, err := client.Groups.Create(context.Background(), "Marketing Team", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Message != "Group already exists" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
