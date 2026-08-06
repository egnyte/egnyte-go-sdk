package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// MetadataService manages custom metadata namespaces, keys and the
// metadata values attached to files, file versions and folders.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/metadata-api
// Authentication requires an OAuth bearer token. The published OAuth scope
// table does not list a dedicated Metadata scope; namespace visibility and
// modification rules still apply.
type MetadataService service

// MetadataKey defines one key inside a namespace.
type MetadataKey struct {
	// Type is one of: integer, string, decimal, date, enum, labels,
	// multi_value_enum.
	Type        string `json:"type"`
	DisplayName string `json:"displayName,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	HelpText    string `json:"helpText,omitempty"`
	// Data lists the enumerated values (required for enum and
	// multi_value_enum, optional for labels).
	Data []string `json:"data,omitempty"`
}

// NamespaceFolderAssociation links a FOLDER_SCOPE namespace to a folder.
type NamespaceFolderAssociation struct {
	FolderID string `json:"folderId"`
	Path     string `json:"path"`
}

// Namespace groups metadata keys and controls their visibility.
type Namespace struct {
	Name                  string `json:"name"`
	DisplayName           string `json:"displayName"`
	Priority              int    `json:"priority"`
	Scope                 string `json:"scope"` // public, protected or private
	SchemaSystemGenerated bool   `json:"schemaSystemGenerated"`
	Inheritable           bool   `json:"inheritable"`
	// MetadataScopeType is GLOBAL, FOLDER_SCOPE or DOCUMENT_TYPE.
	MetadataScopeType          string                       `json:"metadataScopeType"`
	Keys                       map[string]MetadataKey       `json:"keys"`
	FolderAssociations         []NamespaceFolderAssociation `json:"folderAssociations"`
	HasAccessToAllAssociations bool                         `json:"hasAccessToAllAssociations"`
	DocumentTypes              []string                     `json:"documentTypes"`
	DocumentExtensions         []string                     `json:"documentExtensions"`
}

// CreateNamespaceRequest is the body for creating a namespace. Name,
// Scope and Keys are required.
type CreateNamespaceRequest struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"` // public, protected or private
	DisplayName string `json:"displayName,omitempty"`
	Inheritable *bool  `json:"inheritable,omitempty"`
	// MetadataScopeType is GLOBAL (default), FOLDER_SCOPE or DOCUMENT_TYPE.
	MetadataScopeType string `json:"metadataScopeType,omitempty"`
	// AssociatedFolderIDs applies only when MetadataScopeType is FOLDER_SCOPE.
	AssociatedFolderIDs []string `json:"associatedFolderIds,omitempty"`
	// DocumentTypes is required when MetadataScopeType is DOCUMENT_TYPE.
	DocumentTypes []string               `json:"documentTypes,omitempty"`
	Keys          map[string]MetadataKey `json:"keys"`
}

// NamespaceFolderAssociationUpdate associates or dissociates folders.
type NamespaceFolderAssociationUpdate struct {
	AssociateFolderIDs  []string `json:"associateFolderIds,omitempty"`
	DissociateFolderIDs []string `json:"dissociateFolderIds,omitempty"`
}

// NamespaceDocumentTypeAssociationUpdate associates or dissociates
// document types.
type NamespaceDocumentTypeAssociationUpdate struct {
	AssociateDocumentTypes  []string `json:"associateDocumentTypes,omitempty"`
	DissociateDocumentTypes []string `json:"dissociateDocumentTypes,omitempty"`
}

// UpdateNamespaceRequest partially updates a namespace. Set at most one
// of FolderAssociation or DocumentTypeAssociation per request.
type UpdateNamespaceRequest struct {
	DisplayName string `json:"displayName,omitempty"`
	// Priorities maps key names to new priority values.
	Priorities              map[string]int                          `json:"priorities,omitempty"`
	FolderAssociation       *NamespaceFolderAssociationUpdate       `json:"folderAssociation,omitempty"`
	DocumentTypeAssociation *NamespaceDocumentTypeAssociationUpdate `json:"documentTypeAssociation,omitempty"`
}

// CreateMetadataKeyRequest adds a key to an existing namespace.
type CreateMetadataKeyRequest struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	DisplayName string   `json:"displayName,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	HelpText    string   `json:"helpText,omitempty"`
	Data        []string `json:"data,omitempty"`
}

// UpdateMetadataKeyRequest partially updates a key. Types may only be
// widened.
type UpdateMetadataKeyRequest struct {
	DisplayName string   `json:"displayName,omitempty"`
	HelpText    string   `json:"helpText,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	Type        string   `json:"type,omitempty"`
	Data        []string `json:"data,omitempty"`
}

