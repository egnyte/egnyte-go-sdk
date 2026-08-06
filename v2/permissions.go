package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PermissionsService manages folder access control for users and groups.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/permissions-api
// OAuth scope: Egnyte.permission. Modifying a folder requires Owner access or
// an administrator token.
type PermissionsService service

// PermissionLevel is the access level assigned to a user or group.
type PermissionLevel string

// Permission levels, from none to owner.
const (
	PermissionNone       PermissionLevel = "None"
	PermissionNav        PermissionLevel = "Nav"
	PermissionViewerOnly PermissionLevel = "Viewer Only"
	PermissionViewer     PermissionLevel = "Viewer"
	PermissionEditor     PermissionLevel = "Editor"
	PermissionFull       PermissionLevel = "Full"
	PermissionOwner      PermissionLevel = "Owner"
)

// FolderPermissions describes all permissions set on a folder.
type FolderPermissions struct {
	UserPerms           map[string]PermissionLevel `json:"userPerms"`
	GroupPerms          map[string]PermissionLevel `json:"groupPerms"`
	InheritsPermissions bool                       `json:"inheritsPermissions"`
}

// SetFolderPermissions is the delta body for Set: only the permissions
// listed here are modified. Remove a permission by setting it to
// PermissionNone.
type SetFolderPermissions struct {
	UserPerms  map[string]PermissionLevel `json:"userPerms,omitempty"`
	GroupPerms map[string]PermissionLevel `json:"groupPerms,omitempty"`
	// InheritsPermissions toggles inheritance from the parent folder.
	// Include it only when changing inheritance status.
	InheritsPermissions *bool `json:"inheritsPermissions,omitempty"`
	// KeepParentPermissions is only valid when disabling inheritance:
	// whether to copy the currently inherited permissions onto the folder.
	KeepParentPermissions *bool `json:"keepParentPermissions,omitempty"`
}

// PermissionSubject pairs a username or group name with its permission.
type PermissionSubject struct {
	Subject    string          `json:"subject"`
	Permission PermissionLevel `json:"permission"`
}

// FolderPermissionsV1 is the response of the deprecated v1 lookup.
type FolderPermissionsV1 struct {
	Users  []PermissionSubject `json:"users"`
	Groups []PermissionSubject `json:"groups"`
}

// GetEffective returns the effective permission of a user on a folder,
// accounting for group memberships and inheritance. An empty username
// returns the calling user's own permission.
//
// GET /pubapi/v1/perms/user/{username}
// GET /pubapi/v1/perms/user
func (s *PermissionsService) GetEffective(ctx context.Context, username, folder string) (PermissionLevel, *Response, error) {
	if folder == "" {
		return "", nil, fmt.Errorf("egnyte: folder is required")
	}
	urlPath := "v1/perms/user"
	if username != "" {
		urlPath += "/" + url.PathEscape(username)
	}
	urlPath += "?" + url.Values{"folder": {folder}}.Encode()
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return "", nil, err
	}
	var body struct {
		Permission PermissionLevel `json:"permission"`
	}
	resp, err := s.client.Do(req, &body)
	if err != nil {
		return "", resp, err
	}
	return body.Permission, resp, nil
}

// Get returns all user and group permissions set on a folder, including
// its inheritance status.
//
// GET /pubapi/v2/perms/{path}
func (s *PermissionsService) Get(ctx context.Context, path string) (*FolderPermissions, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v2/perms"+EncodePath(ensureLeadingSlash(path)), nil)
	if err != nil {
		return nil, nil, err
	}
	perms := new(FolderPermissions)
	resp, err := s.client.Do(req, perms)
	if err != nil {
		return nil, resp, err
	}
	return perms, resp, nil
}

// Set updates permissions on a folder. This is a delta operation: only
// the entries in perms are changed, everything else is left as is.
//
// POST /pubapi/v2/perms/{path}
func (s *PermissionsService) Set(ctx context.Context, path string, perms SetFolderPermissions) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/perms"+EncodePath(ensureLeadingSlash(path)), perms)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// GetV1 returns folder permissions for specific users and/or groups. At
// least one of users or groups must be non-empty.
//
// Deprecated: use Get (v2), which returns all permissions in one call.
//
// GET /pubapi/v1/perms/folder/{path}
func (s *PermissionsService) GetV1(ctx context.Context, path string, users, groups []string) (*FolderPermissionsV1, *Response, error) {
	if len(users) == 0 && len(groups) == 0 {
		return nil, nil, fmt.Errorf("egnyte: at least one of users or groups is required")
	}
	q := url.Values{}
	if len(users) > 0 {
		q.Set("users", strings.Join(users, "|"))
	}
	if len(groups) > 0 {
		q.Set("groups", strings.Join(groups, "|"))
	}
	urlPath := "v1/perms/folder" + EncodePath(ensureLeadingSlash(path)) + "?" + q.Encode()
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	perms := new(FolderPermissionsV1)
	resp, err := s.client.Do(req, perms)
	if err != nil {
		return nil, resp, err
	}
	return perms, resp, nil
}

// SetV1 sets the same permission level for multiple users and/or groups
// on a folder. At least one of users or groups must be non-empty.
//
// Deprecated: use Set (v2), which updates users and groups individually.
//
// POST /pubapi/v1/perms/folder/{path}
func (s *PermissionsService) SetV1(ctx context.Context, path string, users, groups []string, permission PermissionLevel) (*Response, error) {
	if len(users) == 0 && len(groups) == 0 {
		return nil, fmt.Errorf("egnyte: at least one of users or groups is required")
	}
	body := struct {
		Users      []string        `json:"users,omitempty"`
		Groups     []string        `json:"groups,omitempty"`
		Permission PermissionLevel `json:"permission"`
	}{users, groups, permission}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/perms/folder"+EncodePath(ensureLeadingSlash(path)), body)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
