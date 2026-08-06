package egnyte

import (
	"context"
	"net/http"
	"net/url"
)

// InsightsService returns activity insights for the authenticated user.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/user-insights-api
// OAuth scope: Egnyte.filesystem.
type InsightsService service

// RecentFile is one recently accessed file.
type RecentFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	EntryID     string `json:"entry_id"`
	GroupID     string `json:"group_id"`
	UploadedBy  string `json:"uploaded_by"`
	NumVersions int    `json:"num_versions"`
	// LastModified and LastAccessed are Unix milliseconds.
	LastModified int64 `json:"last_modified"`
	LastAccessed int64 `json:"last_accessed"`
	// RecommendationType is always "recent" for this endpoint.
	RecommendationType string `json:"recommendation_type"`
}

// RecentFiles returns up to 10 files the user recently accessed,
// optionally scoped to folderPath ("" for the whole domain).
//
// GET /pubapi/v1/insights/files
func (s *InsightsService) RecentFiles(ctx context.Context, folderPath string) ([]RecentFile, *Response, error) {
	urlPath := "v1/insights/files"
	if folderPath != "" {
		urlPath += "?" + url.Values{"folder_path": {folderPath}}.Encode()
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	var body struct {
		RecentFiles []RecentFile `json:"recentFiles"`
	}
	resp, err := s.client.Do(req, &body)
	if err != nil {
		return nil, resp, err
	}
	return body.RecentFiles, resp, nil
}