// MetadataKeyDataDeletion names a key and the enumerated values to
// delete from it.
type MetadataKeyDataDeletion struct {
	Key  string   `json:"key"`
	Data []string `json:"data"`
}

// MetadataValues is the metadata attached to an item, keyed by namespace
// name and then by key name.
type MetadataValues struct {
	Results []map[string]map[string]any `json:"results"`
}

// MetadataKeyRef identifies a key for has_key searches.
type MetadataKeyRef struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

// MetadataKeyValueRef identifies a key and a value for key_with_value
// searches (substring match).
type MetadataKeyValueRef struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

// MetadataSearchRequest finds items by metadata criteria. Multiple
// entries in either list are OR-combined.
type MetadataSearchRequest struct {
	// Type restricts results: ALL, FOLDER or FILE.
	Type         string                `json:"type,omitempty"`
	HasKey       []MetadataKeyRef      `json:"has_key,omitempty"`
	KeyWithValue []MetadataKeyValueRef `json:"key_with_value,omitempty"`
}

const forceDeleteHeader = "X-Egnyte-Force-Delete"

// ListNamespaces returns all metadata namespaces in the domain.
//
// GET /pubapi/v1/properties/namespace
func (s *MetadataService) ListNamespaces(ctx context.Context, includeFolderAssociations bool) ([]Namespace, *Response, error) {
	urlPath := "v1/properties/namespace"
	if includeFolderAssociations {
		urlPath += "?includeFolderAssociations=true"
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	var namespaces []Namespace
	resp, err := s.client.Do(req, &namespaces)
	if err != nil {
		return nil, resp, err
	}
	return namespaces, resp, nil
}

// CreateNamespace creates a namespace with its key definitions.
//
// POST /pubapi/v1/properties/namespace
func (s *MetadataService) CreateNamespace(ctx context.Context, ns CreateNamespaceRequest) (*Response, error) {
	if ns.Name == "" || ns.Scope == "" || len(ns.Keys) == 0 {
		return nil, fmt.Errorf("egnyte: namespace name, scope and keys are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/properties/namespace", ns)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// GetNamespace returns the namespace and its key definitions.
//
// GET /pubapi/v1/properties/namespace/{namespaceName}
func (s *MetadataService) GetNamespace(ctx context.Context, name string) (*Namespace, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/properties/namespace/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, nil, err
	}
	ns := new(Namespace)
	resp, err := s.client.Do(req, ns)
	if err != nil {
		return nil, resp, err
	}
	return ns, resp, nil
}

// UpdateNamespace updates namespace attributes and returns the updated
// namespace.
//
// PATCH /pubapi/v1/properties/namespace/{namespaceName}
func (s *MetadataService) UpdateNamespace(ctx context.Context, name string, update UpdateNamespaceRequest) (*Namespace, *Response, error) {
	if update.FolderAssociation != nil && update.DocumentTypeAssociation != nil {
		return nil, nil, fmt.Errorf("egnyte: set only one of folderAssociation or documentTypeAssociation per request")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPatch, "v1/properties/namespace/"+url.PathEscape(name), update)
	if err != nil {
		return nil, nil, err
	}
	ns := new(Namespace)
	resp, err := s.client.Do(req, ns)
	if err != nil {
		return nil, resp, err
	}
	return ns, resp, nil
}

// DeleteNamespace deletes a namespace. force deletes it even when the
// namespace or its keys are in use.
//
// DELETE /pubapi/v1/properties/namespace/{namespaceName}
func (s *MetadataService) DeleteNamespace(ctx context.Context, name string, force bool) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, "v1/properties/namespace/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, err
	}
	if force {
		req.Header.Set(forceDeleteHeader, "Yes")
	}
	return s.client.Do(req, nil)
}

// CreateKey adds a key to an existing namespace.
//
// POST /pubapi/v1/properties/namespace/{namespaceName}/keys
func (s *MetadataService) CreateKey(ctx context.Context, namespace string, key CreateMetadataKeyRequest) (*Response, error) {
	if key.Key == "" || key.Type == "" {
		return nil, fmt.Errorf("egnyte: metadata key name and type are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/properties/namespace/"+url.PathEscape(namespace)+"/keys", key)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// UpdateKey updates attributes of a key.
//
// PATCH /pubapi/v1/properties/namespace/{namespaceName}/keys/{keyName}
func (s *MetadataService) UpdateKey(ctx context.Context, namespace, key string, update UpdateMetadataKeyRequest) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodPatch, s.keyPath(namespace, key), update)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

// DeleteKey deletes a key from a namespace. force deletes it even when
// in use.
//
// DELETE /pubapi/v1/properties/namespace/{namespaceName}/keys/{keyName}
func (s *MetadataService) DeleteKey(ctx context.Context, namespace, key string, force bool) (*Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodDelete, s.keyPath(namespace, key), nil)
	if err != nil {
		return nil, err
	}
	if force {
		req.Header.Set(forceDeleteHeader, "Yes")
	}
	return s.client.Do(req, nil)
}

