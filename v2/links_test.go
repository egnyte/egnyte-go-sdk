package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestLinksService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/links", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("path") != "/Shared/Report.pdf" || q.Get("type") != "file" ||
			q.Get("accessibility") != "domain" || q.Get("offset") != "10" || q.Get("count") != "50" {
			t.Errorf("query = %v", q)
		}
		if q.Has("username") || q.Has("created_before") {
			t.Errorf("zero-value filters should be omitted, query = %v", q)
		}
		w.Write([]byte(`{"ids":["owTMm8H8Sg","KsiryUUgEo"],"offset":10,"count":2,"total_count":5}`))
	})

	list, _, err := client.Links.List(context.Background(), &LinkListOptions{
		Path:          "/Shared/Report.pdf",
		Type:          "file",
		Accessibility: "domain",
		Offset:        10,
		Count:         50,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.IDs) != 2 || list.IDs[0] != "owTMm8H8Sg" || list.TotalCount != 5 {
		t.Errorf("list = %+v", list)
	}
}

func TestLinksService_ListV2(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/links", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"links":[{"id":"abc","url":"https://test.egnyte.com/dl/abc",
			"path":"/Shared/x.txt","type":"file","accessibility":"anyone",
			"created_by":"jane.doe","expiry_clicks":3}],"count":1}`))
	})

	list, _, err := client.Links.ListV2(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListV2: %v", err)
	}
	if list.Count != 1 || len(list.Links) != 1 {
		t.Fatalf("list = %+v", list)
	}
	l := list.Links[0]
	if l.ID != "abc" || l.CreatedBy != "jane.doe" || l.ExpiryClicks != 3 {
		t.Errorf("link = %+v", l)
	}
}

func TestLinksService_Create(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/links", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "Content-Type", "application/json")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["path"] != "/Shared/Report.pdf" || body["type"] != "file" ||
			body["accessibility"] != "recipients" || body["notify"] != false {
			t.Errorf("body = %v", body)
		}
		if _, ok := body["useDefaultSettings"]; ok {
			t.Error("unset optional bools should be omitted")
		}
		if recips, ok := body["recipients"].([]any); !ok || len(recips) != 1 {
			t.Errorf("recipients = %v", body["recipients"])
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"links":[{"id":"5a123b4cde","url":"https://test.egnyte.com/dl/5a123b4cde",
			"recipients":["a@b.com"]}],"path":"/Shared/Report.pdf","type":"file",
			"accessibility":"recipients","created_by":"jane.doe"}`))
	})

	result, resp, err := client.Links.Create(context.Background(), CreateLinkRequest{
		Path:          "/Shared/Report.pdf",
		Type:          "file",
		Accessibility: "recipients",
		Recipients:    []string{"a@b.com"},
		Notify:        Bool(false),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if len(result.Links) != 1 || result.Links[0].ID != "5a123b4cde" || result.CreatedBy != "jane.doe" {
		t.Errorf("result = %+v", result)
	}
}

func TestLinksService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Links.Create(context.Background(), CreateLinkRequest{Type: "file"}); err == nil {
		t.Error("missing path should fail before sending")
	}
	if _, _, err := client.Links.CreateV2(context.Background(), CreateLinkRequest{Path: "/x"}); err == nil {
		t.Error("missing type should fail before sending")
	}
}

func TestLinksService_CreateV2(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/links", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"links":[{"id":"v2id","url":"https://test.egnyte.com/fl/v2id"}],"type":"folder"}`))
	})

	result, _, err := client.Links.CreateV2(context.Background(), CreateLinkRequest{
		Path: "/Shared/Projects/Alpha",
		Type: "folder",
	})
	if err != nil {
		t.Fatalf("CreateV2: %v", err)
	}
	if len(result.Links) != 1 || result.Links[0].ID != "v2id" {
		t.Errorf("result = %+v", result)
	}
}

func TestLinksService_Get(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/links/5a123b4cde", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"id":"5a123b4cde","path":"/Shared/Report.pdf","type":"file",
			"url":"https://test.egnyte.com/dl/5a123b4cde","protection":"NONE",
			"notify":true,"recipients":[]}`))
	})

	link, _, err := client.Links.Get(context.Background(), "5a123b4cde")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if link.ID != "5a123b4cde" || link.Protection != "NONE" || !link.Notify {
		t.Errorf("link = %+v", link)
	}
}

func TestLinksService_GetV2_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/links/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"Errors":[{"description":"Link does not exist.","code":"404"}]}`))
	})

	_, _, err := client.Links.GetV2(context.Background(), "nope")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || len(apiErr.Errors) != 1 {
		t.Errorf("APIError = %+v", apiErr)
	}
}

func TestLinksService_List_errorReturnsNil(t *testing.T) {
	client, mux := setup(t)
	for _, version := range []string{"v1", "v2"} {
		mux.HandleFunc("/pubapi/"+version+"/links", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message":"Insufficient permissions"}`))
		})
	}

	list, _, err := client.Links.List(context.Background(), nil)
	if err == nil {
		t.Fatal("List should fail")
	}
	if list != nil {
		t.Errorf("List on error = %+v, want nil", list)
	}
	listV2, _, err := client.Links.ListV2(context.Background(), nil)
	if err == nil {
		t.Fatal("ListV2 should fail")
	}
	if listV2 != nil {
		t.Errorf("ListV2 on error = %+v, want nil", listV2)
	}
}

func TestLinksService_Delete(t *testing.T) {
	client, mux := setup(t)
	for _, version := range []string{"v1", "v2"} {
		mux.HandleFunc("/pubapi/"+version+"/links/del1", func(w http.ResponseWriter, r *http.Request) {
			testMethod(t, r, http.MethodDelete)
			w.WriteHeader(http.StatusNoContent)
		})
	}

	if _, err := client.Links.Delete(context.Background(), "del1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := client.Links.DeleteV2(context.Background(), "del1"); err != nil {
		t.Fatalf("DeleteV2: %v", err)
	}
}
