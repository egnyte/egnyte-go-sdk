package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// SignService manages e-signature workflows: templates, signature
// requests and agreements.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/sign-api
// OAuth scope: Egnyte.sign.
type SignService service

// SignTemplateDocument is one document configured in a template.
type SignTemplateDocument struct {
	TemplateDocumentID string `json:"template_document_id"`
	DocumentName       string `json:"document_name"`
	DocumentType       string `json:"document_type"`
	EntryID            string `json:"entry_id"`
	Location           string `json:"location"`
	DocumentOrder      int    `json:"document_order"`
}

// SignTemplate is a template summary.
type SignTemplate struct {
	TemplateID        string                 `json:"template_id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Status            string                 `json:"status"` // ACTIVE or INACTIVE
	CreatedBy         int                    `json:"created_by"`
	FullName          string                 `json:"full_name"`
	CreatedAt         string                 `json:"created_at"`
	TemplateDocuments []SignTemplateDocument `json:"template_documents"`
}

// SignTemplateList is the template listing response.
type SignTemplateList struct {
	Count   int            `json:"count"`
	Results []SignTemplate `json:"results"`
}

// SignTemplateParticipant is one participant slot of a template.
type SignTemplateParticipant struct {
	TemplateParticipantID string `json:"template_participant_id"`
	ParticipantType       string `json:"participant_type"` // e.g. SIGNER
	Order                 int    `json:"order"`            // 0 when signing is unordered
	Email                 string `json:"email"`
	Type                  string `json:"type"` // INTERNAL, EXTERNAL or ROLE
	RoleName              string `json:"role_name"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
}

// SignTemplateDetail is the full configuration of a template.
type SignTemplateDetail struct {
	TemplateID              string                    `json:"template_id"`
	Name                    string                    `json:"name"`
	Description             string                    `json:"description"`
	Status                  string                    `json:"status"`
	TargetLocation          string                    `json:"target_location"`
	IsOrderedSigning        bool                      `json:"is_ordered_signing"`
	EmailMessage            string                    `json:"email_message"`
	ExpirationDays          int                       `json:"expiration_days"`
	SignerAllowedToReassign bool                      `json:"signer_allowed_to_reassign"`
	ReminderDetails         map[string]any            `json:"reminder_details"`
	TemplateParticipants    []SignTemplateParticipant `json:"template_participants"`
	TemplateDocuments       []SignTemplateDocument    `json:"template_documents"`
}

// SignAgreementParticipant is one participant of a created agreement.
type SignAgreementParticipant struct {
	ParticipantID string `json:"participant_id"`
	LinkID        string `json:"link_id"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	Order         int    `json:"order"`
	ColorCode     string `json:"color_code"`
	ExpiresAt     string `json:"expires_at"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	IsInternal    bool   `json:"is_internal"`
}

// SignAgreementDocument is one document of an agreement.
type SignAgreementDocument struct {
	DocumentID    string `json:"document_id"`
	EntryID       string `json:"entry_id"`
	DocumentOrder int    `json:"document_order"`
}

// SignAgreement is an agreement created from a signature request.
type SignAgreement struct {
	AgreementID    string                     `json:"agreement_id"`
	AgreementName  string                     `json:"agreement_name"`
	TargetLocation string                     `json:"target_location"`
	Status         string                     `json:"status"`
	Participants   []SignAgreementParticipant `json:"participants"`
	Documents      []SignAgreementDocument    `json:"documents"`
}

// SignRecipient maps a role-based template participant to an email.
type SignRecipient struct {
	TemplateParticipantID string `json:"template_participant_id"`
	Email                 string `json:"email"`
}

// CreateSignatureRequest is the body for sending a signature request
// from a template. Recipients is required when the template contains
// role-based participants.
type CreateSignatureRequest struct {
	TemplateID           string          `json:"template_id"`
	SignatureRequestName string          `json:"signature_request_name"` // max 255 characters
	EmailMessage         string          `json:"email_message,omitempty"`
	Recipients           []SignRecipient `json:"recipients,omitempty"`
}

