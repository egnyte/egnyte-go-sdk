package egnyte

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrorDetail is a single entry of the "Errors" array returned by the API.
type ErrorDetail struct {
	Description string `json:"description"`
	Code        string `json:"code"`
}

// APIError is returned for responses with status >= 400.
//
// The Egnyte API reports errors either as {"Errors": [...]} or as a flat
// {"message": "..."} object; both are captured here. Rate-limited requests
// (429, or 409 on the OAuth endpoint) carry a Retry-After duration.
type APIError struct {
	StatusCode int
	Status     string
	Method     string
	URL        string
	Errors     []ErrorDetail
	Message    string
	// RetryAfter is the server-suggested wait before retrying, zero if
	// the response carried no Retry-After header.
	RetryAfter time.Duration
	// Body is the raw response body, useful when it matched no known
	// error shape.
	Body []byte
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "egnyte: %s %s: %s", e.Method, e.URL, e.Status)
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	for _, d := range e.Errors {
		fmt.Fprintf(&b, ": %s", d.Description)
	}
	return b.String()
}

// IsRateLimit reports whether the error indicates a rate-limited request.
//
// The OAuth token endpoint signals throttling with 409 rather than 429;
// elsewhere 409 means a genuine conflict (for example a folder that
// already exists), so the 409 case is scoped to that endpoint to avoid
// misclassifying ordinary conflicts as rate limits.
func (e *APIError) IsRateLimit() bool {
	if e.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return e.StatusCode == http.StatusConflict && e.RetryAfter > 0 && e.isOAuthEndpoint()
}

func (e *APIError) isOAuthEndpoint() bool {
	if e.URL == "" {
		return false
	}
	u, err := url.Parse(e.URL)
	if err != nil {
		return false
	}
	return u.Path == oauthTokenPath
}

func newAPIError(resp *http.Response) *APIError {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}
	if resp.Request != nil {
		apiErr.Method = resp.Request.Method
		apiErr.URL = resp.Request.URL.String()
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			apiErr.RetryAfter = time.Duration(secs) * time.Second
		}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	apiErr.Body = body

	// The API uses several error envelopes, all observed in the wild:
	// {"Errors":[{description,code}]}, {"message":...},
	// {"errorMessage":...}, {"formErrors":[{code,msg}]},
	// {"errors":[{msg,code}]} and {"inputErrors":{field:[{msg,code}]}}.
	// Capture them all.
	type msgCode struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	var envelope struct {
		Errors       []ErrorDetail        `json:"Errors"`
		Message      string               `json:"message"`
		ErrorMessage string               `json:"errorMessage"`
		FormErrors   []msgCode            `json:"formErrors"`
		LowerErrors  []msgCode            `json:"errors"`
		InputErrors  map[string][]msgCode `json:"inputErrors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		apiErr.Errors = envelope.Errors
		apiErr.Message = envelope.Message
		if apiErr.Message == "" {
			apiErr.Message = envelope.ErrorMessage
		}
		for _, e := range envelope.FormErrors {
			apiErr.Errors = append(apiErr.Errors, ErrorDetail{Description: e.Msg, Code: e.Code})
		}
		for _, e := range envelope.LowerErrors {
			apiErr.Errors = append(apiErr.Errors, ErrorDetail{Description: e.Msg, Code: e.Code})
		}
		for field, errs := range envelope.InputErrors {
			for _, e := range errs {
				apiErr.Errors = append(apiErr.Errors, ErrorDetail{
					Description: field + ": " + e.Msg, Code: e.Code,
				})
			}
		}
	}
	return apiErr
}
