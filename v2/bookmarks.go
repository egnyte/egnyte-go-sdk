package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// BookmarksService marks folders for quick access in the Web UI.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/bookmarks-api
// OAuth scope: Egnyte.bookmark.
type BookmarksService service

// Bookmark marks one folder.
type Bookmark struct {
	ID           int    `json:"id"`
	Path         string `json:"path"`
	FolderID     string `json:"folder_id"`
	CreationDate string `json:"creation_date"`
}

// BookmarkList is a page of bookmarks.
type BookmarkList struct {
	Offset    int        `json:"offset"`
	Count     int        `json:"count"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

// BookmarkListOptions paginate bookmark listings.
type BookmarkListOptions struct {
	Offset int
	Count  int
}

// CreateByPath bookmarks the folder at the given path.
//
// POST /pubapi/v1/bookmarks
func (s *BookmarksService) CreateByPath(ctx context.Context, path string) (*Bookmark, *Response, error) {
	if path == "" {
		return nil, nil, fmt.Errorf("egnyte: bookmark path is required")
	}
	return s.create(ctx, struct {
		Path string `json:"path"`
	}{path})
}

// CreateByFolderID bookmarks the folder with the given UUID.
//
// POST /pubapi/v1/bookmarks
func (s *BookmarksService) CreateByFolderID(ctx context.Context, folderID string) (*Bookmark, *Response, error) {
	if folderID == "" {
		return nil, nil, fmt.Errorf("egnyte: bookmark folder_id is required")
	}
	return s.create(ctx, struct {
		FolderID string `json:"folder_id"`
	}{folderID})
}

func (s *BookmarksService) create(ctx context.Context, body any) (*Bookmark, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/bookmarks", body)
	if err != nil {
		return nil, nil, err
	}
	bookmark := new(Bookmark)
	resp, err := s.client.Do(req, bookmark)
	if err != nil {
		return nil, resp, err
	}
	return bookmark, resp, nil
}

// List returns the authenticated user's bookmarks.
//
// GET /pubapi/v1/bookmarks
func (s *BookmarksService) List(ctx context.Context, opts *BookmarkListOptions) (*BookmarkList, *Response, error) {
	urlPath := "v1/bookmarks"
	if opts != nil {
		q := url.Values{}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
		if opts.Count > 0 {
			q.Set("count", strconv.Itoa(opts.Count))
		}
		if enc := q.Encode(); enc != "" {
			urlPath += "?" + enc
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(BookmarkList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// Get returns the bookmark with the given id.
//
// GET /pubapi/v1/bookmarks/{id}
func (s *BookmarksService) Get(ctx context.Context, id int) (*Bookmark, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/bookmarks/"+strconv.Itoa(id), nil)
	if err != nil {
		return nil, nil, err
	}
	bookmark := new(Bookmark)
	resp, err := s.client.Do(req, bookmark)
	if err != nil {
		return nil, resp, err
	}
	return bookmark, resp, nil
}

// Delete removes the bookmark with the given id.
//
// DELETE /pubapi/v1/bookmarks/{id}
func (s *BookmarksService) Delete(ctx context.Context, id int) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v1/bookmarks/"+strconv.Itoa(id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
