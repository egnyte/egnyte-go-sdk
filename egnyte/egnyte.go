// Package egnyte provides an API interface to Egnyte APIs
package egnyte

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	// Used to marshal empty slices to [] instead of null
	// https://github.com/golang/go/issues/27589 proposes to update standard
	// lib encoding/json to incorporate similar functionality
	// https://go-review.googlesource.com/c/go/+/205897/ is a proposed
	// implementation for the same
	helloeaveJson "github.com/helloeave/json"
)

// NewClient takes a http.Client, a root URL, and an auth token as
// input and returns a custom client.
func NewClient(ctx context.Context, rootUrl, token string, baseClient *http.Client) (*Client, error) {
	client := &Client{
		Client: *baseClient,
		root:   rootUrl,
		token:  token,
	}

	clientId := client.clientId
	if clientId == "" {
		clientId = "go-sdk"
	}

	headers := map[string]string{
		"Content-Type":     "application/json",
		"Egnyte-Client-Id": clientId,
		"User-Agent":       fmt.Sprintf("%s/%s", SourceName, Version),
	}

	headers["Authorization"] = fmt.Sprintf("Bearer %s", client.token)
	client.headers = headers
	return client, nil
}

// doRequest makes the API call and returns the http.Response and error
// Exactly one of http.Response and error will be nil at a time
func (c *Client) doRequest(ctx context.Context, opts *requestOptions, request, response interface{}) (*http.Response, error) {
	root := c.root
	if opts.Root != "" {
		root = opts.Root
	}
	// Marshal the request if given
	// Set the body up as a marshalled object if no body passed in
	if request != nil && opts.Body == nil {
		requestBody, err := helloeaveJson.MarshalSafeCollections(request)
		if err != nil {
			return nil, err
		}
		opts.Body = bytes.NewBuffer(requestBody)
	}
	var urlRoot string
	urlRoot = fmt.Sprintf("https://%s", root)
	parsedUrl, err := url.Parse(urlRoot)
	if err != nil {
		return nil, err
	}
	parsedUrl.Path += opts.Path
	parsedUrl.RawQuery = opts.Parameters.Encode()
	req, err := http.NewRequest(opts.Method, parsedUrl.String(), opts.Body)
	if err != nil {
		return nil, err
	}

	// Set default headers
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}
	// Set any extra headers
	if opts.ExtraHeaders != nil {
		for k, v := range opts.ExtraHeaders {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.Do(req)
	if resp != nil {
		err = checkResponse(resp)
	}
	if err != nil {
		return nil, err
	}
	if response != nil {
		err = decodeJSON(resp, response)
		if err != nil {
			return nil, err
		}
	}
	if !opts.DontCloseBody && resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	return resp, nil
}

// DecodeJSON decodes resp.Body into result
func decodeJSON(resp *http.Response, result interface{}) (err error) {
	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(result)
}

// checkResponse checks the provided http.Response and returns an
// error (of type *Error) if the response status code is not 2xx
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}
	var message, errorCode string
	body, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		// Egnyte APIs are very inconsistent in the format that they return error
		// responses in. We try each error format one by one to see if we can parse
		// the error message properly
		errReply := new(errorReply)
		err = json.Unmarshal(body, errReply)
		if err == nil && errReply.Success == false {
			message = errReply.ResponseMsg
			errorCode = errReply.ResponseCode
		}
		if message == "" {
			errReply := new(errorReply2)
			err = json.Unmarshal(body, errReply)
			if err == nil && len(errReply.FormErrors) > 0 {
				message = errReply.FormErrors[0].Msg
				errorCode = errReply.FormErrors[0].Code
			}
		}
		if message == "" {
			errReply := new(errorReply3)
			err = json.Unmarshal(body, errReply)
			if err == nil {
				message = errReply.ErrorMessage
			}
		}
		if message == "" && resp.StatusCode >= 500 && resp.StatusCode <= 511 {
			message = http.StatusText(resp.StatusCode)
		}
	}
	return &Error{
		StatusCode: resp.StatusCode,
		ErrorCode:  errorCode,
		Body:       string(body),
		Header:     resp.Header,
		Message:    message,
	}
}

// Returns a new Object for provided path with the client set in the object
func (c *Client) Object(path string) *Object {
	return &Object{
		Client: c,
		Path:   path,
	}
}
