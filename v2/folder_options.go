package egnyte

import (
	"context"
	"fmt"
	"net/http"
)

// FolderEmailPreferences are write-only notification settings for a
// folder; they are never included in responses.
type FolderEmailPreferences struct {
	// ContentUpdates notifies when folder content is updated.
	ContentUpdates *bool `json:"content_updates,omitempty"`
	// ContentAccessed notifies when folder content is accessed (Power
	// Users and above only).
	ContentAccessed *bool `json:"content_accessed,omitempty"`
}

// FolderOptions modify a folder's settings. At least one field must be
// set.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/folder-options-api
// OAuth scope: Egnyte.filesystem.
type FolderOptions struct {
	// FolderDescription is a text description (max 200 characters).
	FolderDescription string `json:"folder_description,omitempty"`
	// AllowLinks toggles all link creation in the folder.
	AllowLinks *bool `json:"allow_links,omitempty"`
	// AllowUploadLinks toggles upload links.
	AllowUploadLinks *bool `json:"allow_upload_links,omitempty"`
	// PublicLinks is files_folders, folders, files or disabled.
	PublicLinks string `json:"public_links,omitempty"`
	// RestrictMoveDelete limits move/delete to admins and owners.
	// MoveDeleteFolderRestriction takes precedence when both are set.
	RestrictMoveDelete *bool `json:"restrict_move_delete,omitempty"`
	// MoveDeleteFolderRestriction is admins_owners_or_full_access_users,
	// admins_or_owners or admins_only. Not valid on top-level folders.
	MoveDeleteFolderRestriction string `json:"move_delete_folder_restriction,omitempty"`
	// InheritanceRule controls propagation to subfolders: legacy
	// (default), this_folder_only, this_folder_and_subfolders_until_overridden
	// or this_folder_and_subfolders.
	InheritanceRule string `json:"inheritance_rule,omitempty"`
	// EmailPreferences are write-only notification settings.
	EmailPreferences *FolderEmailPreferences `json:"email_preferences,omitempty"`
}

// SetFolderOptions updates a folder's configuration (description, link
// permissions, move/delete restrictions, notifications) and returns the
// updated folder metadata.
//
// PATCH /pubapi/v1/fs/{path}
func (s *FileSystemService) SetFolderOptions(ctx context.Context, path string, options FolderOptions) (*FileSystemItem, *Response, error) {
	if options == (FolderOptions{}) {
		return nil, nil, fmt.Errorf("egnyte: at least one folder option must be set")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPatch, fsPath("v1", path), options)
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
