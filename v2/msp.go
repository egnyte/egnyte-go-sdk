package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// MSPService lets managed service providers administer their reseller
// account: webhooks, customer domains, Protect tenants, plans, power
// user pools and trials.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/msp-api
// Authentication uses the MSP-specific OAuth setup documented there; this API
// is available only to approved MSP partners and has no general CFS scope.
type MSPService service

// MSPWebhook is a webhook registration summary.
type MSPWebhook struct {
	Status    string `json:"status"`
	WebhookID string `json:"webhook_id"`
}

// MSPWebhookRequest registers or updates an MSP webhook. Updates replace
// all fields.
type MSPWebhookRequest struct {
	// WebhookURL must be HTTPS.
	WebhookURL string `json:"webhook_url"`
	// EventTypes lists reseller events, e.g.
	// RESELLER_CREATED_CONNECT_TRIAL, RESELLER_MRR_CHANGED.
	EventTypes []string `json:"eventTypes"`
	// AuthHeader / AuthHeaderValue add a custom authentication header to
	// webhook deliveries.
	AuthHeader      string `json:"auth_header,omitempty"`
	AuthHeaderValue string `json:"auth_header_value,omitempty"`
}

// MSPWebhookDetails is the full configuration of an MSP webhook.
type MSPWebhookDetails struct {
	WebhookID       string   `json:"webhook_id"`
	EventTypes      []string `json:"event_types"`
	WebhookURL      string   `json:"webhook_url"`
	AuthHeader      string   `json:"auth_header"`
	AuthHeaderValue string   `json:"auth_header_value"`
	Status          string   `json:"status"`
}

// String implements fmt.Stringer with the delivery auth header value
// redacted, so logging the request (including with %+v) never leaks it.
func (r MSPWebhookRequest) String() string {
	type plain MSPWebhookRequest
	c := plain(r)
	if c.AuthHeaderValue != "" {
		c.AuthHeaderValue = redacted
	}
	return fmt.Sprintf("egnyte.MSPWebhookRequest%+v", c)
}

// GoString implements fmt.GoStringer (%#v) with the value redacted.
func (r MSPWebhookRequest) GoString() string { return r.String() }

// String implements fmt.Stringer with the stored auth header value
// redacted; the API returns it in full on reads.
func (d MSPWebhookDetails) String() string {
	type plain MSPWebhookDetails
	c := plain(d)
	if c.AuthHeaderValue != "" {
		c.AuthHeaderValue = redacted
	}
	return fmt.Sprintf("egnyte.MSPWebhookDetails%+v", c)
}

// GoString implements fmt.GoStringer (%#v) with the value redacted.
func (d MSPWebhookDetails) GoString() string { return d.String() }

// MSPFeature is a feature, package or source attached to a plan or
// domain.
type MSPFeature struct {
	Name         string `json:"name"`
	Title        string `json:"title"`
	Category     string `json:"category"`
	Type         string `json:"type"`
	NumPurchased int    `json:"numPurchased"`
}

// MSPDomain is a Connect domain managed by the MSP. Features, Sources
// and MonthlyCost are populated by GetDomain only.
type MSPDomain struct {
	Domain              string       `json:"domain"`
	PlanID              int          `json:"planId"`
	PlanName            string       `json:"planName"`
	Status              string       `json:"status"`
	CreatedOn           string       `json:"createdOn"`
	SubscriptionDate    string       `json:"subscriptionDate"`
	AvailablePULicences int          `json:"availablePuLicences"`
	UsedPULicences      int          `json:"usedPuLicences"`
	UnusedPULicences    int          `json:"unusedPuLicences"`
	AvailableSULicences int          `json:"availableSuLicences"`
	UsedSULicences      int          `json:"usedSuLicences"`
	UnusedSULicences    int          `json:"unusedSuLicences"`
	AvailableStorage    int64        `json:"availableStorage"`
	UsedStorage         int64        `json:"usedStorage"`
	AllocatedStorage    int64        `json:"allocatedStorage"`
	MonthlyCost         float64      `json:"monthlyCost"`
	Features            []MSPFeature `json:"features"`
	Sources             []MSPFeature `json:"sources"`
}

// MSPDomainList is a paginated list of domains.
type MSPDomainList struct {
	Next     string      `json:"next"`
	Previous string      `json:"previous"`
	Count    int         `json:"count"`
	Results  []MSPDomain `json:"results"`
}

