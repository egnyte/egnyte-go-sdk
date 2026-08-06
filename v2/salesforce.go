package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// SalesforceService looks up the Egnyte folders mapped to Salesforce
// records (read-only). Requires the Egnyte.salesforce scope.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/third-party-integrations#egnyte-for-salesforce
// OAuth scope: Egnyte.salesforce.
type SalesforceService service

// SalesforceFolderMap is the Egnyte folder mapped to a Salesforce record.
type SalesforceFolderMap struct {
	// Path is the folder path; when it does not start with "/", treat
	// it as relative to /Shared/Salesforce.com.
	Path     string `json:"path"`
	FolderID string `json:"folder_id"`
}

// FolderMap returns the folder mapped to the Salesforce record with the
// given 18-character record id.
//
// GET /pubapi/v1/sfdc/foldermap/{recordId}
func (s *SalesforceService) FolderMap(ctx context.Context, recordID string) (*SalesforceFolderMap, *Response, error) {
	if len(recordID) != 18 {
		return nil, nil, fmt.Errorf("egnyte: salesforce record id must be 18 characters, got %d", len(recordID))
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/sfdc/foldermap/"+url.PathEscape(recordID), nil)
	if err != nil {
		return nil, nil, err
	}
	folderMap := new(SalesforceFolderMap)
	resp, err := s.client.Do(req, folderMap)
	if err != nil {
		return nil, resp, err
	}
	return folderMap, resp, nil
}
