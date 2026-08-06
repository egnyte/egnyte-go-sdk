package egnyte

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// FileSystemService handles browsing and manipulating files and folders.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/file-system-management
// OAuth scope: Egnyte.filesystem. Access to individual items follows the
// authenticated user's file and folder permissions.
type FileSystemService service

// FolderEntry describes a folder inside a listed folder. Folders report
// modification time as epoch milliseconds (files use an RFC1123 string —
// the two shapes are the live API's, not the SDK's).
type FolderEntry struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	FolderID     string `json:"folder_id"`
	ParentID     string `json:"parent_id"`
	IsFolder     bool   `json:"is_folder"`
	LastModified int64  `json:"lastModified"`
	Uploaded     int64  `json:"uploaded"`
}

// FileEntry describes a file inside a listed folder.
type FileEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	EntryID  string `json:"entry_id"`
	GroupID  string `json:"group_id"`
	ParentID string `json:"parent_id"`
	IsFolder bool   `json:"is_folder"`
	// LastModified is an RFC1123 timestamp, e.g.
	// "Mon, 03 Aug 2026 11:48:35 GMT".
	LastModified string `json:"last_modified"`
	// Uploaded is the upload time in epoch milliseconds.
	Uploaded    int64  `json:"uploaded"`
	Checksum    string `json:"checksum"`
	NumVersions int    `json:"num_versions"`
	UploadedBy  string `json:"uploaded_by"`
	Locked      bool   `json:"locked"`
	Permission  string `json:"permission"`
}

// FileVersion is one entry of a file's version history.
type FileVersion struct {
	EntryID  string `json:"entry_id"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
	IsFolder bool   `json:"is_folder"`
	// LastModified is an RFC1123 timestamp.
	LastModified string `json:"last_modified"`
	// Uploaded is the upload time in epoch milliseconds.
	Uploaded   int64  `json:"uploaded"`
	UploadedBy string `json:"uploaded_by"`
	// CustomMetadata is included with ListCustomMetadata; its shape
	// depends on the namespaces set, so the raw JSON is preserved.
	CustomMetadata json.RawMessage `json:"custom_metadata"`
}

// FileLockInfo identifies who holds the lock on a locked file, included
// with FileSystemGetOptions.IncludeLocks.
type FileLockInfo struct {
	OwnerID   int    `json:"owner_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// FileSystemItem is the metadata of a file or folder. Check IsFolder to
// tell them apart: the live API returns different shapes for the two
// (notably the modification time — see LastModified vs
// FolderLastModified, or use ModTime for either). For folders fetched
// with ListContent, Folders and Files carry the folder contents and
// Count/Offset/TotalCount describe the listing window.
type FileSystemItem struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	IsFolder   bool          `json:"is_folder"`
	FolderID   string        `json:"folder_id"`
	ParentID   string        `json:"parent_id"`
	Count      int           `json:"count"`
	Offset     int           `json:"offset"`
	TotalCount int           `json:"total_count"`
	Folders    []FolderEntry `json:"folders"`
	Files      []FileEntry   `json:"files"`

	// FolderLastModified is the folder modification time in epoch
	// milliseconds. Folders only — files carry LastModified instead.
	FolderLastModified int64 `json:"lastModified"`

	// File-only fields.
	Size    int64  `json:"size"`
	EntryID string `json:"entry_id"`
	GroupID string `json:"group_id"`
	// LastModified is the file modification time as an RFC1123
	// timestamp, e.g. "Mon, 03 Aug 2026 11:48:35 GMT". Files only —
	// folders carry FolderLastModified instead.
	LastModified string `json:"last_modified"`
	// Uploaded is the upload time in epoch milliseconds.
	Uploaded    int64  `json:"uploaded"`
	Checksum    string `json:"checksum"`
	NumVersions int    `json:"num_versions"`
	UploadedBy  string `json:"uploaded_by"`
	Permission  string `json:"permission"`
	Locked      bool   `json:"locked"`
	// LockInfo is set with IncludeLocks when the file is locked.
	LockInfo *FileLockInfo `json:"lock_info"`
	// Versions is the file's version history (latest version excluded).
	Versions []FileVersion `json:"versions"`
	// CustomMetadata is included with ListCustomMetadata; its shape
	// depends on the namespaces set, so the raw JSON is preserved.
	CustomMetadata json.RawMessage `json:"custom_metadata"`

	// Folder option fields, populated by SetFolderOptions and by Get
	// when the folder has options configured.
	FolderDescription           string   `json:"folder_description"`
	AllowLinks                  *bool    `json:"allow_links"`
	AllowUploadLinks            *bool    `json:"allow_upload_links"`
	AllowedFileLinkTypes        []string `json:"allowed_file_link_types"`
	AllowedFolderLinkTypes      []string `json:"allowed_folder_link_types"`
	PublicLinks                 string   `json:"public_links"`
	RestrictMoveDelete          *bool    `json:"restrict_move_delete"`
	MoveDeleteFolderRestriction string   `json:"move_delete_folder_restriction"`
}