// MSPTenant is a Protect tenant managed by the MSP. MonthlyCost is
// populated by GetTenant only.
type MSPTenant struct {
	Tenant            string  `json:"tenant"`
	Domain            string  `json:"domain"`
	PlanID            int     `json:"planId"`
	PlanName          string  `json:"planName"`
	CreatedOn         string  `json:"createdOn"`
	SubscriptionDate  string  `json:"subscriptionDate"`
	Status            string  `json:"status"`
	CupUsed           int     `json:"cupUsed"`
	CupIncluded       int     `json:"cupIncluded"`
	TotalCupPurchased int     `json:"totalCupPurchased"`
	TotalCupIncluded  int     `json:"totalCupIncluded"`
	TotalCupAvailable int     `json:"totalCupAvailable"`
	TotalCupUsed      int     `json:"totalCupUsed"`
	TotalCupUnused    int     `json:"totalCupUnused"`
	MonthlyCost       float64 `json:"monthlyCost"`
}

// MSPTenantList is a paginated list of tenants.
type MSPTenantList struct {
	Next     string      `json:"next"`
	Previous string      `json:"previous"`
	Count    int         `json:"count"`
	Results  []MSPTenant `json:"results"`
}

// MSPPlan is one plan available to the MSP. Connect, DLC and Protect
// plans populate different subsets of the fields.
type MSPPlan struct {
	ID                   int          `json:"id"`
	PlanID               int          `json:"planId"`
	PlanName             string       `json:"planName"`
	MaxPlanPULicences    int          `json:"maxPlanPuLicences"`
	MaxPlanSULicences    int          `json:"maxPlanSuLicences"`
	CostPerUser          float64      `json:"costPerUser"`
	MaxPlanStorage       int64        `json:"maxPlanStorage"`
	AllocatedPlanStorage int64        `json:"allocatedPlanStorage"`
	AvailablePlanStorage int64        `json:"availablePlanStorage"`
	TotalCupPurchased    int          `json:"totalCupPurchased"`
	TotalCupIncluded     int          `json:"totalCupIncluded"`
	TotalCupAvailable    int          `json:"totalCupAvailable"`
	TotalCupUsed         int          `json:"totalCupUsed"`
	TotalCupUnused       int          `json:"totalCupUnused"`
	Features             []MSPFeature `json:"features"`
	Packages             []MSPFeature `json:"packages"`
	Sources              []MSPFeature `json:"sources"`
}

// MSPPlanList is a paginated list of plans.
type MSPPlanList struct {
	Next     string    `json:"next"`
	Previous string    `json:"previous"`
	Count    int       `json:"count"`
	Results  []MSPPlan `json:"results"`
}

// MSPPowerUsersQuote is the pricing outcome of a power user quote or
// purchase. Monetary values are decimal strings.
type MSPPowerUsersQuote struct {
	PreviousPowerUsers int    `json:"previousPowerUsers"`
	NewPowerUsers      int    `json:"newPowerUsers"`
	DeltaPowerUsers    int    `json:"deltaPowerUsers"`
	DeltaMonthly       string `json:"deltaMonthly"`
	ProratedCost       string `json:"proratedCost"`
	ProratedDaysLeft   int    `json:"proratedDaysLeft"`
	ProratedMonthsLeft int    `json:"proratedMonthsLeft"`
}

// MSPPowerUserAllocation is the outcome of allocating pool power users
// to a domain.
type MSPPowerUserAllocation struct {
	Domain             string `json:"domain"`
	PreviousPowerUsers int    `json:"previousPowerUsers"`
	NewPowerUsers      int    `json:"newPowerUsers"`
	PoolAvailableAfter int    `json:"poolAvailableAfter"`
}

