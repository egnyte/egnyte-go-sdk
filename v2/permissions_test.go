package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestPermissionsService_GetEffective(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/perms/user/jsmith", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("folder"); got != "/Shared/Documents" {
			t.Errorf("folder = %q", got)
		}
		w.Write([]byte(`{"permission":"Editor"}`))
	})

	perm, _, err := client.Permissions.GetEffective(context.Background(), "jsmith", "/Shared/Documents")
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if perm != PermissionEditor {
		t.Errorf("permission = %q, want Editor", perm)
	}
}

func TestPermissionsService_GetEffective_ownPermission(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/perms/user", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("folder"); got != "/Shared/Documents" {
			t.Errorf("folder = %q", got)
		}
		w.Write([]byte(`{"permission":"Owner"}`))
	})

	// Empty username asks for the calling user's own permission.
	perm, _, err := client.Permissions.GetEffective(context.Background(), "", "/Shared/Documents")
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if perm != PermissionOwner {
		t.Errorf("permission = %q, want Owner", perm)
	}
}

func TestPermissionsService_GetEffective_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Permissions.GetEffective(context.Background(), "jsmith", ""); err == nil {
		t.Error("missing folder should fail before sending")
	}
}

func TestPermissionsService_Get(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/perms/Shared/Documents", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"userPerms":{"jsmith":"Full","ajones":"Viewer"},
			"groupPerms":{"Marketing":"Editor"},"inheritsPermissions":true}`))
	})

	perms, _, err := client.Permissions.Get(context.Background(), "/Shared/Documents")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if perms.UserPerms["jsmith"] != PermissionFull || perms.GroupPerms["Marketing"] != PermissionEditor {
		t.Errorf("perms = %+v", perms)
	}
	if !perms.InheritsPermissions {
		t.Error("InheritsPermissions = false, want true")
	}
}

func TestPermissionsService_Set(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/perms/Shared/Documents", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		userPerms, _ := body["userPerms"].(map[string]any)
		if userPerms["jsmith"] != "None" {
			t.Errorf("userPerms = %v", userPerms)
		}
		if body["inheritsPermissions"] != false || body["keepParentPermissions"] != true {
			t.Errorf("body = %v", body)
		}
	})

	_, err := client.Permissions.Set(context.Background(), "/Shared/Documents", SetFolderPermissions{
		UserPerms:             map[string]PermissionLevel{"jsmith": PermissionNone},
		InheritsPermissions:   Bool(false),
		KeepParentPermissions: Bool(true),
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
}

func TestPermissionsService_GetV1(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/perms/folder/Shared/MyFolder", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("users") != "jsmith|ajones" || q.Get("groups") != "All Power Users|Marketing" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"users":[{"subject":"jsmith","permission":"Owner"}],
			"groups":[{"subject":"Marketing","permission":"Viewer Only"}]}`))
	})

	perms, _, err := client.Permissions.GetV1(context.Background(), "/Shared/MyFolder",
		[]string{"jsmith", "ajones"}, []string{"All Power Users", "Marketing"})
	if err != nil {
		t.Fatalf("GetV1: %v", err)
	}
	if len(perms.Users) != 1 || perms.Users[0].Permission != PermissionOwner {
		t.Errorf("users = %+v", perms.Users)
	}
	if len(perms.Groups) != 1 || perms.Groups[0].Permission != PermissionViewerOnly {
		t.Errorf("groups = %+v", perms.Groups)
	}
}

func TestPermissionsService_SetV1(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/perms/folder/Shared/MyFolder", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["permission"] != "Viewer" {
			t.Errorf("permission = %v", body["permission"])
		}
		if _, ok := body["groups"]; ok {
			t.Error("empty groups should be omitted")
		}
	})

	_, err := client.Permissions.SetV1(context.Background(), "/Shared/MyFolder",
		[]string{"jsmith"}, nil, PermissionViewer)
	if err != nil {
		t.Fatalf("SetV1: %v", err)
	}
}

func TestPermissionsService_V1_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Permissions.GetV1(context.Background(), "/Shared", nil, nil); err == nil {
		t.Error("GetV1 with no users/groups should fail before sending")
	}
	if _, err := client.Permissions.SetV1(context.Background(), "/Shared", nil, nil, PermissionFull); err == nil {
		t.Error("SetV1 with no users/groups should fail before sending")
	}
}

func TestPermissionsService_Get_forbidden(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/perms/Private/x", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Insufficient permissions"}`))
	})

	_, _, err := client.Permissions.Get(context.Background(), "/Private/x")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Message != "Insufficient permissions" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