// SignSentParticipant is one participant of a sent request.
type SignSentParticipant struct {
	ParticipantID string `json:"participant_id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	Order         int    `json:"order"`
	SignedAt      string `json:"signed_at"`
	IsInQueue     bool   `json:"is_in_queue"`
}

// SignDocumentRef references a document by id.
type SignDocumentRef struct {
	DocumentID string `json:"document_id"`
}

// SentSignatureRequest is one request sent by the authenticated user.
type SentSignatureRequest struct {
	AgreementID          string                `json:"agreement_id"`
	SignatureRequestName string                `json:"signature_request_name"`
	Status               string                `json:"status"`
	Documents            []SignDocumentRef     `json:"documents"`
	Participants         []SignSentParticipant `json:"participants"`
	CreatedAt            string                `json:"created_at"`
	DueDate              string                `json:"due_date"`
	TemplateID           string                `json:"template_id"`
	TemplateName         string                `json:"template_name"`
}

// SentSignatureRequestList is a page of sent requests.
type SentSignatureRequestList struct {
	Count   int                    `json:"count"`
	Results []SentSignatureRequest `json:"results"`
}

// MySignatureRequest is one request assigned to the authenticated user.
type MySignatureRequest struct {
	AgreementID                string            `json:"agreement_id"`
	SignatureRequestName       string            `json:"signature_request_name"`
	SentByName                 string            `json:"sent_by_name"`
	SentByEmail                string            `json:"sent_by_email"`
	CreatedBy                  int               `json:"created_by"`
	ExpiresAt                  string            `json:"expires_at"`
	CreatedAt                  string            `json:"created_at"`
	LinkURL                    string            `json:"link_url"`
	SigningLogGenerationStatus string            `json:"signing_log_generation_status"`
	DocumentInfo               []SignDocumentRef `json:"document_info"`
	AgreementParticipantStatus string            `json:"agreement_participant_status"`
	Status                     string            `json:"status"`
}

// MySignatureRequestList is a page of assigned requests.
type MySignatureRequestList struct {
	Count   int                  `json:"count"`
	Results []MySignatureRequest `json:"results"`
}

// SignTemplateListOptions filter template listings.
type SignTemplateListOptions struct {
	// Status is ACTIVE or INACTIVE.
	Status string
	// SortBy is "name" or "created_at".
	SortBy string
	// SortDirection is ASC or DESC.
	SortDirection string
}

// SignRequestListOptions filter signature request listings.
type SignRequestListOptions struct {
	// Status is INPROGRESS, DECLINED, COMPLETED, CANCELLED (sent
	// requests also allow EXPIRED, FAILED).
	Status string
	// SortBy is "name", "dueDate" or "createdAt".
	SortBy string
	// SortDirection is ASC or DESC.
	SortDirection string
	// Limit is the page size (1-50).
	Limit  int
	Offset int
}

func signListQuery(status, sortBy, sortDirection string, limit, offset int) url.Values {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	if sortBy != "" {
		q.Set("sort_by", sortBy)
	}
	if sortDirection != "" {
		q.Set("sort_direction", sortDirection)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return q
}

// ListTemplates returns the signature templates available to the user.
//
// GET /pubapi/v1/egnyte-sign/templates
func (s *SignService) ListTemplates(ctx context.Context, opts *SignTemplateListOptions) (*SignTemplateList, *Response, error) {
	urlPath := "v1/egnyte-sign/templates"
	if opts != nil {
		if q := signListQuery(opts.Status, opts.SortBy, opts.SortDirection, 0, 0).Encode(); q != "" {
			urlPath += "?" + q
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(SignTemplateList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// GetTemplate returns a template's full configuration, including
// participants and documents.
//
// GET /pubapi/v1/egnyte-sign/templates/{template_id}
func (s *SignService) GetTemplate(ctx context.Context, templateID string) (*SignTemplateDetail, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/egnyte-sign/templates/"+url.PathEscape(templateID), nil)
	if err != nil {
		return nil, nil, err
	}
	detail := new(SignTemplateDetail)
	resp, err := s.client.Do(req, detail)
	if err != nil {
		return nil, resp, err
	}
	return detail, resp, nil
}

// CreateSignatureRequest sends a signature request based on a template
// and returns the created agreements.
//
// POST /pubapi/v1/egnyte-sign/signature-requests
func (s *SignService) CreateSignatureRequest(ctx context.Context, request CreateSignatureRequest) ([]SignAgreement, *Response, error) {
	if request.TemplateID == "" || request.SignatureRequestName == "" {
		return nil, nil, fmt.Errorf("egnyte: signature request template_id and signature_request_name are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/egnyte-sign/signature-requests", request)
	if err != nil {
		return nil, nil, err
	}
	var body struct {
		Results []SignAgreement `json:"results"`
	}
	resp, err := s.client.Do(req, &body)
	if err != nil {
		return nil, resp, err
	}
	return body.Results, resp, nil
}

// ListSentRequests returns signature requests sent by the user.
//
// GET /pubapi/v1/egnyte-sign/signature-requests/sent-requests
func (s *SignService) ListSentRequests(ctx context.Context, opts *SignRequestListOptions) (*SentSignatureRequestList, *Response, error) {
	urlPath := "v1/egnyte-sign/signature-requests/sent-requests"
	if opts != nil {
		if q := signListQuery(opts.Status, opts.SortBy, opts.SortDirection, opts.Limit, opts.Offset).Encode(); q != "" {
			urlPath += "?" + q
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(SentSignatureRequestList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// ListMyRequests returns signature requests assigned to the user.
//
// GET /pubapi/v1/egnyte-sign/signature-requests/my-requests
func (s *SignService) ListMyRequests(ctx context.Context, opts *SignRequestListOptions) (*MySignatureRequestList, *Response, error) {
	urlPath := "v1/egnyte-sign/signature-requests/my-requests"
	if opts != nil {
		if q := signListQuery(opts.Status, opts.SortBy, opts.SortDirection, opts.Limit, opts.Offset).Encode(); q != "" {
			urlPath += "?" + q
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	list := new(MySignatureRequestList)
	resp, err := s.client.Do(req, list)
	if err != nil {
		return nil, resp, err
	}
	return list, resp, nil
}

// CancelSignatureRequest cancels an in-progress signature request; this
// cannot be undone. message optionally records the reason (max 500
// characters).
//
// POST /pubapi/v1/egnyte-sign/signature-requests/{agreement_id}/cancel
func (s *SignService) CancelSignatureRequest(ctx context.Context, agreementID, message string) (*Response, error) {
	var body any
	if message != "" {
		body = struct {
			Message string `json:"message"`
		}{message}
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/egnyte-sign/signature-requests/"+url.PathEscape(agreementID)+"/cancel", body)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}
