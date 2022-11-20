package egnyte

import (
	"context"
	"net/url"
	"path"
	"strconv"
)

// Egnyte User Management APIs

// ListUsers retrieves all, or a chosen subset of users
// Returns all users for this domain by iterating through all available pages
func (c *Client) ListUsers(ctx context.Context) ([]*User, error) {
	var users []*User
	startIndex := 1
	itemsPerPage := 100
	for {
		params := url.Values{}
		params.Set("startIndex", strconv.Itoa(startIndex))
		params.Set("count", strconv.Itoa(itemsPerPage))
		opts := &requestOptions{
			Method:     "GET",
			Path:       URI_USERS,
			Parameters: params,
		}
		var listUsersResp *listUserResponse
		_, err := c.doRequest(ctx, opts, nil, &listUsersResp)
		if err != nil {
			return nil, err
		}
		users = append(users, listUsersResp.Resources...)
		if listUsersResp.StartIndex+listUsersResp.ItemsPerPage-1 >= listUsersResp.TotalResults {
			break
		}
		startIndex += itemsPerPage
	}
	return users, nil
}

// GetUser returns a single user provided user ID
func (c *Client) GetUser(ctx context.Context, userId int) (*User, error) {
	uri := path.Join(URI_USERS, strconv.Itoa(userId))
	opts := &requestOptions{
		Method: "GET",
		Path:   uri,
	}
	var user *User
	_, err := c.doRequest(ctx, opts, nil, &user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser creates a user using the provided User object
func (c *Client) CreateUser(ctx context.Context, user *User, sendInvite bool) (*User, error) {
	opts := &requestOptions{
		Method: "POST",
		Path:   URI_USERS,
	}
	if user.UserType == "" {
		user.UserType = "standard"
	}
	if user.AuthType == "" {
		user.AuthType = "egnyte"
	}
	userRequest := createUserRequest{
		UserName:          user.UserName,
		ExternalID:        user.ExternalID,
		Email:             user.Email,
		Name:              user.Name,
		Active:            user.Active,
		IsServiceAccount:  user.IsServiceAccount,
		Language:          user.Language,
		AuthType:          user.AuthType,
		UserType:          user.UserType,
		Role:              user.Role,
		IdpUserID:         user.IdpUserID,
		UserPrincipalName: user.UserPrincipalName,
		SendInvite:        sendInvite,
	}
	var createdUser *User
	_, err := c.doRequest(ctx, opts, &userRequest, &createdUser)
	if err != nil {
		return nil, err
	}
	return createdUser, nil
}

// DeleteUser deletes an user
func (c *Client) DeleteUser(ctx context.Context, userId int) error {
	uri := path.Join(URI_USERS, strconv.Itoa(userId))
	opts := &requestOptions{
		Method: "DELETE",
		Path:   uri,
	}
	_, err := c.doRequest(ctx, opts, nil, nil)
	return err
}

// UpdateUser updates an user
func (c *Client) UpdateUser(ctx context.Context, user *User) error {
	uri := path.Join(URI_USERS, strconv.Itoa(user.ID))
	opts := &requestOptions{
		Method: "PATCH",
		Path:   uri,
	}
	_, err := c.doRequest(ctx, opts, &user, nil)
	return err
}

// Userinfo fetches the username for the provided domain
func (c *Client) Userinfo(ctx context.Context) (*userInfoResponse, error) {
	opts := &requestOptions{
		Method: "GET",
		Path:   URI_USERINFO,
	}
	var userinfo *userInfoResponse
	_, err := c.doRequest(ctx, opts, nil, &userinfo)
	if err != nil {
		return nil, err
	}
	c.Username = userinfo.Username
	c.Email = userinfo.Email
	return userinfo, nil
}
