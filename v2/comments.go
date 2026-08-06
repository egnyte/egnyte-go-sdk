package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// CommentsService annotates files and folders with comments (also called
// notes — the API path is /v1/notes).
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/comments-api
// OAuth scope: Egnyte.filesystem. Listing comments across the entire domain
// requires an administrator; file-scoped operations follow file permissions.
type CommentsService service

// Comment is one comment (note) on a file or folder.
type Comment struct {
	ID            string `json:"id"`
	Message       string `json:"message"`
	Username      string `json:"username"`
	CanDelete     bool   `json:"can_delete"`
	CreationTime  string `json:"creation_time"`
	FormattedName string `json:"formatted_name"`
	FilePath      string `json:"file_path"`
	GroupID       string `json:"group_id"`
}

// CommentList is a page of comments.
type CommentList struct {
	TotalResults int       `json:"total_results"`
	Count        int       `json:"count"`
	Offset       int       `json:"offset"`
	Notes        []Comment `json:"notes"`
}

// CommentListOptions filter comment listings. Zero values are omitted.
type CommentListOptions struct {
	// File limits results to one file or folder path. When empty, all
	// comments in the domain are returned (administrators only).
	File string
	// StartTime / EndTime bound the creation time (ISO-8601).
	StartTime string
	EndTime   string
	// Count is the page size (max 100, default 25).
	Count int
	// Offset is the zero-based index of the first result.
	Offset int
}

// Add creates a comment on the file or folder at path.
//
// POST /pubapi/v1/notes
func (s *CommentsService) Add(ctx context.Context, path, body string) (*Comment, *Response, error) {
	if path == "" || body == "" {
		return nil, nil, fmt.Errorf("egnyte: comment path and body are required")
	}
	payload := struct {
		Path string `json:"path"`
		Body string `json:"body"`
	}{path, body}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/notes", payload)
	if err != nil {
		return nil, nil, err
	}
	comment := new(Comment)
	resp, err := s.client.Do(req, comment)
	if err != nil {
		return nil, resp, err
	}
	return comment, resp, nil
}

// List returns comments for a file, or for the whole domain when
// opts.File is empty (administrators only).
//
// GET /pubapi/v1/notes
func (s *CommentsService) List(ctx context.Context, opts *CommentListOptions) (*CommentList, *Response, error) {
	urlPath := "v1/notes"
	if opts != nil {
		q := url.Values{}
		if opts.File != "" {
			q.Set("file", opts.File)
		}
		if opts.StartTime != "" {
			q.Set("start_time", opts.StartTime)
		}
		if opts.EndTime != "" {
			q.Set("end_time", opts.EndTime)
		}
		if opts.Count > 0 {
			q.Set("count", strconv.Itoa(opts.Count))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
		if enc := q.Encode(); enc != "" {
			urlPath += "?" + enc
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(CommentList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// Get returns the comment with the given id.
//
// GET /pubapi/v1/notes/{UUID}
func (s *CommentsService) Get(ctx context.Context, id string) (*Comment, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/notes/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	comment := new(Comment)
	resp, err := s.client.Do(req, comment)
	if err != nil {
		return nil, resp, err
	}
	return comment, resp, nil
}

// Delete removes the comment with the given id.
//
// DELETE /pubapi/v1/notes/{UUID}
func (s *CommentsService) Delete(ctx context.Context, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v1/notes/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
