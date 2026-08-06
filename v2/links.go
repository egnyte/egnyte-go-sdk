package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// LinksService handles creating, listing, retrieving and deleting
// shareable file and folder links.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/links-api
// OAuth scope: Egnyte.link. Non-administrators can list only links they
// created; administrators can list links across the domain.
type LinksService service

// Link describes an existing link.
type Link struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Path          string   `json:"path"`
	Type          string   `json:"type"`          // file, folder or upload
	Accessibility string   `json:"accessibility"` // anyone, password, domain or recipients
	Protection    string   `json:"protection"`    // PREVIEW for preview-only links, otherwise NONE
	Recipients    []string `json:"recipients"`
	Notify        bool     `json:"notify"`
	LinkToCurrent bool     `json:"link_to_current"`
	CreationDate  string   `json:"creation_date"`
	CreatedBy     string   `json:"created_by"`
	ResourceID    string   `json:"resource_id"`
	ExpiryClicks  int      `json:"expiry_clicks"`
	ExpiryDate    string   `json:"expiry_date"`
	LastAccessed  string   `json:"last_accessed"`
}

// LinkList is the v1 listing: link IDs plus a paging window.
type LinkList struct {
	IDs        []string `json:"ids"`
	Offset     int      `json:"offset"`
	Count      int      `json:"count"`
	TotalCount int      `json:"total_count"`
}

// LinkListV2 is the v2 listing: full link objects.
type LinkListV2 struct {
	Links []Link `json:"links"`
	Count int    `json:"count"`
}

// LinkListOptions filter link listings. Zero values are omitted.
type LinkListOptions struct {
	// Path returns only links pointing to this file or folder.
	Path string
	// Username returns only links created by this user.
	Username string
	// CreatedBefore / CreatedAfter bound the creation date (ISO-8601).
	CreatedBefore string
	CreatedAfter  string
	// Type filters by link type: file or folder.
	Type string
	// Accessibility filters by access level: anyone, password, domain
	// or recipients.
	Accessibility string
	// Offset is the zero-based index of the first link to return.
	Offset int
	// Count caps the number of links returned (default and max 500).
	Count int
}

func (o *LinkListOptions) values() url.Values {
	v := url.Values{}
	if o == nil {
		return v
	}
	setString := func(name, val string) {
		if val != "" {
			v.Set(name, val)
		}
	}
	setString("path", o.Path)
	setString("username", o.Username)
	setString("created_before", o.CreatedBefore)
	setString("created_after", o.CreatedAfter)
	setString("type", o.Type)
	setString("accessibility", o.Accessibility)
	if o.Offset > 0 {
		v.Set("offset", strconv.Itoa(o.Offset))
	}
	if o.Count > 0 {
		v.Set("count", strconv.Itoa(o.Count))
	}
	return v
}

// CreateLinkRequest is the body for creating a link. Path and Type are
// required; optional booleans use pointers so that false can be sent
// explicitly (see the Bool helper).
type CreateLinkRequest struct {
	Path               string   `json:"path"`
	Type               string   `json:"type"`                    // file, folder or upload
	Accessibility      string   `json:"accessibility,omitempty"` // anyone, password, domain or recipients
	UseDefaultSettings *bool    `json:"useDefaultSettings,omitempty"`
	SendEmail          *bool    `json:"send_email,omitempty"`
	Recipients         []string `json:"recipients,omitempty"` // required if SendEmail is true
	Message            string   `json:"message,omitempty"`
	CopyMe             *bool    `json:"copy_me,omitempty"`
	Notify             *bool    `json:"notify,omitempty"`
	LinkToCurrent      *bool    `json:"link_to_current,omitempty"`
	ExpiryDate         string   `json:"expiry_date,omitempty"` // YYYY-MM-DD
	ExpiryClicks       int      `json:"expiry_clicks,omitempty"`
	AddFileName        *bool    `json:"add_file_name,omitempty"`
	Password           string   `json:"password,omitempty"`
	Protection         string   `json:"protection,omitempty"` // PREVIEW or NONE
	FolderPerRecipient *bool    `json:"folder_per_recipient,omitempty"`
}

