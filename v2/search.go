package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"
)

// SearchService finds files and folders by filename, content and
// metadata. Results are scoped to what the authenticated user may access.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/search-api
// OAuth scope: Egnyte.filesystem. Results are limited to content accessible
// to the authenticated user.
type SearchService service

// CustomProperty is one namespace/key/value tuple attached to a result.
type CustomProperty struct {
	Scope              string `json:"scope"`
	Namespace          string `json:"namespace"`
	Key                string `json:"key"`
	Value              string `json:"value"`
	ValueToStampOnFile string `json:"valueToStampOnFile"`
}

// SearchResult is one matching file or folder.
type SearchResult struct {
	Name               string           `json:"name"`
	Path               string           `json:"path"`
	Type               string           `json:"type"` // MIME type
	Size               int64            `json:"size"`
	Snippet            string           `json:"snippet"`
	SnippetHTML        string           `json:"snippet_html"`
	EntryID            string           `json:"entry_id"`
	GroupID            string           `json:"group_id"`
	LastModified       string           `json:"last_modified"`
	UploadedBy         string           `json:"uploaded_by"`
	UploadedByUsername string           `json:"uploaded_by_username"`
	NumVersions        int              `json:"num_versions"`
	IsFolder           bool             `json:"is_folder"`
	Score              float64          `json:"score"`
	CustomProperties   []CustomProperty `json:"custom_properties"`
}

// SearchResults is the v1 search response.
type SearchResults struct {
	Results    []SearchResult `json:"results"`
	TotalCount int            `json:"total_count"`
	Offset     int            `json:"offset"`
	Count      int            `json:"count"`
}

// SearchResultsV2 is the v2 search response.
type SearchResultsV2 struct {
	Count      int            `json:"count"`
	Offset     int            `json:"offset"`
	TotalCount int            `json:"total_count"`
	HasMore    bool           `json:"hasMore"`
	Results    []SearchResult `json:"results"`
}

// SearchOptions refine a v1 keyword search. Zero values are omitted.
type SearchOptions struct {
	// Offset is the 0-based index of the first result.
	Offset int
	// Count is the page size (1-20).
	Count int
	// Folder limits the search to a folder and its descendants.
	Folder string
	// ModifiedBefore/After and UploadedBefore/After are ISO-8601
	// timestamps bounding the results.
	ModifiedBefore string
	ModifiedAfter  string
	UploadedBefore string
	UploadedAfter  string
	// Type restricts results to FILE or FOLDER objects.
	Type string
	// SnippetRequested toggles content snippets (server default true).
	SnippetRequested *bool
	// SortBy is one of: last_modified, size, name, score.
	SortBy string
	// SortDirection is "ascending" or "descending".
	SortDirection string
	// FileQueryFields is one of: ALL, FILENAME, COMMENTS, CONTENT.
	FileQueryFields string
	// FolderQueryFields is one of: ALL, FOLDERNAME, DESCRIPTION.
	FolderQueryFields string
	// QueryOperator matches ANY word (default) or ALL words.
	QueryOperator string
}

func (o *SearchOptions) values() url.Values {
	v := url.Values{}
	if o == nil {
		return v
	}
	if o.Offset > 0 {
		v.Set("offset", strconv.Itoa(o.Offset))
	}
	if o.Count > 0 {
		v.Set("count", strconv.Itoa(o.Count))
	}
	setString := func(name, val string) {
		if val != "" {
			v.Set(name, val)
		}
	}
	setString("folder", o.Folder)
	setString("modified_before", o.ModifiedBefore)
	setString("modified_after", o.ModifiedAfter)
	setString("uploaded_before", o.UploadedBefore)
	setString("uploaded_after", o.UploadedAfter)
	setString("type", o.Type)
	if o.SnippetRequested != nil {
		v.Set("snippet_requested", strconv.FormatBool(*o.SnippetRequested))
	}
	setString("sort_by", o.SortBy)
	setString("sort_direction", o.SortDirection)
	setString("file_query_fields", o.FileQueryFields)
	setString("folder_query_fields", o.FolderQueryFields)
	setString("query_operator", o.QueryOperator)
	return v
}

