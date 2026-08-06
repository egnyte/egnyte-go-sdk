package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// GroupsService manages custom groups via Egnyte's SCIM 2.0 endpoints.
// All operations require an administrator token.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/group-management
// OAuth scope: Egnyte.group.
type GroupsService service

// scimPatchOpSchema is the SCIM 2.0 schema URN required in PatchOp bodies.
const scimPatchOpSchema = "urn:ietf:params:scim:api:messages:2.0:PatchOp"

// GroupMember is one member of a group.
type GroupMember struct {
	Value   string `json:"value"`   // user id
	Display string `json:"display"` // username
}

// Group is a SCIM group resource.
type Group struct {
	ID          string        `json:"id"`
	DisplayName string        `json:"displayName"`
	Members     []GroupMember `json:"members"`
}

// GroupList is a SCIM list response of groups.
type GroupList struct {
	TotalResults int     `json:"totalResults"`
	ItemsPerPage int     `json:"itemsPerPage"`
	StartIndex   int     `json:"startIndex"`
	Resources    []Group `json:"resources"`
}

// GroupListOptions control listing and filtering of groups.
type GroupListOptions struct {
	// StartIndex is the 1-based index of the first result (default 1).
	StartIndex int
	// Count caps the number of groups returned (default 100).
	Count int
	// Filter is a SCIM filter expression on displayName, e.g.
	// `displayName sw "Mar"`.
	Filter string
}

func (o *GroupListOptions) values() url.Values {
	v := url.Values{}
	if o == nil {
		return v
	}
	if o.StartIndex > 0 {
		v.Set("startIndex", strconv.Itoa(o.StartIndex))
	}
	if o.Count > 0 {
		v.Set("count", strconv.Itoa(o.Count))
	}
	if o.Filter != "" {
		v.Set("filter", o.Filter)
	}
	return v
}

// GroupPatchOperation is one SCIM 2.0 patch operation: add, remove or
// replace an attribute.
type GroupPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}

type groupWriteRequest struct {
	DisplayName string   `json:"displayName"`
	Members     []string `json:"members,omitempty"`
}

// List returns a paginated list of custom groups.
//
// GET /pubapi/v2/groups
func (s *GroupsService) List(ctx context.Context, opts *GroupListOptions) (*GroupList, *Response, error) {
	urlPath := "v2/groups"
	if q := opts.values().Encode(); q != "" {
		urlPath += "?" + q
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(GroupList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// Create creates a new group. memberIDs optionally seeds the membership
// with user ids.
//
// POST /pubapi/v2/groups
func (s *GroupsService) Create(ctx context.Context, displayName string, memberIDs []string) (*Group, *Response, error) {
	if displayName == "" {
		return nil, nil, fmt.Errorf("egnyte: group displayName is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/groups", groupWriteRequest{displayName, memberIDs})
	if err != nil {
		return nil, nil, err
	}
	group := new(Group)
	resp, err := s.client.Do(req, group)
	if err != nil {
		return nil, resp, err
	}
	return group, resp, nil
}

// Get returns the group with the given id.
//
// GET /pubapi/v2/groups/{groupId}
func (s *GroupsService) Get(ctx context.Context, id string) (*Group, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v2/groups/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	group := new(Group)
	resp, err := s.client.Do(req, group)
	if err != nil {
		return nil, resp, err
	}
	return group, resp, nil
}

// Update replaces the group's display name and full membership. Members
// omitted from memberIDs are removed from the group.
//
// PUT /pubapi/v2/groups/{groupId}
func (s *GroupsService) Update(ctx context.Context, id, displayName string, memberIDs []string) (*Group, *Response, error) {
	if displayName == "" {
		return nil, nil, fmt.Errorf("egnyte: group displayName is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPut, "v2/groups/"+url.PathEscape(id), groupWriteRequest{displayName, memberIDs})
	if err != nil {
		return nil, nil, err
	}
	group := new(Group)
	resp, err := s.client.Do(req, group)
	if err != nil {
		return nil, resp, err
	}
	return group, resp, nil
}

// Patch applies SCIM 2.0 patch operations to the group, e.g. renaming it
// or adding/removing members without resending the full membership.
//
// PATCH /pubapi/v2/groups/{groupId}
func (s *GroupsService) Patch(ctx context.Context, id string, ops []GroupPatchOperation) (*Group, *Response, error) {
	if len(ops) == 0 {
		return nil, nil, fmt.Errorf("egnyte: at least one patch operation is required")
	}
	body := struct {
		Schemas    []string              `json:"schemas"`
		Operations []GroupPatchOperation `json:"Operations"`
	}{[]string{scimPatchOpSchema}, ops}
	req, err := s.client.NewRequest(ctx, http.MethodPatch, "v2/groups/"+url.PathEscape(id), body)
	if err != nil {
		return nil, nil, err
	}
	group := new(Group)
	resp, err := s.client.Do(req, group)
	if err != nil {
		return nil, resp, err
	}
	return group, resp, nil
}

// Delete removes the group and all its membership associations.
//
// DELETE /pubapi/v2/groups/{groupId}
func (s *GroupsService) Delete(ctx context.Context, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v2/groups/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
