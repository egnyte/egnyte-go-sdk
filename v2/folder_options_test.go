package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestFileSystemService_SetFolderOptions(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/Shared/Contracts", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["public_links"] != "files_folders" ||
			body["move_delete_folder_restriction"] != "admins_or_owners" ||
			body["allow_upload_links"] != true ||
			body["inheritance_rule"] != "this_folder_and_subfolders_until_overridden" {
			t.Errorf("body = %v", body)
		}
		prefs, _ := body["email_preferences"].(map[string]any)
		if prefs["content_updates"] != true {
			t.Errorf("email_preferences = %v", body["email_preferences"])
		}
		if _, ok := body["allow_links"]; ok {
			t.Error("unset options should be omitted")
		}
		w.Write([]byte(`{"name":"Contracts","path":"/Shared/Contracts","is_folder":true,
			"folder_id":"b6f42a4b","folder_description":"Construction contracts",
			"allow_links":true,"allow_upload_links":true,
			"allowed_file_link_types":["domain","anyone"],
			"public_links":"files_folders","restrict_move_delete":true,
			"move_delete_folder_restriction":"admins_or_owners"}`))
	})

	item, _, err := client.FileSystem.SetFolderOptions(context.Background(), "/Shared/Contracts", FolderOptions{
		FolderDescription:           "Construction contracts",
		PublicLinks:                 "files_folders",
		MoveDeleteFolderRestriction: "admins_or_owners",
		AllowUploadLinks:            Bool(true),
		InheritanceRule:             "this_folder_and_subfolders_until_overridden",
		EmailPreferences:            &FolderEmailPreferences{ContentUpdates: Bool(true)},
	})
	if err != nil {
		t.Fatalf("SetFolderOptions: %v", err)
	}
	if item.PublicLinks != "files_folders" || item.MoveDeleteFolderRestriction != "admins_or_owners" {
		t.Errorf("item = %+v", item)
	}
	if item.AllowLinks == nil || !*item.AllowLinks || len(item.AllowedFileLinkTypes) != 2 {
		t.Errorf("link fields = %+v", item)
	}
}

func TestFileSystemService_SetFolderOptions_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.FileSystem.SetFolderOptions(context.Background(), "/Shared/x", FolderOptions{}); err == nil {
		t.Error("empty options should fail before sending")
	}
}

func TestFileSystemService_SetFolderOptions_topLevelError(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/Shared", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"statusCode":400,"errorMessage":"Cannot restrict top-level folders"}`))
	})

	_, _, err := client.FileSystem.SetFolderOptions(context.Background(), "/Shared", FolderOptions{
		RestrictMoveDelete: Bool(true),
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("APIError = %+v", apiErr)
	}
}