// DeleteKeyData removes specific enumerated values from a key. force
// deletes values even when in use.
//
// PATCH /pubapi/v1/properties/namespace/{namespaceName}/keys/{keyName}/data
func (s *MetadataService) DeleteKeyData(ctx context.Context, namespace, key string, values []string, force bool) (*Response, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("egnyte: at least one value to delete is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPatch, s.keyPath(namespace, key)+"/data", values)
	if err != nil {
		return nil, err
	}
	if force {
		req.Header.Set(forceDeleteHeader, "Yes")
	}
	return s.client.Do(req, nil)
}

// DeleteKeysData removes enumerated values from multiple keys in one
// request. force deletes values even when in use.
//
// PATCH /pubapi/v1/properties/namespace/{namespaceName}/keys/data
func (s *MetadataService) DeleteKeysData(ctx context.Context, namespace string, deletions []MetadataKeyDataDeletion, force bool) (*Response, error) {
	if len(deletions) == 0 {
		return nil, fmt.Errorf("egnyte: at least one key data deletion is required")
	}
	body := struct {
		KeyDataDeletions []MetadataKeyDataDeletion `json:"keyDataDeletions"`
	}{deletions}
	req, err := s.client.NewRequest(ctx, http.MethodPatch, "v1/properties/namespace/"+url.PathEscape(namespace)+"/keys/data", body)
	if err != nil {
		return nil, err
	}
	if force {
		req.Header.Set(forceDeleteHeader, "Yes")
	}
	return s.client.Do(req, nil)
}

func (s *MetadataService) keyPath(namespace, key string) string {
	return "v1/properties/namespace/" + url.PathEscape(namespace) + "/keys/" + url.PathEscape(key)
}

// SetFileMetadata sets values for keys of one namespace on a file. Use a
// group id for the latest version or an entry id for a specific version.
//
// PUT /pubapi/v1/fs/ids/file/{id}/properties/{namespaceName}
func (s *MetadataService) SetFileMetadata(ctx context.Context, fileID, namespace string, values map[string]any) (*Response, error) {
	return s.setValues(ctx, "file", fileID, namespace, values)
}

// GetFileMetadata returns the values of one namespace on a file or file
// version.
//
// GET /pubapi/v1/fs/ids/file/{id}/properties/{namespaceName}
func (s *MetadataService) GetFileMetadata(ctx context.Context, fileID, namespace string) (*MetadataValues, *Response, error) {
	return s.getValues(ctx, "file", fileID, namespace)
}

// SetFolderMetadata sets values for keys of one namespace on a folder.
//
// PUT /pubapi/v1/fs/ids/folder/{id}/properties/{namespaceName}
func (s *MetadataService) SetFolderMetadata(ctx context.Context, folderID, namespace string, values map[string]any) (*Response, error) {
	return s.setValues(ctx, "folder", folderID, namespace, values)
}

// GetFolderMetadata returns the values of one namespace on a folder.
//
// GET /pubapi/v1/fs/ids/folder/{id}/properties/{namespaceName}
func (s *MetadataService) GetFolderMetadata(ctx context.Context, folderID, namespace string) (*MetadataValues, *Response, error) {
	return s.getValues(ctx, "folder", folderID, namespace)
}

func (s *MetadataService) valuesPath(itemType, id, namespace string) string {
	return "v1/fs/ids/" + itemType + "/" + url.PathEscape(id) + "/properties/" + url.PathEscape(namespace)
}

func (s *MetadataService) setValues(ctx context.Context, itemType, id, namespace string, values map[string]any) (*Response, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("egnyte: at least one metadata value is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPut, s.valuesPath(itemType, id, namespace), values)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req, nil)
}

func (s *MetadataService) getValues(ctx context.Context, itemType, id, namespace string) (*MetadataValues, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, s.valuesPath(itemType, id, namespace), nil)
	if err != nil {
		return nil, nil, err
	}
	values := new(MetadataValues)
	resp, err := s.client.Do(req, values)
	if err != nil {
		return nil, resp, err
	}
	return values, resp, nil
}

// Search finds files and folders by metadata criteria. key_with_value
// matching is substring, not exact.
//
// POST /pubapi/v1/search
func (s *MetadataService) Search(ctx context.Context, search MetadataSearchRequest) (*SearchResults, *Response, error) {
	if len(search.HasKey) == 0 && len(search.KeyWithValue) == 0 {
		return nil, nil, fmt.Errorf("egnyte: metadata search requires has_key or key_with_value criteria")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/search", search)
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