// MetadataRange bounds a BETWEEN metadata filter.
type MetadataRange struct {
	Start any `json:"start,omitempty"`
	End   any `json:"end,omitempty"`
}

// MetadataFilter is one custom-metadata predicate for v2 search.
// Operator is one of EQUALS, GREATER_THAN, LESS_THAN, BETWEEN, IN,
// HAS_KEY, HAS_NOT_KEY, CONTAINS, SUB_SET. Depending on the operator the
// predicate uses Value (single value), Range (BETWEEN) or Values (IN,
// SUB_SET).
type MetadataFilter struct {
	Namespace string         `json:"namespace"`
	Key       string         `json:"key"`
	Operator  string         `json:"operator"`
	Value     any            `json:"value,omitempty"`
	Range     *MetadataRange `json:"range,omitempty"`
	Values    []any          `json:"values,omitempty"`
}

// SearchRequestV2 is the body of a v2 search. At least one of Query or
// CustomMetadata is required. Timestamps are Unix milliseconds.
type SearchRequestV2 struct {
	Query             string   `json:"query,omitempty"` // 3-100 characters
	Offset            int      `json:"offset,omitempty"`
	Count             int      `json:"count,omitempty"` // 1-20
	Folder            string   `json:"folder,omitempty"`
	ModifiedBefore    int64    `json:"modified_before,omitempty"`
	ModifiedAfter     int64    `json:"modified_after,omitempty"`
	UploadedBefore    int64    `json:"uploaded_before,omitempty"`
	UploadedAfter     int64    `json:"uploaded_after,omitempty"`
	Type              string   `json:"type,omitempty"` // FILE, FOLDER or ALL
	SnippetRequested  *bool    `json:"snippet_requested,omitempty"`
	SortBy            string   `json:"sort_by,omitempty"`
	SortDirection     string   `json:"sort_direction,omitempty"`
	QueryOperator     string   `json:"query_operator,omitempty"`
	FileQueryFields   []string `json:"file_query_fields,omitempty"`
	FolderQueryFields []string `json:"folder_query_fields,omitempty"`
	// MoreLikeThese holds document entry IDs used as similarity references.
	MoreLikeThese []string `json:"mlt,omitempty"`
	// MoreLikeTexts holds text strings used as similarity references.
	MoreLikeTexts  []string         `json:"mltt,omitempty"`
	Namespaces     []string         `json:"namespaces,omitempty"`
	CustomMetadata []MetadataFilter `json:"custom_metadata,omitempty"`
}

// Search performs a v1 keyword search. query must be 3-100 characters.
//
// GET /pubapi/v1/search
func (s *SearchService) Search(ctx context.Context, query string, opts *SearchOptions) (*SearchResults, *Response, error) {
	// Characters, not bytes: a non-ASCII query would otherwise be
	// rejected (or accepted) at the wrong length.
	if n := utf8.RuneCountInString(query); n < 3 || n > 100 {
		return nil, nil, fmt.Errorf("egnyte: search query must be 3-100 characters, got %d", n)
	}
	q := opts.values()
	q.Set("query", query)
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/search?"+q.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}
	results := new(SearchResults)
	resp, err := s.client.Do(req, results)
	if err != nil {
		return nil, resp, err
	}
	return results, resp, nil
}

// SearchV2 performs a v2 metadata-aware search.
//
// POST /pubapi/v2/search
func (s *SearchService) SearchV2(ctx context.Context, search SearchRequestV2) (*SearchResultsV2, *Response, error) {
	if search.Query == "" && len(search.CustomMetadata) == 0 &&
		len(search.MoreLikeThese) == 0 && len(search.MoreLikeTexts) == 0 {
		return nil, nil, fmt.Errorf("egnyte: v2 search requires a query, custom metadata filters, or similarity references")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/search", search)
	if err != nil {
		return nil, nil, err
	}
	results := new(SearchResultsV2)
	resp, err := s.client.Do(req, results)
	if err != nil {
		return nil, resp, err
	}
	return results, resp, nil
}
