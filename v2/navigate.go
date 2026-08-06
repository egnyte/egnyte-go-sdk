package egnyte

import (
	"context"
	"net/http"
)

// NavigateService (formerly the Embedded UI API) generates one-time URLs
// that open the Egnyte Web UI, optionally as an embedded view. Requires
// the Egnyte.launchwebsession scope.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/navigate-api
// OAuth scope: Egnyte.launchwebsession.
type NavigateService service

// Navigate returns a one-time URL that opens the Egnyte Web UI. With
// embedded true the view omits headers, search and the folder tree. path
// optionally opens a specific folder (defaults to the user's home
// folder). The URL is single-use; reusing it returns 404.
//
// POST /pubapi/v2/navigate
func (s *NavigateService) Navigate(ctx context.Context, embedded bool, path string) (string, *Response, error) {
	body := struct {
		Embedded bool   `json:"embedded"`
		Path     string `json:"path,omitempty"`
	}{embedded, path}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v2/navigate", body)
	if err != nil {
		return "", nil, err
	}
	var result struct {
		Redirect string `json:"redirect"`
	}
	resp, err := s.client.Do(req, &result)
	if err != nil {
		return "", resp, err
	}
	return result.Redirect, resp, nil
}

// NavigateV1 returns a one-time redirect URL for the given scope (e.g.
// "home" or "folder/Shared/test") from the Location header of the v1
// endpoint's 303 response.
//
// Deprecated: use Navigate (v2), which returns the URL as JSON.
//
// POST /pubapi/v1/navigate/embedded/{scope}
func (s *NavigateService) NavigateV1(ctx context.Context, scope string) (string, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/navigate/embedded/"+EncodePath(scope), nil)
	if err != nil {
		return "", nil, err
	}
	// The redirect URL is the payload here — it must be read from the
	// 303 response, not followed.
	hc := *s.client.httpClient
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	rawResp, err := hc.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer rawResp.Body.Close()
	resp := newResponse(rawResp)
	if rawResp.StatusCode >= 400 {
		return "", resp, newAPIError(rawResp)
	}
	return rawResp.Header.Get("Location"), resp, nil
}
