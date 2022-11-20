package egnyte

import (
	"context"
	"net/url"
	"path"
	"strconv"
)

// Egnyte Groups Management APIs

// ListGroups lists all the groups in the domain. Note that this returns only custom
// groups. Egnyte default groups (i.e. 'All Power Users', 'All Standard Users and
// Power Users', and 'All Standard Users') are not returned.
// Returns all groups for this domain by iterating through all available pages
func (c *Client) ListGroups(ctx context.Context) ([]*Group, error) {
	var groups []*Group
	startIndex := 1
	itemsPerPage := 100
	for {
		params := url.Values{}
		params.Set("startIndex", strconv.Itoa(startIndex))
		params.Set("count", strconv.Itoa(itemsPerPage))
		opts := &requestOptions{
			Method:     "GET",
			Path:       URI_GROUPS,
			Parameters: params,
		}
		var groupsResp *listGroupResponse
		_, err := c.doRequest(ctx, opts, nil, &groupsResp)
		if err != nil {
			return nil, err
		}
		groups = append(groups, groupsResp.Resources...)
		if groupsResp.StartIndex+groupsResp.ItemsPerPage-1 >= groupsResp.TotalResults {
			break
		}
		startIndex += itemsPerPage
	}
	return groups, nil
}

// GetGroup returns the group attributes along with the list of users which are
// in the group for the provided group ID
func (c *Client) GetGroup(ctx context.Context, groupId string) (*Group, error) {
	uri := path.Join(URI_GROUPS, groupId)
	opts := &requestOptions{
		Method: "GET",
		Path:   uri,
	}
	var group *Group
	_, err := c.doRequest(ctx, opts, nil, &group)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// CreateGroup creates a group
func (c *Client) CreateGroup(ctx context.Context, name string, members []*GroupMember) (*Group, error) {
	opts := &requestOptions{
		Method: "POST",
		Path:   URI_GROUPS,
	}
	groupReq := &createGroupRequest{
		DisplayName: name,
		Members:     members,
	}
	var groupResp *Group
	_, err := c.doRequest(ctx, opts, &groupReq, &groupResp)
	if err != nil {
		return nil, err
	}
	return groupResp, nil
}

// TODO: UpdateGroup
func (c *Client) UpdateGroup(ctx context.Context, name string, members []*GroupMember) (*Group, error) {
	return nil, nil
}

// TODO: UpdateGroup
func (c *Client) PartialUpdateGroup(ctx context.Context, name string, members []*GroupMember) (*Group, error) {
	return nil, nil
}

// DeleteGroup deletes the group
func (c *Client) DeleteGroup(ctx context.Context, groupId string) error {
	uri := path.Join(URI_GROUPS, groupId)
	opts := &requestOptions{
		Method: "DELETE",
		Path:   uri,
	}
	_, err := c.doRequest(ctx, opts, nil, nil)
	return err
}