// String implements fmt.Stringer with the link password redacted, so
// logging the request (including with %+v) never leaks it. JSON
// encoding is unaffected.
func (r CreateLinkRequest) String() string {
	type plain CreateLinkRequest
	c := plain(r)
	if c.Password != "" {
		c.Password = redacted
	}
	return fmt.Sprintf("egnyte.CreateLinkRequest%+v", c)
}

// GoString implements fmt.GoStringer (%#v) with the password redacted.
func (r CreateLinkRequest) GoString() string { return r.String() }

// CreatedLink is one link produced by a create call (one per recipient
// for recipient-scoped links).
type CreatedLink struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Recipients []string `json:"recipients"`
}

// CreateLinkResult is the response to a create call.
type CreateLinkResult struct {
	Links         []CreatedLink `json:"links"`
	Path          string        `json:"path"`
	Type          string        `json:"type"`
	Accessibility string        `json:"accessibility"`
	Notify        bool          `json:"notify"`
	LinkToCurrent bool          `json:"link_to_current"`
	CreationDate  string        `json:"creation_date"`
	CreatedBy     string        `json:"created_by"`
	ExpiryDate    string        `json:"expiry_date"`
	ExpiryClicks  int           `json:"expiry_clicks"`
}

// List returns link IDs visible to the user (administrators see all
// links, other users only their own).
//
// GET /pubapi/v1/links
func (s *LinksService) List(ctx context.Context, opts *LinkListOptions) (*LinkList, *Response, error) {
	list := new(LinkList)
	resp, err := s.list(ctx, "v1", opts, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// ListV2 returns links with full details in a single response.
//
// GET /pubapi/v2/links
func (s *LinksService) ListV2(ctx context.Context, opts *LinkListOptions) (*LinkListV2, *Response, error) {
	list := new(LinkListV2)
	resp, err := s.list(ctx, "v2", opts, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

func (s *LinksService) list(ctx context.Context, version string, opts *LinkListOptions, v any) (*Response, error) {
	urlPath := version + "/links"
	if q := opts.values().Encode(); q != "" {
		urlPath += "?" + q
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, v)
}

// Create creates a new link for a file or folder.
//
// POST /pubapi/v1/links
func (s *LinksService) Create(ctx context.Context, link CreateLinkRequest) (*CreateLinkResult, *Response, error) {
	return s.create(ctx, "v1", link)
}

// CreateV2 is Create against the v2 endpoint.
//
// POST /pubapi/v2/links
func (s *LinksService) CreateV2(ctx context.Context, link CreateLinkRequest) (*CreateLinkResult, *Response, error) {
	return s.create(ctx, "v2", link)
}

func (s *LinksService) create(ctx context.Context, version string, link CreateLinkRequest) (*CreateLinkResult, *Response, error) {
	if link.Path == "" || link.Type == "" {
		return nil, nil, fmt.Errorf("egnyte: link path and type are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, version+"/links", link)
	if err != nil {
		return nil, nil, err
	}
	result := new(CreateLinkResult)
	resp, err := s.client.Do(req, result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// Get returns details of the link with the given id. Only administrators
// or the link's creator can retrieve it.
//
// GET /pubapi/v1/links/{linkId}
func (s *LinksService) Get(ctx context.Context, id string) (*Link, *Response, error) {
	return s.get(ctx, "v1", id)
}

// GetV2 is Get against the v2 endpoint.
//
// GET /pubapi/v2/links/{linkId}
func (s *LinksService) GetV2(ctx context.Context, id string) (*Link, *Response, error) {
	return s.get(ctx, "v2", id)
}

func (s *LinksService) get(ctx context.Context, version, id string) (*Link, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, version+"/links/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	link := new(Link)
	resp, err := s.client.Do(req, link)
	if err != nil {
		return nil, resp, err
	}
	return link, resp, nil
}

// Delete removes the link with the given id. Only administrators or the
// link's creator can delete it.
//
// DELETE /pubapi/v1/links/{linkId}
func (s *LinksService) Delete(ctx context.Context, id string) (*Response, error) {
	return s.delete(ctx, "v1", id)
}

// DeleteV2 is Delete against the v2 endpoint.
//
// DELETE /pubapi/v2/links/{linkId}
func (s *LinksService) DeleteV2(ctx context.Context, id string) (*Response, error) {
	return s.delete(ctx, "v2", id)
}

func (s *LinksService) delete(ctx context.Context, version, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, version+"/links/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
