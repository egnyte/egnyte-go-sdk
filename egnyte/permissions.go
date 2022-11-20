package egnyte

import (
	"context"
	"errors"
	"fmt"
	"path"
)

// Egnyte Permissions APIs

var validPermissions = []string{
	"Viewer",
	"Editor",
	"Full",
	"Owner",
}

var FolderRequired = errors.New("object must be a folder")

// isValidPermission checks if the provided string is a valid Egnyte permission or not
func isValidPermission(permissionString string) bool {
	for _, permission := range validPermissions {
		if permission == permissionString {
			return true
		}
	}
	return false
}

// GetPermissions fetches the permissions for the object
func (o *Object) GetPermissions(ctx context.Context) (*FolderPermission, error) {
	if !o.IsFolder {
		return nil, FolderRequired
	}
	uri := path.Join(URI_PERMISSIONS, o.Path)
	opts := &requestOptions{
		Method: "GET",
		Path:   uri,
	}
	var perms *FolderPermission
	_, err := o.Client.doRequest(ctx, opts, nil, &perms)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

// SetPermissions sets the provided permissions on the object
func (o *Object) SetPermissions(ctx context.Context, perms FolderPermission) error {
	if !o.IsFolder {
		return FolderRequired
	}
	// Check all permission strings
	for _, perms := range []map[string]string{perms.UserPerms, perms.GroupPerms} {
		for name, permissionString := range perms {
			if !isValidPermission(permissionString) && permissionString != "None" {
				return errors.New(fmt.Sprintf("%s is not a valid permission for %s", permissionString, name))
			}
		}
	}

	uri := path.Join(URI_PERMISSIONS, o.Path)
	opts := &requestOptions{
		Method: "POST",
		Path:   uri,
	}
	_, err := o.Client.doRequest(ctx, opts, &perms, nil)
	if err != nil {
		return err
	}
	return nil
}

// SetUserPermission sets a permission for a single user on the object
func (o *Object) SetUserPermission(ctx context.Context, userName string, permString string) error {
	if !o.IsFolder {
		return FolderRequired
	}
	perms := FolderPermission{
		UserPerms: map[string]string{
			userName: permString,
		},
	}
	return o.SetPermissions(ctx, perms)
}

// RemoveUserPermission removes permissions for the given user on the object
func (o *Object) RemoveUserPermission(ctx context.Context, userName string) error {
	if !o.IsFolder {
		return FolderRequired
	}
	perms := FolderPermission{
		UserPerms: map[string]string{
			userName: "None",
		},
	}
	return o.SetPermissions(ctx, perms)
}

// SetGroupPermission sets a permission for a single group on the object
func (o *Object) SetGroupPermission(ctx context.Context, groupName string, permString string) error {
	if !o.IsFolder {
		return FolderRequired
	}
	perms := FolderPermission{
		GroupPerms: map[string]string{
			groupName: permString,
		},
	}
	return o.SetPermissions(ctx, perms)
}

// RemoveGroupPermission removes permissions for the given group on the object
func (o *Object) RemoveGroupPermission(ctx context.Context, groupName string) error {
	if !o.IsFolder {
		return FolderRequired
	}
	perms := FolderPermission{
		GroupPerms: map[string]string{
			groupName: "None",
		},
	}
	return o.SetPermissions(ctx, perms)
}
