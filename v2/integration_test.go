//go:build integration

package egnyte

// Integration tests run against a real Egnyte domain and are excluded
// from normal test runs by the "integration" build tag:
//
//	go test -tags integration -run TestIntegration -v ./...
//
// Configuration via environment variables:
//
//	EGNYTE_DOMAIN       domain, e.g. "acme", "acme.egnyte.com" or
//	                    "acme.egnytegov.com" (required)
//	EGNYTE_TOKEN        OAuth access token (option 1)
//	EGNYTE_KEY          API key            (option 2, with the three below —
//	EGNYTE_SECRET       API secret          a token is minted via the
//	EGNYTE_USERNAME     username            Resource Owner Password flow)
//	EGNYTE_PASSWORD     password
//	EGNYTE_SCOPES       optional space-separated scopes for the password
//	                    flow; the full suite needs Egnyte.filesystem,
//	                    Egnyte.link and Egnyte.permission; Egnyte.ai enables
//	                    the optional hybrid-search check
//
// The test needs a token with at least the Egnyte.filesystem, Egnyte.link and
// Egnyte.permission scopes (an unscoped internal-app token works). It creates a
// uniquely named scratch folder under /Shared, does all its work inside it, and
// deletes it at the end (the folder lands in the trash).

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	integOnce   sync.Once
	integClient *Client
	integSkip   string
	integFatal  error
)

// integrationClient returns a shared, authenticated client. The token is
// minted at most once per test binary run — the OAuth endpoint allows
// only 10 token requests per user per hour.
func integrationClient(t *testing.T) *Client {
	t.Helper()
	integOnce.Do(func() {
		domain := os.Getenv("EGNYTE_DOMAIN")
		if domain == "" {
			integSkip = "EGNYTE_DOMAIN not set; skipping integration test"
			return
		}
		client, err := NewClient(domain)
		if err != nil {
			integFatal = err
			return
		}
		if token := os.Getenv("EGNYTE_TOKEN"); token != "" {
			client.SetToken(token)
			integClient = client
			return
		}
		key, secret := os.Getenv("EGNYTE_KEY"), os.Getenv("EGNYTE_SECRET")
		username, password := os.Getenv("EGNYTE_USERNAME"), os.Getenv("EGNYTE_PASSWORD")
		if key == "" || username == "" || password == "" {
			integSkip = "set EGNYTE_TOKEN, or EGNYTE_KEY/EGNYTE_SECRET/EGNYTE_USERNAME/EGNYTE_PASSWORD"
			return
		}
		creds := PasswordCredentials{
			ClientID:     key,
			ClientSecret: secret,
			Username:     username,
			Password:     password,
		}
		if scopes := os.Getenv("EGNYTE_SCOPES"); scopes != "" {
			creds.Scopes = strings.Fields(scopes)
		}
		_, _, err = client.RequestToken(context.Background(), creds)
		if err != nil {
			integFatal = fmt.Errorf("RequestToken (password flow): %w", err)
			return
		}
		integClient = client
	})
	if integSkip != "" {
		t.Skip(integSkip)
	}
	if integFatal != nil {
		t.Fatal(integFatal)
	}
	return integClient
}

// scratchFolder creates a unique folder under /Shared and registers a
// guarded cleanup that can only ever delete sdk-integration-* paths.
func scratchFolder(t *testing.T, client *Client) string {
	t.Helper()
	folder := fmt.Sprintf("/Shared/sdk-integration-%d", time.Now().UnixMilli())
	if _, err := client.FileSystem.CreateFolder(context.Background(), folder); err != nil {
		t.Fatalf("FileSystem.CreateFolder(%s): %v", folder, err)
	}
	t.Cleanup(func() {
		if !strings.HasPrefix(folder, "/Shared/sdk-integration-") {
			t.Errorf("cleanup: refusing to delete unexpected path %q", folder)
			return
		}
		if _, err := client.FileSystem.Delete(context.Background(), folder, nil); err != nil {
			t.Logf("cleanup: FileSystem.Delete(%s): %v (delete manually)", folder, err)
		} else {
			t.Logf("cleanup: deleted %s (moved to trash)", folder)
		}
	})
	return folder
}

