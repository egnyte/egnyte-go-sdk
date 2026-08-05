package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestFileSystemService_Get_listContent(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/Shared/Documents", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("list_content") != "true" || q.Get("count") != "2" ||
			q.Get("sort_by") != "name" || q.Get("sort_direction") != "descending" {
			t.Errorf("query = %v", q)
		}
		if q.Has("offset") || q.Has("perms") || q.Has("key") {
			t.Errorf("zero-value options should be omitted, query = %v", q)
		}
		// Live shape: the folder itself and folder entries use
		// lastModified (epoch ms), file entries use last_modified
		// (RFC1123 string).
		w.Write([]byte(`{
			"name": "Documents", "path": "/Shared/Documents", "is_folder": true,
			"folder_id": "fid1", "offset": 0, "count": 2, "total_count": 4,
			"lastModified": 1785757715000,
			"folders": [{"name": "Projects", "path": "/Shared/Documents/Projects",
			             "is_folder": true, "lastModified": 1719999999000}],
			"files": [{"name": "Proposal.docx", "path": "/Shared/Documents/Proposal.docx",
			           "size": 12345, "checksum": "abc", "num_versions": 3, "locked": false,
			           "permission": "Owner", "uploaded_by": "jsmith", "uploaded": 1785757715559,
			           "last_modified": "Mon, 03 Aug 2026 11:48:35 GMT"}]
		}`))
	})

	item, _, err := client.FileSystem.Get(context.Background(), "/Shared/Documents", &FileSystemGetOptions{
		ListContent:   true,
		Count:         2,
		SortBy:        "name",
		SortDirection: "descending",
	})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !item.IsFolder || item.TotalCount != 4 || len(item.Folders) != 1 || len(item.Files) != 1 {
		t.Errorf("item = %+v", item)
	}
	if item.FolderLastModified != 1785757715000 {
		t.Errorf("FolderLastModified = %d", item.FolderLastModified)
	}
	if got := item.ModTime().UnixMilli(); got != 1785757715000 {
		t.Errorf("ModTime = %d ms", got)
	}
	if item.Folders[0].LastModified != 1719999999000 {
		t.Errorf("folder entry = %+v", item.Folders[0])
	}
	f := item.Files[0]
	if f.Size != 12345 || f.NumVersions != 3 ||
		f.LastModified != "Mon, 03 Aug 2026 11:48:35 GMT" ||
		f.UploadedBy != "jsmith" || f.Permission != "Owner" {
		t.Errorf("file entry = %+v", f)
	}
}

func TestFileSystemService_Get_fileMetadata(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/Shared/fixture.txt", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		// Captured from the live API (include_locks, include_perm,
		// list_custom_metadata) with a locked file.
		w.Write([]byte(`{"checksum":"d134","size":99,"path":"/Shared/fixture.txt",
			"name":"fixture.txt",
			"versions":[{"checksum":"2579","size":104,"is_folder":false,
				"entry_id":"3b473403","uploaded":1785757714418,
				"last_modified":"Mon, 03 Aug 2026 11:48:34 GMT",
				"uploaded_by":"sdk","custom_metadata":[]}],
			"locked":true,"permission":"Owner","is_folder":false,
			"entry_id":"259931c0","group_id":"a6e4aac5","uploaded":1785757715559,
			"last_modified":"Mon, 03 Aug 2026 11:48:35 GMT","uploaded_by":"sdk",
			"custom_metadata":[],"num_versions":2,"parent_id":"ec5dd2a5",
			"lock_info":{"owner_id":12,"first_name":"sdk","last_name":"sdk",
				"email":"user@example.com"}}`))
	})

	item, _, err := client.FileSystem.Get(context.Background(), "/Shared/fixture.txt",
		&FileSystemGetOptions{IncludeLocks: true, IncludePerm: true, ListCustomMetadata: true})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.IsFolder || item.LastModified != "Mon, 03 Aug 2026 11:48:35 GMT" ||
		item.UploadedBy != "sdk" || item.Permission != "Owner" || !item.Locked {
		t.Errorf("item = %+v", item)
	}
	if item.LockInfo == nil || item.LockInfo.OwnerID != 12 || item.LockInfo.FirstName != "sdk" {
		t.Errorf("lock info = %+v", item.LockInfo)
	}
	if len(item.Versions) != 1 {
		t.Fatalf("versions = %+v", item.Versions)
	}
	v := item.Versions[0]
	if v.EntryID != "3b473403" || v.Size != 104 ||
		v.LastModified != "Mon, 03 Aug 2026 11:48:34 GMT" || v.UploadedBy != "sdk" {
		t.Errorf("version = %+v", v)
	}
	// RFC1123 file timestamps parse through ModTime.
	if got := item.ModTime().UTC().Format("2006-01-02 15:04:05"); got != "2026-08-03 11:48:35" {
		t.Errorf("ModTime = %s", got)
	}
}

