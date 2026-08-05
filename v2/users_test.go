package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestUsersService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/users", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("startIndex") != "1" || q.Get("count") != "10" || q.Get("filter") != `email eq "jmiller@example.com"` {
			t.Errorf("query = %v", q)
		}
		// deleteOnExpiry as a quoted string mirrors live API behavior.
		w.Write([]byte(`{"totalResults":1,"itemsPerPage":1,"startIndex":1,"resources":[
			{"id":12345678,"userName":"jmiller","email":"jmiller@example.com",
			 "name":{"givenName":"John","familyName":"Miller"},"active":true,
			 "authType":"sso","userType":"power","deleteOnExpiry":"false",
			 "groups":[{"displayName":"Marketing Team","value":"1ee2e04c"}]}]}`))
	})

	list, _, err := client.Users.List(context.Background(), &UserListOptions{
		StartIndex: 1,
		Count:      10,
		Filter:     `email eq "jmiller@example.com"`,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.TotalResults != 1 || len(list.Resources) != 1 {
		t.Fatalf("list = %+v", list)
	}
	u := list.Resources[0]
	if u.ID != 12345678 || u.Name.GivenName != "John" || len(u.Groups) != 1 {
		t.Errorf("user = %+v", u)
	}
	if u.DeleteOnExpiry == nil || bool(*u.DeleteOnExpiry) {
		t.Errorf("DeleteOnExpiry = %v, want false decoded from string", u.DeleteOnExpiry)
	}
}

func TestUsersService_Create(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/users", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["userName"] != "jmiller" || body["active"] != true ||
			body["authType"] != "sso" || body["userType"] != "power" || body["sendInvite"] != false {
			t.Errorf("body = %v", body)
		}
		name, _ := body["name"].(map[string]any)
		if name["givenName"] != "John" || name["familyName"] != "Miller" {
			t.Errorf("name = %v", name)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":12345678,"userName":"jmiller","email":"jmiller@example.com",
			"name":{"givenName":"John","familyName":"Miller","formatted":"John Miller"},
			"active":true,"locked":false,"authType":"sso","userType":"power"}`))
	})

	user, resp, err := client.Users.Create(context.Background(), CreateUserRequest{
		UserName:   "jmiller",
		Email:      "jmiller@example.com",
		Name:       UserName{GivenName: "John", FamilyName: "Miller"},
		Active:     true,
		SendInvite: Bool(false),
		AuthType:   "sso",
		UserType:   "power",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	if user.ID != 12345678 || user.Name.Formatted != "John Miller" {
		t.Errorf("user = %+v", user)
	}
}

func TestUsersService_Create_validation(t *testing.T) {
	client, _ := setup(t)
	_, _, err := client.Users.Create(context.Background(), CreateUserRequest{
		UserName: "jmiller",
		Email:    "j@example.com",
		Name:     UserName{GivenName: "John"}, // missing FamilyName
		AuthType: "sso",
		UserType: "power",
	})
	if err == nil {
		t.Error("missing familyName should fail before sending")
	}
}

func TestUsersService_Get(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/users/12345678", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"id":12345678,"userName":"jmiller","userType":"admin",
			"deleteOnExpiry":null,"lastActiveDate":null}`))
	})

	user, _, err := client.Users.Get(context.Background(), 12345678)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if user.UserName != "jmiller" || user.UserType != "admin" {
		t.Errorf("user = %+v", user)
	}
	if user.DeleteOnExpiry != nil {
		t.Errorf("DeleteOnExpiry = %v, want nil for null", *user.DeleteOnExpiry)
	}
}

func TestUsersService_Update(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/users/12345678", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["email"] != "john.miller@example.com" || body["userType"] != "admin" {
			t.Errorf("body = %v", body)
		}
		if _, ok := body["active"]; ok {
			t.Error("unset optional fields should be omitted")
		}
		w.Write([]byte(`{"id":12345678,"email":"john.miller@example.com","userType":"admin"}`))
	})

	user, _, err := client.Users.Update(context.Background(), 12345678, UpdateUserRequest{
		Email:    "john.miller@example.com",
		UserType: "admin",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if user.Email != "john.miller@example.com" {
		t.Errorf("user = %+v", user)
	}
}

func TestUsersService_Delete(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/users/12345678", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Users.Delete(context.Background(), 12345678); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestUsersService_Get_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/users/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"User not found"}`))
	})

	_, _, err := client.Users.Get(context.Background(), 999)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Message != "User not found" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