// MSPCreateTrialRequest creates a trial domain.
type MSPCreateTrialRequest struct {
	Domain string `json:"domain"` // max 50 characters
	// ID is the plan id from ListPlans.
	ID         int    `json:"id"`
	PowerUsers int    `json:"powerUsers"`
	StorageGB  int    `json:"storageGB"`
	Region     string `json:"region"` // e.g. "us"
	// Optional admin account details.
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"` // write-only
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

// String implements fmt.Stringer with the admin password redacted, so
// logging the request (including with %+v) never leaks it.
func (r MSPCreateTrialRequest) String() string {
	type plain MSPCreateTrialRequest
	c := plain(r)
	if c.Password != "" {
		c.Password = redacted
	}
	return fmt.Sprintf("egnyte.MSPCreateTrialRequest%+v", c)
}

// GoString implements fmt.GoStringer (%#v) with the password redacted.
func (r MSPCreateTrialRequest) GoString() string { return r.String() }

// MSPTrial is a created trial domain.
type MSPTrial struct {
	Domain     string `json:"domain"`
	DomainURL  string `json:"domainUrl"`
	TrialStart string `json:"trialStart"`
	TrialEnd   string `json:"trialEnd"`
	PowerUsers int    `json:"powerUsers"`
	StorageGB  int    `json:"storageGB"`
}

// MSPTrialActivation is the outcome of converting a trial to paid.
type MSPTrialActivation struct {
	Domain         string `json:"domain"`
	DomainURL      string `json:"domainUrl"`
	ActivationDate string `json:"activationDate"`
}

// ListWebhooks returns the MSP's registered webhooks.
//
// GET /pubapi/v1/msp/webhooks
func (s *MSPService) ListWebhooks(ctx context.Context) ([]MSPWebhook, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/msp/webhooks", nil)
	if err != nil {
		return nil, nil, err
	}
	var webhooks []MSPWebhook
	resp, err := s.client.Do(req, &webhooks)
	if err != nil {
		return nil, resp, err
	}
	return webhooks, resp, nil
}

// CreateWebhook registers a webhook for reseller events.
//
// POST /pubapi/v1/msp/webhooks
func (s *MSPService) CreateWebhook(ctx context.Context, webhook MSPWebhookRequest) (*MSPWebhook, *Response, error) {
	if webhook.WebhookURL == "" || len(webhook.EventTypes) == 0 {
		return nil, nil, fmt.Errorf("egnyte: webhook_url and eventTypes are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/msp/webhooks", webhook)
	if err != nil {
		return nil, nil, err
	}
	created := new(MSPWebhook)
	resp, err := s.client.Do(req, created)
	if err != nil {
		return nil, resp, err
	}
	return created, resp, nil
}

// GetWebhook returns the full configuration of one webhook.
//
// GET /pubapi/v1/msp/webhooks/{webhook_id}
func (s *MSPService) GetWebhook(ctx context.Context, id string) (*MSPWebhookDetails, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/msp/webhooks/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}
	details := new(MSPWebhookDetails)
	resp, err := s.client.Do(req, details)
	if err != nil {
		return nil, resp, err
	}
	return details, resp, nil
}

// UpdateWebhook replaces a webhook's configuration.
//
// PUT /pubapi/v1/msp/webhooks/{webhook_id}
func (s *MSPService) UpdateWebhook(ctx context.Context, id string, webhook MSPWebhookRequest) (*MSPWebhook, *Response, error) {
	if webhook.WebhookURL == "" || len(webhook.EventTypes) == 0 {
		return nil, nil, fmt.Errorf("egnyte: webhook_url and eventTypes are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPut, "v1/msp/webhooks/"+url.PathEscape(id), webhook)
	if err != nil {
		return nil, nil, err
	}
	updated := new(MSPWebhook)
	resp, err := s.client.Do(req, updated)
	if err != nil {
		return nil, resp, err
	}
	return updated, resp, nil
}

// DeleteWebhook unregisters a webhook.
//
// DELETE /pubapi/v1/msp/webhooks/{webhook_id}
func (s *MSPService) DeleteWebhook(ctx context.Context, id string) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v1/msp/webhooks/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// ListDomains returns the MSP's Connect domains, optionally filtered by
// plan name.
//
// GET /pubapi/v1/msp/domains
func (s *MSPService) ListDomains(ctx context.Context, planName string) (*MSPDomainList, *Response, error) {
	urlPath := "v1/msp/domains"
	if planName != "" {
		urlPath += "?" + url.Values{"planName": {planName}}.Encode()
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(MSPDomainList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// GetDomain returns full details of a domain, including license usage,
// features and sources.
//
// GET /pubapi/v1/msp/domains/{domain}
func (s *MSPService) GetDomain(ctx context.Context, domain string) (*MSPDomain, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/msp/domains/"+url.PathEscape(domain), nil)
	if err != nil {
		return nil, nil, err
	}
	details := new(MSPDomain)
	resp, err := s.client.Do(req, details)
	if err != nil {
		return nil, resp, err
	}
	return details, resp, nil
}

// ListTenants returns the MSP's Protect tenants, optionally filtered by
// plan name.
//
// GET /pubapi/v1/msp/tenants
func (s *MSPService) ListTenants(ctx context.Context, planName string) (*MSPTenantList, *Response, error) {
	urlPath := "v1/msp/tenants"
	if planName != "" {
		urlPath += "?" + url.Values{"planName": {planName}}.Encode()
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(MSPTenantList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// GetTenant returns full details of a Protect tenant.
//
// GET /pubapi/v1/msp/tenants/{tenant_name}
func (s *MSPService) GetTenant(ctx context.Context, tenantName string) (*MSPTenant, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/msp/tenants/"+url.PathEscape(tenantName), nil)
	if err != nil {
		return nil, nil, err
	}
	tenant := new(MSPTenant)
	resp, err := s.client.Do(req, tenant)
	if err != nil {
		return nil, resp, err
	}
	return tenant, resp, nil
}

// ListPlans returns the plans available to the MSP.
//
// GET /pubapi/v1/msp/plans
func (s *MSPService) ListPlans(ctx context.Context) (*MSPPlanList, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/msp/plans", nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(MSPPlanList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

func (s *MSPService) powerUsers(ctx context.Context, urlPath string, powerUsers int) (*MSPPowerUsersQuote, *Response, error) {
	if powerUsers < 1 {
		return nil, nil, fmt.Errorf("egnyte: powerUsers must be at least 1")
	}
	body := struct {
		PowerUsers int `json:"powerUsers"`
	}{powerUsers}
	req, err := s.client.NewRequest(ctx, http.MethodPost, urlPath, body)
	if err != nil {
		return nil, nil, err
	}
	quote := new(MSPPowerUsersQuote)
	resp, err := s.client.Do(req, quote)
	if err != nil {
		return nil, resp, err
	}
	return quote, resp, nil
}

// PowerUsersQuote prices a change of the MSP pool to powerUsers total
// power users (absolute count) without committing it.
//
// POST /pubapi/v1/msp/plans/{planId}/power-users/quote/
func (s *MSPService) PowerUsersQuote(ctx context.Context, planID, powerUsers int) (*MSPPowerUsersQuote, *Response, error) {
	return s.powerUsers(ctx, fmt.Sprintf("v1/msp/plans/%d/power-users/quote/", planID), powerUsers)
}

// PurchasePowerUsers changes the MSP pool to powerUsers total power
// users; the prorated cost is billed immediately.
//
// POST /pubapi/v1/msp/plans/{planId}/power-users/purchase/
func (s *MSPService) PurchasePowerUsers(ctx context.Context, planID, powerUsers int) (*MSPPowerUsersQuote, *Response, error) {
	return s.powerUsers(ctx, fmt.Sprintf("v1/msp/plans/%d/power-users/purchase/", planID), powerUsers)
}

// AllocatePowerUsers assigns powerUsers (absolute count) from the MSP
// pool to a customer domain.
//
// POST /pubapi/v1/msp/domains/{domain}/power-users/allocate
func (s *MSPService) AllocatePowerUsers(ctx context.Context, domain string, powerUsers int) (*MSPPowerUserAllocation, *Response, error) {
	if powerUsers < 1 {
		return nil, nil, fmt.Errorf("egnyte: powerUsers must be at least 1")
	}
	body := struct {
		PowerUsers int `json:"powerUsers"`
	}{powerUsers}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/msp/domains/"+url.PathEscape(domain)+"/power-users/allocate", body)
	if err != nil {
		return nil, nil, err
	}
	allocation := new(MSPPowerUserAllocation)
	resp, err := s.client.Do(req, allocation)
	if err != nil {
		return nil, resp, err
	}
	return allocation, resp, nil
}

// CreateTrial provisions a new trial domain. A 409 response means the
// domain name is taken (the error body includes suggestions).
//
// POST /pubapi/v1/msp/trials/
func (s *MSPService) CreateTrial(ctx context.Context, trial MSPCreateTrialRequest) (*MSPTrial, *Response, error) {
	switch {
	case trial.Domain == "", trial.ID == 0, trial.PowerUsers < 1,
		trial.StorageGB < 1, trial.Region == "":
		return nil, nil, fmt.Errorf("egnyte: trial domain, id, powerUsers, storageGB and region are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/msp/trials/", trial)
	if err != nil {
		return nil, nil, err
	}
	created := new(MSPTrial)
	resp, err := s.client.Do(req, created)
	if err != nil {
		return nil, resp, err
	}
	return created, resp, nil
}

// ActivateTrial converts a trial domain to a paid subscription.
// powerUsers and storageGB must be at least the current usage.
//
// POST /pubapi/v1/msp/trials/{domain}/activate/
func (s *MSPService) ActivateTrial(ctx context.Context, domain string, planID, powerUsers, storageGB int) (*MSPTrialActivation, *Response, error) {
	if planID < 1 || powerUsers < 1 || storageGB < 1 {
		return nil, nil, fmt.Errorf("egnyte: planId, powerUsers and storageGB must be at least 1")
	}
	body := struct {
		PlanID     int `json:"planId"`
		PowerUsers int `json:"powerUsers"`
		StorageGB  int `json:"storageGB"`
	}{planID, powerUsers, storageGB}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/msp/trials/"+url.PathEscape(domain)+"/activate/", body)
	if err != nil {
		return nil, nil, err
	}
	activation := new(MSPTrialActivation)
	resp, err := s.client.Do(req, activation)
	if err != nil {
		return nil, resp, err
	}
	return activation, resp, nil
}