func TestFileSystemService_GetByID(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/ids/file/a6e4aac5", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"name":"fixture.txt","is_folder":false,"group_id":"a6e4aac5",
			"last_modified":"Mon, 03 Aug 2026 11:48:35 GMT"}`))
	})
	mux.HandleFunc("/pubapi/v1/fs/ids/folder/ec5dd2a5", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("list_content"); got != "true" {
			t.Errorf("list_content = %q", got)
		}
		w.Write([]byte(`{"name":"scratch","is_folder":true,"folder_id":"ec5dd2a5",
			"lastModified":1785757715000}`))
	})

	ctx := context.Background()
	file, _, err := client.FileSystem.GetFileByID(ctx, "a6e4aac5", nil)
	if err != nil {
		t.Fatalf("GetFileByID: %v", err)
	}
	if file.GroupID != "a6e4aac5" || file.IsFolder {
		t.Errorf("file = %+v", file)
	}
	folder, _, err := client.FileSystem.GetFolderByID(ctx, "ec5dd2a5", &FileSystemGetOptions{ListContent: true})
	if err != nil {
		t.Fatalf("GetFolderByID: %v", err)
	}
	if folder.FolderID != "ec5dd2a5" || !folder.IsFolder || folder.FolderLastModified != 1785757715000 {
		t.Errorf("folder = %+v", folder)
	}
}

func TestFileSystemService_FolderStatsByID(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/ids/folder/ec5dd2a5/stats", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		// Captured from the live API.
		w.Write([]byte(`{"allVersionsSize":203,"allFilesSize":99,"filesCount":1,
			"fileVersionsCount":2,"foldersCount":1,"allFilesSizeInKB":0,"allVersionsSizeInKB":0}`))
	})

	stats, _, err := client.FileSystem.FolderStatsByID(context.Background(), "ec5dd2a5")
	if err != nil {
		t.Fatalf("FolderStatsByID: %v", err)
	}
	want := FolderStats{AllVersionsSize: 203, AllFilesSize: 99, FilesCount: 1,
		FileVersionsCount: 2, FoldersCount: 1}
	if *stats != want {
		t.Errorf("stats = %+v, want %+v", *stats, want)
	}
}

func TestFileSystemService_LockUnlock(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/Shared/lockme.txt", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		body := decodeAction(t, r)
		switch body["action"] {
		case "lock":
			w.Write([]byte(`{"lock_token":"7cf131ee","timeout":3599}`))
		case "unlock":
			if body["lock_token"] != "7cf131ee" {
				t.Errorf("unlock body = %v", body)
			}
		default:
			t.Errorf("action = %v", body["action"])
		}
	})

	ctx := context.Background()
	lock, _, err := client.FileSystem.Lock(ctx, "/Shared/lockme.txt")
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if lock.LockToken != "7cf131ee" || lock.Timeout != 3599 {
		t.Errorf("lock = %+v", lock)
	}
	if _, err := client.FileSystem.Unlock(ctx, "/Shared/lockme.txt", lock.LockToken); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if _, err := client.FileSystem.Unlock(ctx, "/Shared/lockme.txt", ""); err == nil {
		t.Error("empty lock token should fail before sending")
	}
}

func TestFileSystemService_Get_encodesPath(t *testing.T) {
	client, mux := setup(t)
	var gotPath string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Write([]byte(`{}`))
	})

	_, _, err := client.FileSystem.Get(context.Background(), "/Shared/example?path/$file.txt", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := "/pubapi/v1/fs/Shared/example%3Fpath/%24file.txt"
	if gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
}

func TestFileSystemService_GetV2(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/fs/Shared", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"name":"Shared","path":"/Shared","is_folder":true}`))
	})

	item, _, err := client.FileSystem.GetV2(context.Background(), "/Shared", nil)
	if err != nil {
		t.Fatalf("GetV2: %v", err)
	}
	if item.Name != "Shared" {
		t.Errorf("item = %+v", item)
	}
}

