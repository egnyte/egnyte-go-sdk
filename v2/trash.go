package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// TrashService lists, restores and permanently deletes trashed items.
// Requires an administrator or a power user with trash management
// privileges.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/trash-api
// OAuth scope: Egnyte.filesystem. Access additionally requires an
// administrator or a power user with trash management privileges.
type TrashService service

// TrashItemOwner identifies the owner of a trashed file.
type TrashItemOwner struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	TypeName  string `json:"typeName"`
}

// TrashRetentionInfo describes data retention holding a trashed item.
type TrashRetentionInfo struct {
	// Retention is RETENTION_POLICY, GRACE_PERIOD or DEFAULT_RETENTION_PERIOD.
	Retention string `json:"retention"`
	// EarliestExpiry is a Unix timestamp of when retention expires.
	EarliestExpiry string `json:"earliest_expiry"`
	IsRetained     bool   `json:"is_retained"`
}

// TrashItem is one entry in the trash.
type TrashItem struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // file, folder or version
	Path         string `json:"path"` // original path before deletion
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	DeletedBy    string `json:"deleted_by"`
	DeleteDate   string `json:"delete_date"`
	PurgeDate    string `json:"purge_date"`
	LastModified string `json:"last_modified"`
	// Owner is set for files only.
	Owner             *TrashItemOwner     `json:"owner"`
	DataRetentionInfo *TrashRetentionInfo `json:"data_retention_info"`
}

// TrashList is the deprecated v1 listing with a total count.
type TrashList struct {
	Count      int         `json:"count"`
	Offset     int         `json:"offset"`
	TotalCount int         `json:"total_count"`
	Items      []TrashItem `json:"items"`
}

// TrashListV2 is the v2 listing; for the overall number of items call
// TrashService.TotalCount separately.
type TrashListV2 struct {
	Count   int         `json:"count"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"has_more"`
	Items   []TrashItem `json:"items"`
}

// TrashListOptions control sorting and paging of trash listings.
type TrashListOptions struct {
	// SortBy is date_deleted (default), name, deleted_by, purge_date —
	// v2 additionally supports orig_location, owner, type.
	SortBy string
	// SortDirection is "asc" or "desc" (default).
	SortDirection string
	// Count is the page size (default 50).
	Count int
	// Offset is the zero-based index of the first entry.
	Offset int
}

// TrashListV2Options add v2-only filters. DeletedBy, StartDate and
// EndDate must be provided together.
type TrashListV2Options struct {
	TrashListOptions
	// DeletedBy filters by the deleting user's first and/or last name.
	DeletedBy string
	// StartDate / EndDate bound the deletion date (ISO-8601).
	StartDate string
	EndDate   string
}

func (o *TrashListOptions) values() url.Values {
	v := url.Values{}
	if o == nil {
		return v
	}
	if o.SortBy != "" {
		v.Set("sort_by", o.SortBy)
	}
	if o.SortDirection != "" {
		v.Set("sort_direction", o.SortDirection)
	}
	if o.Count > 0 {
		v.Set("count", strconv.Itoa(o.Count))
	}
	if o.Offset > 0 {
		v.Set("offset", strconv.Itoa(o.Offset))
	}
	return v
}

// TrashActionResource is the per-item outcome of a restore or purge.
type TrashActionResource struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	// Descriptions carries the error description when Code is not 200.
	Descriptions string `json:"descriptions"`
}

// TrashActionResult reports per-item outcomes. Resources is empty when
// every item succeeded (HTTP 200); on partial failure (HTTP 207) it
// lists each item's status.
type TrashActionResult struct {
	Resources []TrashActionResource `json:"resources"`
}

// List returns trash contents with a total count.
//
// Deprecated: use ListV2 for new integrations.
//
// GET /pubapi/v1/fs/trash
func (s *TrashService) List(ctx context.Context, opts *TrashListOptions) (*TrashList, *Response, error) {
	urlPath := "v1/fs/trash"
	if q := opts.values().Encode(); q != "" {
		urlPath += "?" + q
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(TrashList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// ListV2 returns trash contents with optional user and date filters.
//
// GET /pubapi/v2/fs/trash
func (s *TrashService) ListV2(ctx context.Context, opts *TrashListV2Options) (*TrashListV2, *Response, error) {
	q := url.Values{}
	if opts != nil {
		q = opts.TrashListOptions.values()
		if opts.DeletedBy != "" {
			q.Set("deletedBy", opts.DeletedBy)
		}
		if opts.StartDate != "" {
			q.Set("startDate", opts.StartDate)
		}
		if opts.EndDate != "" {
			q.Set("endDate", opts.EndDate)
		}
	}
	urlPath := "v2/fs/trash"
	if enc := q.Encode(); enc != "" {
		urlPath += "?" + enc
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(TrashListV2)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// TotalCount returns the number of items in the trash without listing
// them.
//
// GET /pubapi/v2/fs/trash/totalcount
func (s *TrashService) TotalCount(ctx context.Context) (int, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v2/fs/trash/totalcount", nil)
	if err != nil {
		return 0, nil, err
	}
	var body struct {
		TotalCount int `json:"totalCount"`
	}
	resp, err := s.client.Do(req, &body)
	if err != nil {
		return 0, resp, err
	}
	return body.TotalCount, resp, nil
}

// Restore returns up to 10 trashed items to their original locations.
// Check the result (and resp.StatusCode 207) for per-item failures.
//
// POST /pubapi/v1/fs/trash
func (s *TrashService) Restore(ctx context.Context, ids []string) (*TrashActionResult, *Response, error) {
	return s.action(ctx, "RESTORE", ids)
}

// Purge permanently deletes up to 10 trashed items. Purged items cannot
// be recovered.
//
// POST /pubapi/v1/fs/trash
func (s *TrashService) Purge(ctx context.Context, ids []string) (*TrashActionResult, *Response, error) {
	return s.action(ctx, "PURGE", ids)
}

func (s *TrashService) action(ctx context.Context, action string, ids []string) (*TrashActionResult, *Response, error) {
	if len(ids) == 0 || len(ids) > 10 {
		return nil, nil, fmt.Errorf("egnyte: trash actions accept 1-10 item ids, got %d", len(ids))
	}
	body := struct {
		Action string   `json:"action"`
		IDs    []string `json:"ids"`
	}{action, ids}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/fs/trash", body)
	if err != nil {
		return nil, nil, err
	}
	result := new(TrashActionResult)
	resp, err := s.client.Do(req, result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}
