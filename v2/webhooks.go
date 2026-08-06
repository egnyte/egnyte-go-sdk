package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// WebhooksService registers and manages webhook callbacks that deliver
// event notifications to your application.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/webhooks-api
// OAuth scope: Egnyte.webhooks.
type WebhooksService service

// Webhook is a webhook registration summary.
type Webhook struct {
	WebhookID string `json:"webhookId"`
	// Expires is the Unix timestamp when the registration expires.
	Expires int64  `json:"expires"`
	Status  string `json:"status"` // enabled or disabled
}

// WebhookDetails is the full configuration of a webhook.
type WebhookDetails struct {
	WebhookID string   `json:"webhookId"`
	Expires   int64    `json:"expires"`
	Status    string   `json:"status"`
	EventType []string `json:"eventType"`
	// Path is a comma-separated list of monitored folder paths.
	Path       string `json:"path"`
	AuthHeader string `json:"authHeader"`
	URL        string `json:"url"`
}

// WebhookRequest is the body for registering or updating a webhook.
// URL is required. Status is only honored on updates.
type WebhookRequest struct {
	// URL is the HTTPS endpoint that receives notifications.
	URL string `json:"url"`
	// EventType lists subscriptions: category wildcards (fs:*, link:*,
	// comment:*, permission:*, meta:*, workflow:*, group:*, user:*) or
	// specific events (fs:add_file). Omitted = all events.
	EventType []string `json:"eventType,omitempty"`
	// Path is a comma-separated list of folder paths to scope the
	// webhook (max 100 paths).
	Path string `json:"path,omitempty"`
	// Status (updates only): enabled or disabled.
	Status string `json:"status,omitempty"`
	// AuthHeader / AuthHeaderValue optionally add a custom
	// authentication header to webhook deliveries.
	AuthHeader      string `json:"authHeader,omitempty"`
	AuthHeaderValue string `json:"authHeaderValue,omitempty"`
}

// String implements fmt.Stringer with the delivery auth header value
// redacted, so logging the request (including with %+v) never leaks it.
func (r WebhookRequest) String() string {
	type plain WebhookRequest
	c := plain(r)
	if c.AuthHeaderValue != "" {
		c.AuthHeaderValue = redacted
	}
	return fmt.Sprintf("egnyte.WebhookRequest%+v", c)
}

// GoString implements fmt.GoStringer (%#v) with the value redacted.
func (r WebhookRequest) GoString() string { return r.String() }

// WhoAmI describes the application and user behind the current token.
type WhoAmI struct {
	ClientID string   `json:"clientId"`
	Username string   `json:"username"`
	Domain   string   `json:"domain"`
	Scopes   []string `json:"scopes"`
}

// WebhookUser identifies the user who triggered a delivered event.
type WebhookUser struct {
	ID             int    `json:"id"`
	DisplayName    string `json:"displayName"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	ClientIDHash   string `json:"clientIdHash"`
	ImpersonatedBy string `json:"impersonatedBy"`
}

// WebhookEvent is one event object of the JSON array Egnyte POSTs to a
// registered webhook endpoint. It is provided for consumers decoding
// webhook deliveries; the SDK itself never receives these.
type WebhookEvent struct {
	EventID          string            `json:"eventId"`
	Domain           string            `json:"domain"`
	Timestamp        int64             `json:"timestamp"`
	User             WebhookUser       `json:"user"`
	ActionSource     string            `json:"actionSource"`
	EventType        string            `json:"eventType"` // e.g. fs:add_file, link:create
	WebhookID        string            `json:"webhookId"`
	CustomProperties map[string]string `json:"customProperties"`
	// Data is event-type specific.
	Data map[string]any `json:"data"`
}

// List returns all webhooks registered by the authenticated client.
//
// GET /pubapi/v1/webhooks
func (s *WebhooksService) List(ctx context.Context) ([]Webhook, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/webhooks", nil)
	if err != nil {
		return nil, nil, err
	}
	var webhooks []Webhook
	resp, err := s.client.Do(req, &webhooks)
	if err != nil {
		return nil, resp, err
	}
	return webhooks, resp, nil
}

// Register creates a new webhook. Registration is subject to quota
// limits (409 when exceeded).
//
// POST /pubapi/v1/webhooks
func (s *WebhooksService) Register(ctx context.Context, webhook WebhookRequest) (*Webhook, *Response, error) {
	if webhook.URL == "" {
		return nil, nil, fmt.Errorf("egnyte: webhook url is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/webhooks", webhook)
	if err != nil {
		return nil, nil, err
	}
	created := new(Webhook)
	resp, err := s.client.Do(req, created)
	if err != nil {
		return nil, resp, err
	}
	return created, resp, nil
}

// Update replaces the configuration of an existing webhook. URL is
// required even when unchanged.
//
// PUT /pubapi/v1/webhooks/{webhookId}
func (s *WebhooksService) Update(ctx context.Context, id string, webhook WebhookRequest) (*Response, error) {
	if webhook.URL == "" {
		return nil, fmt.Errorf("egnyte: webhook url is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPut, "v1/webhooks/"+url.PathEscape(id), webhook)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// Delete unregisters the webhook.
//
// DELETE /pubapi/v1/webhooks/{webhookId}
func (s *WebhooksService) Delete(ctx context.Context, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v1/webhooks/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// Status returns the current status and expiration of the webhook.
//
// GET /pubapi/v1/webhooks/{webhookId}/status
func (s *WebhooksService) Status(ctx context.Context, id string) (*Webhook, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/webhooks/"+url.PathEscape(id)+"/status", nil)
	if err != nil {
		return nil, nil, err
	}
	webhook := new(Webhook)
	resp, err := s.client.Do(req, webhook)
	if err != nil {
		return nil, resp, err
	}
	return webhook, resp, nil
}

// SetStatus enables or disables the webhook. status must be "enabled" or
// "disabled".
//
// POST /pubapi/v1/webhooks/{webhookId}/status
func (s *WebhooksService) SetStatus(ctx context.Context, id, status string) (*Webhook, *Response, error) {
	if status != "enabled" && status != "disabled" {
		return nil, nil, fmt.Errorf("egnyte: webhook status must be %q or %q", "enabled", "disabled")
	}
	body := struct {
		Status string `json:"status"`
	}{status}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/webhooks/"+url.PathEscape(id)+"/status", body)
	if err != nil {
		return nil, nil, err
	}
	webhook := new(Webhook)
	resp, err := s.client.Do(req, webhook)
	if err != nil {
		return nil, resp, err
	}
	return webhook, resp, nil
}

// Details returns the full configuration of the webhook.
//
// GET /pubapi/v1/webhooks/{webhookId}/details
func (s *WebhooksService) Details(ctx context.Context, id string) (*WebhookDetails, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/webhooks/"+url.PathEscape(id)+"/details", nil)
	if err != nil {
		return nil, nil, err
	}
	details := new(WebhookDetails)
	resp, err := s.client.Do(req, details)
	if err != nil {
		return nil, resp, err
	}
	return details, resp, nil
}

// WhoAmI returns the clientId, user, domain and scopes associated with
// the current token.
//
// GET /pubapi/v1/whoami
func (s *WebhooksService) WhoAmI(ctx context.Context) (*WhoAmI, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/whoami", nil)
	if err != nil {
		return nil, nil, err
	}
	who := new(WhoAmI)
	resp, err := s.client.Do(req, who)
	if err != nil {
		return nil, resp, err
	}
	return who, resp, nil
}
