package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// UsersService manages users via Egnyte's SCIM-style endpoints. All
// operations require an administrator token.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/user-management-api
// OAuth scope: Egnyte.user.
type UsersService service

// UserName is the SCIM name attribute of a user.
type UserName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	Formatted  string `json:"formatted,omitempty"`
}

// GroupRef references a group a user belongs to.
type GroupRef struct {
	DisplayName string `json:"displayName"`
	Value       string `json:"value"` // group id (UUID)
}

// User is a SCIM user resource.
type User struct {
	ID                   int      `json:"id"`
	UserName             string   `json:"userName"`
	ExternalID           string   `json:"externalId"`
	Email                string   `json:"email"`
	Name                 UserName `json:"name"`
	Active               bool     `json:"active"`
	Locked               bool     `json:"locked"`
	AuthType             string   `json:"authType"`
	UserType             string   `json:"userType"`
	Role                 string   `json:"role"`
	Language             string   `json:"language"`
	IdpUserID            string   `json:"idpUserId"`
	UserPrincipalName    string   `json:"userPrincipalName"`
	CreatedDate          string   `json:"createdDate"`
	LastModificationDate string   `json:"lastModificationDate"`
	LastActiveDate       string   `json:"lastActiveDate"`
	ExpiryDate           string   `json:"expiryDate"`
	// DeleteOnExpiry is returned by the live API either as a boolean or
	// as the string "false"/"true"; nil when the user has no expiry.
	DeleteOnExpiry *LenientBool `json:"deleteOnExpiry"`
	Groups         []GroupRef   `json:"groups"`
}

// UserList is a SCIM list response of users.
type UserList struct {
	TotalResults int    `json:"totalResults"`
	ItemsPerPage int    `json:"itemsPerPage"`
	StartIndex   int    `json:"startIndex"`
	Resources    []User `json:"resources"`
}

// UserListOptions control listing and filtering of users.
type UserListOptions struct {
	// StartIndex is the 1-based index of the first result (default 1).
	StartIndex int
	// Count caps the number of users returned (default and max 100).
	Count int
	// Filter is a SCIM filter expression, e.g. `userName eq "john"`.
	// Supported attributes: userName, email, externalId.
	Filter string
}

func (o *UserListOptions) values() url.Values {
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

// CreateUserRequest is the body for creating a user. UserName, Email,
// Name (given and family), AuthType and UserType are required; Active is
// always sent.
type CreateUserRequest struct {
	UserName          string   `json:"userName"`
	ExternalID        string   `json:"externalId,omitempty"`
	Email             string   `json:"email"`
	Name              UserName `json:"name"`
	Active            bool     `json:"active"`
	SendInvite        *bool    `json:"sendInvite,omitempty"`
	Language          string   `json:"language,omitempty"`
	AuthType          string   `json:"authType"`
	UserType          string   `json:"userType"`
	Role              string   `json:"role,omitempty"`
	IdpUserID         string   `json:"idpUserId,omitempty"`
	UserPrincipalName string   `json:"userPrincipalName,omitempty"`
}

// UpdateUserRequest is the body for a partial user update; only fields
// present are changed.
type UpdateUserRequest struct {
	Email             string `json:"email,omitempty"`
	FamilyName        string `json:"familyName,omitempty"`
	GivenName         string `json:"givenName,omitempty"`
	Active            *bool  `json:"active,omitempty"`
	SendInvite        *bool  `json:"sendInvite,omitempty"`
	Language          string `json:"language,omitempty"`
	AuthType          string `json:"authType,omitempty"`
	UserType          string `json:"userType,omitempty"`
	Role              string `json:"role,omitempty"`
	IdpUserID         string `json:"idpUserId,omitempty"`
	UserPrincipalName string `json:"userPrincipalName,omitempty"`
}

// List returns a paginated list of users.
//
// GET /pubapi/v2/users
func (s *UsersService) List(ctx context.Context, opts *UserListOptions) (*UserList, *Response, error) {
	urlPath := "v2/users"
	if q := opts.values().Encode(); q != "" {
		urlPath += "?" + q
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(UserList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// Create creates a new user.
//
// POST /pubapi/v2/users
func (s *UsersService) Create(ctx context.Context, user CreateUserRequest) (*User, *Response, error) {
	switch {
	case user.UserName == "", user.Email == "",
		user.Name.GivenName == "", user.Name.FamilyName == "",
		user.AuthType == "", user.UserType == "":
		return nil, nil, fmt.Errorf("egnyte: userName, email, name (given and family), authType and userType are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/users", user)
	if err != nil {
		return nil, nil, err
	}
	created := new(User)
	resp, err := s.client.Do(req, created)
	if err != nil {
		return nil, resp, err
	}
	return created, resp, nil
}

// Get returns the user with the given id.
//
// GET /pubapi/v2/users/{id}
func (s *UsersService) Get(ctx context.Context, id int) (*User, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v2/users/"+strconv.Itoa(id), nil)
	if err != nil {
		return nil, nil, err
	}
	user := new(User)
	resp, err := s.client.Do(req, user)
	if err != nil {
		return nil, resp, err
	}
	return user, resp, nil
}

// Update partially updates the user with the given id and returns the
// updated resource.
//
// PATCH /pubapi/v2/users/{id}
func (s *UsersService) Update(ctx context.Context, id int, update UpdateUserRequest) (*User, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPatch, "v2/users/"+strconv.Itoa(id), update)
	if err != nil {
		return nil, nil, err
	}
	user := new(User)
	resp, err := s.client.Do(req, user)
	if err != nil {
		return nil, resp, err
	}
	return user, resp, nil
}

// Delete removes the user with the given id.
//
// DELETE /pubapi/v2/users/{id}
func (s *UsersService) Delete(ctx context.Context, id int) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v2/users/"+strconv.Itoa(id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