// ModTime returns the modification time regardless of whether the item
// is a file (RFC1123 string) or a folder (epoch milliseconds). The zero
// time is returned if neither field is set or parseable.
func (i *FileSystemItem) ModTime() time.Time {
	if i.IsFolder {
		if i.FolderLastModified == 0 {
			return time.Time{}
		}
		return time.UnixMilli(i.FolderLastModified)
	}
	t, err := time.Parse(time.RFC1123, i.LastModified)
	if err != nil {
		return time.Time{}
	}
	return t
}

// FileSystemGetOptions control the Get / GetV2 metadata queries. Zero
// values are omitted from the request.
type FileSystemGetOptions struct {
	// ListContent includes folder contents in the response.
	ListContent bool
	// Count limits how many items are returned (default 100, max 1000).
	Count int
	// Offset is the zero-based index to start listing from.
	Offset int
	// SortBy is one of: name, path, size, last_modified, created_time.
	SortBy string
	// SortDirection is "ascending" (default) or "descending".
	SortDirection string
	// Perms returns users/groups with permissions on the folder
	// (requires an admin token).
	Perms bool
	// IncludePerm includes permission info for the item.
	IncludePerm bool
	// IncludeLocks includes file locking info.
	IncludeLocks bool
	// IncludeCollaboration includes collaboration metadata.
	IncludeCollaboration bool
	// AllowedLinkTypes includes info about allowed link types.
	AllowedLinkTypes bool
	// Key retrieves a specific file version by its entry ID.
	Key string
	// ListCustomMetadata includes custom metadata in the response.
	ListCustomMetadata bool
}

func (o *FileSystemGetOptions) values() url.Values {
	v := url.Values{}
	if o == nil {
		return v
	}
	setBool := func(name string, val bool) {
		if val {
			v.Set(name, "true")
		}
	}
	setBool("list_content", o.ListContent)
	if o.Count > 0 {
		v.Set("count", strconv.Itoa(o.Count))
	}
	if o.Offset > 0 {
		v.Set("offset", strconv.Itoa(o.Offset))
	}
	if o.SortBy != "" {
		v.Set("sort_by", o.SortBy)
	}
	if o.SortDirection != "" {
		v.Set("sort_direction", o.SortDirection)
	}
	setBool("perms", o.Perms)
	setBool("include_perm", o.IncludePerm)
	setBool("include_locks", o.IncludeLocks)
	setBool("include_collaboration", o.IncludeCollaboration)
	setBool("allowed_link_types", o.AllowedLinkTypes)
	if o.Key != "" {
		v.Set("key", o.Key)
	}
	setBool("list_custom_metadata", o.ListCustomMetadata)
	return v
}

// FileSystemAction is the request body of the POST fs endpoints. Action
// is one of add_folder, move or copy. For move and copy, Destination is
// the full target path including the (possibly new) item name — the
// live API rejects unknown properties, so renaming is expressed through
// the destination path rather than a separate field.
type FileSystemAction struct {
	Action      string `json:"action"`
	Destination string `json:"destination,omitempty"`
	// Permissions optionally propagates permissions on move/copy; its
	// shape is not fixed by the spec, so any JSON-encodable value is
	// accepted.
	Permissions any `json:"permissions,omitempty"`
}

// FileSystemDeleteOptions control Delete / DeleteV2.
type FileSystemDeleteOptions struct {
	// EntryID deletes only the given version of a file.
	EntryID string
}

func fsPath(version, path string) string {
	return version + "/fs" + EncodePath(ensureLeadingSlash(path))
}