func decodeAction(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestFileSystemService_CreateFolder(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/Shared/NewFolder", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testHeader(t, r, "Content-Type", "application/json")
		body := decodeAction(t, r)
		if body["action"] != "add_folder" {
			t.Errorf("body = %v", body)
		}
		if _, ok := body["destination"]; ok {
			t.Error("empty destination should be omitted")
		}
		w.WriteHeader(http.StatusCreated)
	})

	if _, err := client.FileSystem.CreateFolder(context.Background(), "/Shared/NewFolder"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
}

func TestFileSystemService_MoveCopyRename(t *testing.T) {
	client, mux := setup(t)
	var got map[string]any
	mux.HandleFunc("/pubapi/v1/fs/Shared/a.txt", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		got = decodeAction(t, r)
	})

	ctx := context.Background()
	// Destination is the full target path incl. the (new) name.
	if _, err := client.FileSystem.Move(ctx, "/Shared/a.txt", "/Shared/Archive/b.txt"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got["action"] != "move" || got["destination"] != "/Shared/Archive/b.txt" {
		t.Errorf("move body = %v", got)
	}

	if _, err := client.FileSystem.Copy(ctx, "/Shared/a.txt", "/Shared/Backup/a.txt"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if got["action"] != "copy" || got["destination"] != "/Shared/Backup/a.txt" {
		t.Errorf("copy body = %v", got)
	}

	// Rename is a move within the same parent folder.
	if _, err := client.FileSystem.Rename(ctx, "/Shared/a.txt", "c.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if got["action"] != "move" || got["destination"] != "/Shared/c.txt" {
		t.Errorf("rename body = %v", got)
	}
	if _, ok := got["new_name"]; ok {
		t.Error("new_name must never be sent — the live API rejects it")
	}
}

func TestFileSystemService_Action_requiresAction(t *testing.T) {
	client, _ := setup(t)
	if _, err := client.FileSystem.Action(context.Background(), "/Shared/x", FileSystemAction{}); err == nil {
		t.Error("Action with empty action should fail before sending")
	}
}

func TestFileSystemService_ActionV2(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/fs/Shared/x", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		body := decodeAction(t, r)
		if body["action"] != "move" || body["permissions"] == nil {
			t.Errorf("body = %v", body)
		}
	})

	_, err := client.FileSystem.ActionV2(context.Background(), "/Shared/x", FileSystemAction{
		Action:      "move",
		Destination: "/Shared/y",
		Permissions: map[string]string{"mode": "keep_original"},
	})
	if err != nil {
		t.Fatalf("ActionV2: %v", err)
	}
}

func TestFileSystemService_Delete(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/fs/Shared/old.txt", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
		if got := r.URL.Query().Get("entry_id"); got != "ver123" {
			t.Errorf("entry_id = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.FileSystem.Delete(context.Background(), "/Shared/old.txt", &FileSystemDeleteOptions{EntryID: "ver123"})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestFileSystemService_DeleteV2_conflictError(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v2/fs/Shared/dir", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodDelete)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Folder is not empty"}`))
	})

	_, err := client.FileSystem.DeleteV2(context.Background(), "/Shared/dir", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Message != "Folder is not empty" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
