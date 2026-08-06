package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ProjectCustomFieldsService defines structured metadata fields for
// projects. Requires the Projects feature and customized project
// metadata enabled on the domain.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/project-custom-metadata-api
// Authentication requires an OAuth bearer token. The published OAuth scope
// table does not list a dedicated Project Custom Fields scope.
type ProjectCustomFieldsService service

// ProjectCustomField is one custom field definition.
type ProjectCustomField struct {
	// Name is the internal identifier, auto-derived from DisplayName.
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	// Type is string, integer, decimal, date, enum, labels or
	// multi_value_enum.
	Type string `json:"type"`
	// Data lists predefined values (enum, labels, multi_value_enum).
	Data      []string `json:"data"`
	HelpText  string   `json:"helpText"`
	Priority  int      `json:"priority"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

// CreateProjectCustomFieldRequest is the body for creating a field.
// DisplayName, Type and HelpText are required; Data is required for enum
// and multi_value_enum, optional for labels, and must be empty for
// primitive types.
type CreateProjectCustomFieldRequest struct {
	DisplayName string   `json:"displayName"` // max 40 characters
	Type        string   `json:"type"`
	Data        []string `json:"data,omitempty"`
	HelpText    string   `json:"helpText"` // max 200 characters
	// Priority is the display order (0-65535, lower first); nil appends
	// the field last.
	Priority *int `json:"priority,omitempty"`
}

// UpdateProjectCustomFieldRequest partially updates a field. For enum
// fields Data replaces the existing values; for multi_value_enum and
// labels it appends.
type UpdateProjectCustomFieldRequest struct {
	DisplayName string   `json:"displayName,omitempty"`
	HelpText    string   `json:"helpText,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Data        []string `json:"data,omitempty"`
}

const projectCustomFieldsPath = "v1/properties/project-tag/custom-fields"

// List returns all custom field definitions for the domain, keyed by
// field name.
//
// GET /pubapi/v1/properties/project-tag/custom-fields
func (s *ProjectCustomFieldsService) List(ctx context.Context) (map[string]ProjectCustomField, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, projectCustomFieldsPath, nil)
	if err != nil {
		return nil, nil, err
	}
	var body struct {
		Keys map[string]ProjectCustomField `json:"keys"`
	}
	resp, err := s.client.Do(req, &body)
	if err != nil {
		return nil, resp, err
	}
	return body.Keys, resp, nil
}

// Create defines a new custom field. The internal name is derived from
// DisplayName by the server (lowercase, spaces to hyphens).
//
// POST /pubapi/v1/properties/project-tag/custom-fields
func (s *ProjectCustomFieldsService) Create(ctx context.Context, field CreateProjectCustomFieldRequest) (*Response, error) {
	if field.DisplayName == "" || field.Type == "" || field.HelpText == "" {
		return nil, fmt.Errorf("egnyte: custom field displayName, type and helpText are required")
	}
	switch field.Type {
	case "enum", "multi_value_enum":
		if len(field.Data) == 0 {
			return nil, fmt.Errorf("egnyte: %s fields require a non-empty data array", field.Type)
		}
	case "string", "integer", "decimal", "date":
		if len(field.Data) > 0 {
			return nil, fmt.Errorf("egnyte: %s fields must not include data", field.Type)
		}
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, projectCustomFieldsPath, field)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// Update modifies an existing custom field definition.
//
// PATCH /pubapi/v1/properties/project-tag/custom-fields/{field_name}
func (s *ProjectCustomFieldsService) Update(ctx context.Context, fieldName string, update UpdateProjectCustomFieldRequest) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPatch, projectCustomFieldsPath+"/"+url.PathEscape(fieldName), update)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// Delete permanently removes a custom field definition and its values
// from all projects. force deletes the field even when in use. This
// cannot be undone.
//
// DELETE /pubapi/v1/properties/project-tag/custom-fields/{field_name}
func (s *ProjectCustomFieldsService) Delete(ctx context.Context, fieldName string, force bool) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, projectCustomFieldsPath+"/"+url.PathEscape(fieldName), nil)
	if err != nil {
		return nil, err
	}
	if force {
		req.Header.Set(forceDeleteHeader, "Yes")
	}
	return s.client.Do(req, nil)
}

// DeleteData removes specific predefined values from an enum, labels or
// multi_value_enum field and clears their usage from all projects. force
// deletes values even when in use. This cannot be undone.
//
// PATCH /pubapi/v1/properties/project-tag/custom-fields/{field_name}/data
func (s *ProjectCustomFieldsService) DeleteData(ctx context.Context, fieldName string, values []string, force bool) (*Response, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("egnyte: at least one value to delete is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPatch, projectCustomFieldsPath+"/"+url.PathEscape(fieldName)+"/data", values)
	if err != nil {
		return nil, err
	}
	if force {
		req.Header.Set(forceDeleteHeader, "Yes")
	}
	return s.client.Do(req, nil)
}