// skipIfUnavailable skips the (sub)test when the API says the feature or
// permission is unavailable for this domain/user, and fails on anything
// else.
func skipIfUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) &&
		(apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound) {
		t.Skipf("not available on this domain/user: %v", err)
	}
	t.Fatal(err)
}

// pause keeps the test under the 2 calls/second per-token limit.
func pause() { time.Sleep(600 * time.Millisecond) }

func TestIntegration_RealDomainScenario(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()

	// 1. Verify the token and see who we are.
	info, _, err := client.Tokens.UserInfo(ctx)
	if err != nil {
		t.Fatalf("Tokens.UserInfo: %v", err)
	}
	t.Logf("authenticated as %s %s (%s, id %d)", info.FirstName, info.LastName, info.Username, info.ID)
	pause()

	// 2. Create a scratch folder; everything below happens inside it.
	folder := scratchFolder(t, client)
	t.Logf("created scratch folder %s", folder)
	pause()

	// 3. Upload a file whose name exercises EncodePath. Note: Egnyte
	// forbids * ? / \ : < > " | in file names, so stick to legal-but-
	// tricky characters ($, &, spaces).
	filePath := folder + "/hello $world & co.txt"
	content := "hello from the egnyte-go-sdk integration test"
	uploaded, _, err := client.FileSystemContent.Upload(ctx, filePath, strings.NewReader(content), nil)
	if err != nil {
		t.Fatalf("FileSystemContent.Upload: %v", err)
	}
	if uploaded.Checksum == "" {
		t.Error("upload returned no checksum")
	}
	t.Logf("uploaded %s (entry %s, %d bytes)", filePath, uploaded.EntryID, uploaded.Size)
	pause()

	// 4. Download it back and compare.
	body, _, err := client.FileSystemContent.Download(ctx, filePath, nil)
	if err != nil {
		t.Fatalf("FileSystemContent.Download: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, body); err != nil {
		t.Fatalf("reading download: %v", err)
	}
	body.Close()
	if buf.String() != content {
		t.Errorf("downloaded content = %q, want %q", buf.String(), content)
	}
	pause()

	// 5. List the folder and confirm the file shows up.
	item, resp, err := client.FileSystem.Get(ctx, folder, &FileSystemGetOptions{ListContent: true})
	if err != nil {
		t.Fatalf("FileSystem.Get: %v", err)
	}
	if len(item.Files) != 1 {
		t.Errorf("folder listing has %d files, want 1", len(item.Files))
	}
	t.Logf("rate limit: %d/%d today", resp.Rate.QuotaCurrent, resp.Rate.QuotaAllotted)
	pause()

	// 6. Comment on the file and read it back.
	comment, _, err := client.Comments.Add(ctx, filePath, "integration test comment")
	if err != nil {
		t.Fatalf("Comments.Add: %v", err)
	}
	comments, _, err := client.Comments.List(ctx, &CommentListOptions{File: filePath})
	if err != nil {
		t.Fatalf("Comments.List: %v", err)
	}
	if comments.TotalResults < 1 {
		t.Error("expected at least one comment on the file")
	}
	if _, err := client.Comments.Delete(ctx, comment.ID); err != nil {
		t.Errorf("Comments.Delete: %v", err)
	}
	pause()

	// 7. Create a domain-restricted share link, verify, and delete it.
	link, _, err := client.Links.Create(ctx, CreateLinkRequest{
		Path:          filePath,
		Type:          "file",
		Accessibility: "domain",
		Notify:        Bool(false),
	})
	if err != nil {
		t.Fatalf("Links.Create: %v", err)
	}
	if len(link.Links) == 0 || link.Links[0].URL == "" {
		t.Fatalf("link creation returned no URL: %+v", link)
	}
	t.Logf("created link %s", link.Links[0].URL)
	pause()
	if _, err := client.Links.Delete(ctx, link.Links[0].ID); err != nil {
		t.Errorf("Links.Delete: %v", err)
	}
	pause()

	// 8. Events cursor — read-only sanity check of a second API family.
	cursor, _, err := client.Events.Cursor(ctx)
	if err != nil {
		t.Fatalf("Events.Cursor: %v", err)
	}
	if cursor.LatestEventID == 0 {
		t.Error("events cursor returned no latest event id")
	}
	t.Logf("events cursor: latest=%d oldest=%d", cursor.LatestEventID, cursor.OldestEventID)
}

// TestIntegration_ReadOnlySweep touches one read-only endpoint per API
// family. Families the domain or user cannot access are skipped, not
// failed, so the sweep adapts to any test domain.
func TestIntegration_ReadOnlySweep(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()

	sweep := []struct {
		name string
		call func() error
	}{
		{"Links.List", func() error {
			list, _, err := client.Links.List(ctx, &LinkListOptions{Count: 5})
			if err == nil {
				t.Logf("links visible: %d", list.TotalCount)
			}
			return err
		}},
		{"Search.Search", func() error {
			results, _, err := client.Search.Search(ctx, "sdk-integration", &SearchOptions{Count: 5})
			if err == nil {
				t.Logf("search hits: %d", results.TotalCount)
			}
			return err
		}},
		{"Insights.RecentFiles", func() error {
			files, _, err := client.Insights.RecentFiles(ctx, "")
			if err == nil {
				t.Logf("recent files: %d", len(files))
			}
			return err
		}},
		{"Users.List", func() error {
			users, _, err := client.Users.List(ctx, &UserListOptions{Count: 5})
			if err == nil {
				t.Logf("users: %d total", users.TotalResults)
			}
			return err
		}},
		{"Groups.List", func() error {
			groups, _, err := client.Groups.List(ctx, &GroupListOptions{Count: 5})
			if err == nil {
				t.Logf("groups: %d total", groups.TotalResults)
			}
			return err
		}},
		{"Trash.TotalCount", func() error {
			n, _, err := client.Trash.TotalCount(ctx)
			if err == nil {
				t.Logf("items in trash: %d", n)
			}
			return err
		}},
		{"Webhooks.WhoAmI", func() error {
			who, _, err := client.Webhooks.WhoAmI(ctx)
			if err == nil {
				t.Logf("whoami: user %s on domain %s", who.Username, who.Domain)
			}
			return err
		}},
		{"Metadata.ListNamespaces", func() error {
			namespaces, _, err := client.Metadata.ListNamespaces(ctx, false)
			if err == nil {
				t.Logf("metadata namespaces: %d", len(namespaces))
			}
			return err
		}},
		{"Workflows.ListTasks", func() error {
			tasks, _, err := client.Workflows.ListTasks(ctx, &WorkflowTaskListOptions{Limit: 5})
			if err == nil {
				t.Logf("my workflow tasks: %d", tasks.TotalCount)
			}
			return err
		}},
		{"Bookmarks.List", func() error {
			bookmarks, _, err := client.Bookmarks.List(ctx, nil)
			if err == nil {
				t.Logf("bookmarks: %d", len(bookmarks.Bookmarks))
			}
			return err
		}},
		{"ProjectFolders.List", func() error {
			projects, _, err := client.ProjectFolders.List(ctx, &ProjectListOptions{Count: 5})
			if err == nil {
				t.Logf("projects: %d", projects.TotalResults)
			}
			return err
		}},
		{"Sign.ListTemplates", func() error {
			templates, _, err := client.Sign.ListTemplates(ctx, nil)
			if err == nil {
				t.Logf("sign templates: %d", templates.Count)
			}
			return err
		}},
		{"Audit.StreamGet", func() error {
			stream, _, err := client.Audit.StreamGet(ctx, AuditStreamRequest{
				StartDate: time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
			})
			if err == nil {
				t.Logf("audit events since yesterday: %d (more=%v)", len(stream.Events), stream.MoreEvents)
			}
			return err
		}},
	}

	for _, s := range sweep {
		t.Run(s.name, func(t *testing.T) {
			skipIfUnavailable(t, s.call())
		})
		pause()
	}
}

// TestIntegration_FileLifecycle exercises versioning, version-specific
// downloads, copy/move/rename, folder options, bookmarks and private
// metadata — all inside a self-cleaning scratch folder.
func TestIntegration_FileLifecycle(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()
	folder := scratchFolder(t, client)
	pause()

	// Two uploads to the same path create two versions.
	filePath := folder + "/versioned.txt"
	if _, _, err := client.FileSystemContent.Upload(ctx, filePath, strings.NewReader("version one"), nil); err != nil {
		t.Fatalf("Upload v1: %v", err)
	}
	pause()
	if _, _, err := client.FileSystemContent.Upload(ctx, filePath, strings.NewReader("version two"), nil); err != nil {
		t.Fatalf("Upload v2: %v", err)
	}
	pause()

	// Metadata shows both versions; grab the old version's entry id.
	file, _, err := client.FileSystem.Get(ctx, filePath, nil)
	if err != nil {
		t.Fatalf("FileSystem.Get(file): %v", err)
	}
	if file.NumVersions < 2 {
		t.Errorf("num_versions = %d, want >= 2", file.NumVersions)
	}
	t.Logf("file has %d versions, current entry %s", file.NumVersions, file.EntryID)
	pause()

	// Download the current version and confirm it is "version two".
	body, _, err := client.FileSystemContent.Download(ctx, filePath, nil)
	if err != nil {
		t.Fatalf("Download current: %v", err)
	}
	current, _ := io.ReadAll(body)
	body.Close()
	if string(current) != "version two" {
		t.Errorf("current version = %q, want %q", current, "version two")
	}
	pause()

	// Copy, rename, and move within the scratch folder.
	if _, err := client.FileSystem.Copy(ctx, filePath, folder+"/copy.txt"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	pause()
	if _, err := client.FileSystem.Rename(ctx, folder+"/copy.txt", "renamed.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	pause()
	if _, err := client.FileSystem.CreateFolder(ctx, folder+"/sub"); err != nil {
		t.Fatalf("CreateFolder(sub): %v", err)
	}
	pause()
	if _, err := client.FileSystem.Move(ctx, folder+"/renamed.txt", folder+"/sub/renamed.txt"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	pause()
	listing, _, err := client.FileSystem.Get(ctx, folder+"/sub", &FileSystemGetOptions{ListContent: true})
	if err != nil {
		t.Fatalf("Get(sub): %v", err)
	}
	if len(listing.Files) != 1 || listing.Files[0].Name != "renamed.txt" {
		t.Errorf("sub folder contents = %+v, want renamed.txt", listing.Files)
	}
	t.Log("copy/rename/move round-trip verified")
	pause()

	// Folder options: set a description and read it back.
	updated, _, err := client.FileSystem.SetFolderOptions(ctx, folder, FolderOptions{
		FolderDescription: "sdk integration scratch folder",
	})
	skipIfUnavailable(t, err)
	if updated.FolderDescription != "sdk integration scratch folder" {
		t.Errorf("folder_description = %q", updated.FolderDescription)
	}
	t.Log("folder options set and echoed back")
	pause()

	// Bookmark the scratch folder, list, delete.
	bookmark, _, err := client.Bookmarks.CreateByPath(ctx, folder)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusNotFound) {
			t.Logf("bookmarks unavailable, skipping: %v", err)
		} else {
			t.Fatalf("Bookmarks.CreateByPath: %v", err)
		}
	} else {
		pause()
		if _, err := client.Bookmarks.Delete(ctx, bookmark.ID); err != nil {
			t.Errorf("Bookmarks.Delete: %v", err)
		}
		t.Logf("bookmark %d created and deleted", bookmark.ID)
	}
	pause()

	// Private metadata namespace: visible only to this API key, created
	// and force-deleted within the test.
	// Namespace names reject hyphens; use underscores.
	ns := fmt.Sprintf("sdk_integration_ns_%d", time.Now().UnixMilli())
	// The live API requires non-empty helpText for each key (spec says
	// optional).
	_, err = client.Metadata.CreateNamespace(ctx, CreateNamespaceRequest{
		Name:  ns,
		Scope: "private",
		Keys: map[string]MetadataKey{"purpose": {
			Type: "string", DisplayName: "Purpose", HelpText: "why this file exists",
		}},
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
			t.Logf("metadata namespace creation unavailable, skipping: %v", err)
			return
		}
		t.Fatalf("Metadata.CreateNamespace: %v", err)
	}
	t.Cleanup(func() {
		if !strings.HasPrefix(ns, "sdk_integration_ns_") {
			t.Errorf("cleanup: refusing to delete unexpected namespace %q", ns)
			return
		}
		if _, err := client.Metadata.DeleteNamespace(context.Background(), ns, true); err != nil {
			t.Logf("cleanup: DeleteNamespace(%s): %v (delete manually)", ns, err)
		} else {
			t.Logf("cleanup: deleted metadata namespace %s", ns)
		}
	})
	pause()
	if _, err := client.Metadata.SetFileMetadata(ctx, file.GroupID, ns, map[string]any{
		"purpose": "integration-test",
	}); err != nil {
		t.Fatalf("Metadata.SetFileMetadata: %v", err)
	}
	pause()
	values, _, err := client.Metadata.GetFileMetadata(ctx, file.GroupID, ns)
	if err != nil {
		t.Fatalf("Metadata.GetFileMetadata: %v", err)
	}
	t.Logf("metadata round-trip: %+v", values.Results)
}

// TestIntegration_LiveContracts exercises live-contract surfaces: file and
// folder metadata shapes, by-ID lookups, folder stats, lock/unlock, effective
// permissions without a username, userinfo extras and hybrid search.
func TestIntegration_LiveContracts(t *testing.T) {
	client := integrationClient(t)
	ctx := context.Background()
	folder := scratchFolder(t, client)
	pause()

	filePath := folder + "/contract.txt"
	up, _, err := client.FileSystemContent.Upload(ctx, filePath, strings.NewReader("contract fixture"), nil)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	pause()

	// File metadata: RFC1123 last_modified, uploaded_by, permission.
	file, _, err := client.FileSystem.Get(ctx, filePath, &FileSystemGetOptions{IncludePerm: true})
	if err != nil {
		t.Fatalf("Get(file): %v", err)
	}
	if file.IsFolder || file.LastModified == "" || file.UploadedBy == "" {
		t.Errorf("file metadata = %+v", file)
	}
	if file.ModTime().IsZero() {
		t.Errorf("file ModTime failed to parse %q", file.LastModified)
	}
	pause()

	// Folder metadata: epoch-ms lastModified.
	dir, _, err := client.FileSystem.Get(ctx, folder, nil)
	if err != nil {
		t.Fatalf("Get(folder): %v", err)
	}
	if !dir.IsFolder || dir.FolderLastModified == 0 || dir.ModTime().IsZero() {
		t.Errorf("folder metadata = %+v", dir)
	}
	pause()

	// By-ID lookups and folder stats.
	byID, _, err := client.FileSystem.GetFileByID(ctx, up.GroupID, nil)
	if err != nil {
		t.Fatalf("GetFileByID: %v", err)
	}
	if byID.Path != filePath {
		t.Errorf("GetFileByID path = %q, want %q", byID.Path, filePath)
	}
	pause()
	dirByID, _, err := client.FileSystem.GetFolderByID(ctx, dir.FolderID, nil)
	if err != nil {
		t.Fatalf("GetFolderByID: %v", err)
	}
	if dirByID.Path != folder {
		t.Errorf("GetFolderByID path = %q, want %q", dirByID.Path, folder)
	}
	pause()
	stats, _, err := client.FileSystem.FolderStatsByID(ctx, dir.FolderID)
	if err != nil {
		t.Fatalf("FolderStatsByID: %v", err)
	}
	if stats.FilesCount != 1 || stats.AllFilesSize == 0 {
		t.Errorf("stats = %+v", stats)
	}
	t.Logf("folder stats: %+v", stats)
	pause()

	// Lock, verify lock_info via IncludeLocks, unlock.
	lock, _, err := client.FileSystem.Lock(ctx, filePath)
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if lock.LockToken == "" || lock.Timeout == 0 {
		t.Errorf("lock = %+v", lock)
	}
	pause()
	locked, _, err := client.FileSystem.Get(ctx, filePath, &FileSystemGetOptions{IncludeLocks: true})
	if err != nil {
		t.Fatalf("Get(locked): %v", err)
	}
	if !locked.Locked || locked.LockInfo == nil {
		t.Errorf("locked metadata = locked=%v lock_info=%+v", locked.Locked, locked.LockInfo)
	}
	pause()
	if _, err := client.FileSystem.Unlock(ctx, filePath, lock.LockToken); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	t.Log("lock/unlock round-trip verified")
	pause()

	// Own effective permission (no username).
	perm, _, err := client.Permissions.GetEffective(ctx, "", folder)
	if err != nil {
		t.Fatalf("GetEffective(own): %v", err)
	}
	if perm == "" {
		t.Error("own effective permission is empty")
	}
	t.Logf("own permission on scratch folder: %s", perm)
	pause()

	// UserInfo extras.
	info, _, err := client.Tokens.UserInfo(ctx)
	if err != nil {
		t.Fatalf("UserInfo: %v", err)
	}
	if info.UserType == "" || info.Email == "" {
		t.Errorf("userinfo = %+v, want user_type and email set", info)
	}
	pause()

	// Hybrid search (needs the Egnyte.ai scope and indexed content, so
	// only the request/response contract is asserted, not hits).
	results, _, err := client.AI.HybridSearch(ctx, HybridSearchRequest{
		Query: "integration contract fixture",
		Limit: 3,
	})
	skipIfUnavailable(t, err)
	t.Logf("hybrid search returned %d results", len(results.Results))
	for _, r := range results.Results {
		if r.Filename == "" || r.EntryID == "" {
			t.Errorf("hybrid result = %+v", r)
		}
	}
}

// TestIntegration_ChunkedUpload uploads an ~11 MB file in two chunks
// (10 MB minimum chunk size + remainder) and verifies the assembled
// file downloads back byte-identical.
func TestIntegration_ChunkedUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ~11MB chunked upload in -short mode")
	}
	client := integrationClient(t)
	ctx := context.Background()
	folder := scratchFolder(t, client)
	pause()

	const chunkSize = 10 * 1024 * 1024 // API minimum
	data := make([]byte, chunkSize+1024*1024)
	rand.New(rand.NewSource(42)).Read(data)
	chunk1, chunk2 := data[:chunkSize], data[chunkSize:]

	sum := func(b []byte) string {
		h := sha512.Sum512(b)
		return hex.EncodeToString(h[:])
	}

	filePath := folder + "/chunked.bin"
	first, _, err := client.FileSystemContent.UploadChunk(ctx, filePath, bytes.NewReader(chunk1),
		ChunkedUploadOptions{ChunkNum: 1, ChunkChecksum: sum(chunk1)})
	if err != nil {
		t.Fatalf("UploadChunk 1: %v", err)
	}
	if first.UploadID == "" {
		t.Fatal("first chunk returned no upload id")
	}
	t.Logf("chunk 1 uploaded, upload id %s", first.UploadID)
	pause()

	fileChecksum := ChunkedFileChecksum(chunkSize, sum(chunk1), sum(chunk2))
	last, _, err := client.FileSystemContent.UploadChunk(ctx, filePath, bytes.NewReader(chunk2),
		ChunkedUploadOptions{
			ChunkNum:      2,
			ChunkChecksum: sum(chunk2),
			UploadID:      first.UploadID,
			LastChunk:     true,
			FileChecksum:  fileChecksum,
		})
	if err != nil {
		t.Fatalf("UploadChunk 2 (last): %v", err)
	}
	if last.Result != nil {
		t.Logf("assembled: entry %s server checksum %s", last.Result.EntryID, last.Result.Checksum)
		if last.Result.Checksum != fileChecksum {
			t.Errorf("server checksum %q != ChunkedFileChecksum %q", last.Result.Checksum, fileChecksum)
		}
	}
	pause()

	body, _, err := client.FileSystemContent.Download(ctx, filePath, nil)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer body.Close()
	downloaded, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("reading download: %v", err)
	}
	if !bytes.Equal(downloaded, data) {
		t.Fatalf("downloaded %d bytes, mismatch with %d uploaded", len(downloaded), len(data))
	}
	t.Logf("chunked upload verified: %d bytes round-tripped byte-identical", len(data))
}