func ensureLeadingSlash(p string) string {
	if len(p) == 0 || p[0] != '/' {
		return "/" + p
	}
	return p
}

func (s *FileSystemService) get(ctx context.Context, version, path string, opts *FileSystemGetOptions) (*FileSystemItem, *Response, error) {
	urlPath := fsPath(version, path)
	if q := opts.values().Encode(); q != "" {
		urlPath += "?" + q
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	item := new(FileSystemItem)
	resp, err := s.client.Do(req, item)
	if err != nil {
		return nil, resp, err
	}
	return item, resp, nil
}

// Get returns metadata for the file or folder at path. With
// opts.ListContent, folder contents are included.
//
// GET /pubapi/v1/fs/{path}
func (s *FileSystemService) Get(ctx context.Context, path string, opts *FileSystemGetOptions) (*FileSystemItem, *Response, error) {
	return s.get(ctx, "v1", path, opts)
}

// GetV2 is Get against the v2 endpoint, which may include additional
// metadata fields.
//
// GET /pubapi/v2/fs/{path}
func (s *FileSystemService) GetV2(ctx context.Context, path string, opts *FileSystemGetOptions) (*FileSystemItem, *Response, error) {
	return s.get(ctx, "v2", path, opts)
}

// Action performs a raw file system operation at path. Prefer the typed
// helpers (CreateFolder, Move, Copy, Rename); Action is the escape hatch
// for permission propagation and future actions.
//
// POST /pubapi/v1/fs/{path}
func (s *FileSystemService) Action(ctx context.Context, path string, action FileSystemAction) (*Response, error) {
	return s.action(ctx, "v1", path, action)
}

// ActionV2 is Action against the v2 endpoint, which supports advanced
// permission handling.
//
// POST /pubapi/v2/fs/{path}
func (s *FileSystemService) ActionV2(ctx context.Context, path string, action FileSystemAction) (*Response, error) {
	return s.action(ctx, "v2", path, action)
}

func (s *FileSystemService) action(ctx context.Context, version, path string, action FileSystemAction) (*Response, error) {
	if action.Action == "" {
		return nil, fmt.Errorf("egnyte: file system action must not be empty")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, fsPath(version, path), action)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// CreateFolder creates a new folder at path.
func (s *FileSystemService) CreateFolder(ctx context.Context, path string) (*Response, error) {
	return s.Action(ctx, path, FileSystemAction{Action: "add_folder"})
}

// Move moves the file or folder at path. destination is the full target
// path including the item name — using a different name renames the
// item as part of the move.
func (s *FileSystemService) Move(ctx context.Context, path, destination string) (*Response, error) {
	return s.Action(ctx, path, FileSystemAction{Action: "move", Destination: destination})
}

// Copy copies the file or folder at path. destination is the full target
// path including the copy's name.
func (s *FileSystemService) Copy(ctx context.Context, path, destination string) (*Response, error) {
	return s.Action(ctx, path, FileSystemAction{Action: "copy", Destination: destination})
}

// Rename renames the file or folder at path to newName within its
// current parent folder (a move to the same directory — the API has no
// separate rename action).
func (s *FileSystemService) Rename(ctx context.Context, path, newName string) (*Response, error) {
	p := ensureLeadingSlash(path)
	parent := p[:strings.LastIndex(p, "/")+1]
	return s.Action(ctx, path, FileSystemAction{Action: "move", Destination: parent + newName})
}

// Delete removes the file or folder at path. Folders must be empty.
//
// DELETE /pubapi/v1/fs/{path}
func (s *FileSystemService) Delete(ctx context.Context, path string, opts *FileSystemDeleteOptions) (*Response, error) {
	return s.delete(ctx, "v1", path, opts)
}

// DeleteV2 is Delete against the v2 endpoint.
//
// DELETE /pubapi/v2/fs/{path}
func (s *FileSystemService) DeleteV2(ctx context.Context, path string, opts *FileSystemDeleteOptions) (*Response, error) {
	return s.delete(ctx, "v2", path, opts)
}

func (s *FileSystemService) delete(ctx context.Context, version, path string, opts *FileSystemDeleteOptions) (*Response, error) {
	urlPath := fsPath(version, path)
	if opts != nil && opts.EntryID != "" {
		urlPath += "?" + url.Values{"entry_id": {opts.EntryID}}.Encode()
	}
	req, err := s.client.NewRequest(ctx, http.MethodDelete, urlPath, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

func (s *FileSystemService) getByID(ctx context.Context, urlPath string, opts *FileSystemGetOptions) (*FileSystemItem, *Response, error) {
	if q := opts.values().Encode(); q != "" {
		urlPath += "?" + q
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	item := new(FileSystemItem)
	resp, err := s.client.Do(req, item)
	if err != nil {
		return nil, resp, err
	}
	return item, resp, nil
}

// GetFileByID returns metadata for the file with the given file system
// id (group_id), which is stable across renames and moves.
//
// GET /pubapi/v1/fs/ids/file/{id}
func (s *FileSystemService) GetFileByID(ctx context.Context, id string, opts *FileSystemGetOptions) (*FileSystemItem, *Response, error) {
	return s.getByID(ctx, "v1/fs/ids/file/"+url.PathEscape(id), opts)
}

// GetFolderByID returns metadata for the folder with the given folder id.
//
// GET /pubapi/v1/fs/ids/folder/{id}
func (s *FileSystemService) GetFolderByID(ctx context.Context, id string, opts *FileSystemGetOptions) (*FileSystemItem, *Response, error) {
	return s.getByID(ctx, "v1/fs/ids/folder/"+url.PathEscape(id), opts)
}

// FolderStats summarize a folder's contents recursively.
type FolderStats struct {
	AllVersionsSize     int64 `json:"allVersionsSize"`
	AllFilesSize        int64 `json:"allFilesSize"`
	FilesCount          int   `json:"filesCount"`
	FileVersionsCount   int   `json:"fileVersionsCount"`
	FoldersCount        int   `json:"foldersCount"`
	AllFilesSizeInKB    int64 `json:"allFilesSizeInKB"`
	AllVersionsSizeInKB int64 `json:"allVersionsSizeInKB"`
}

// FolderStatsByID returns recursive size and item counts for the folder
// with the given folder id. There is no path-based variant of this
// endpoint.
//
// GET /pubapi/v1/fs/ids/folder/{id}/stats
func (s *FileSystemService) FolderStatsByID(ctx context.Context, id string) (*FolderStats, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/fs/ids/folder/"+url.PathEscape(id)+"/stats", nil)
	if err != nil {
		return nil, nil, err
	}
	stats := new(FolderStats)
	resp, err := s.client.Do(req, stats)
	if err != nil {
		return nil, resp, err
	}
	return stats, resp, nil
}

// FileLock is the result of locking a file.
type FileLock struct {
	// LockToken must be presented to Unlock.
	LockToken string `json:"lock_token"`
	// Timeout is the lock's remaining lifetime in seconds.
	Timeout int `json:"timeout"`
}

// String implements fmt.Stringer with the lock token redacted, so
// logging the lock (including with %+v) never leaks it.
func (l FileLock) String() string {
	return fmt.Sprintf("egnyte.FileLock{LockToken:%s, Timeout:%d}", redacted, l.Timeout)
}

// GoString implements fmt.GoStringer (%#v) with the token redacted.
func (l FileLock) GoString() string { return l.String() }

// Lock locks the file at path against modification by other users. The
// returned token is required to unlock.
//
// POST /pubapi/v1/fs/{path} (action lock)
func (s *FileSystemService) Lock(ctx context.Context, path string) (*FileLock, *Response, error) {
	body := struct {
		Action string `json:"action"`
	}{"lock"}
	req, err := s.client.NewRequest(ctx, http.MethodPost, fsPath("v1", path), body)
	if err != nil {
		return nil, nil, err
	}
	lock := new(FileLock)
	resp, err := s.client.Do(req, lock)
	if err != nil {
		return nil, resp, err
	}
	return lock, resp, nil
}

// Unlock releases the lock on the file at path using the token returned
// by Lock.
//
// POST /pubapi/v1/fs/{path} (action unlock)
func (s *FileSystemService) Unlock(ctx context.Context, path, lockToken string) (*Response, error) {
	if lockToken == "" {
		return nil, fmt.Errorf("egnyte: lock token is required")
	}
	body := struct {
		Action    string `json:"action"`
		LockToken string `json:"lock_token"`
	}{"unlock", lockToken}
	req, err := s.client.NewRequest(ctx, http.MethodPost, fsPath("v1", path), body)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
