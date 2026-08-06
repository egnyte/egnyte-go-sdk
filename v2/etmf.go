package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ETMFService accesses electronic Trial Master File studies, filing
// levels, artifacts and documents. Requires the Egnyte.etmf scope.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/etmf-api
// OAuth scope: Egnyte.etmf.
type ETMFService service

// ETMFStudy is one clinical study.
type ETMFStudy struct {
	SystemID    string `json:"systemId"`
	ID          string `json:"id"` // human-readable, e.g. DEM-81
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// ETMFSite is a site filing level.
type ETMFSite struct {
	SystemID string `json:"systemId"`
	ID       string `json:"id"`
	Status   string `json:"status"`
}

// ETMFCountry is a country filing level with its sites.
type ETMFCountry struct {
	SystemID string     `json:"systemId"`
	Code     string     `json:"code"` // ISO country code
	Status   string     `json:"status"`
	Sites    []ETMFSite `json:"sites"`
}

// ETMFTrial is the trial filing level with its countries.
type ETMFTrial struct {
	SystemID  string        `json:"systemId"`
	Status    string        `json:"status"`
	Countries []ETMFCountry `json:"countries"`
}

// ETMFFilingLevels is the filing level tree of a study.
type ETMFFilingLevels struct {
	Trial *ETMFTrial `json:"trial"`
}

// ETMFArtifact is one artifact definition of a filing level.
type ETMFArtifact struct {
	ReferenceModelArtifactID string `json:"referenceModelArtifactId"`
	Number                   string `json:"number"` // e.g. 01.01.01
	Name                     string `json:"name"`
	IsUnblinded              bool   `json:"isUnblinded"`
	MilestoneNumber          int    `json:"milestoneNumber"`
}

// ETMFMilestone is one milestone of a filing level.
type ETMFMilestone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Number int    `json:"number"`
	Status string `json:"status"`
}

// ETMFArtifacts pairs the artifacts and milestones of a filing level.
type ETMFArtifacts struct {
	Artifacts  []ETMFArtifact  `json:"artifacts"`
	Milestones []ETMFMilestone `json:"milestones"`
}

// ETMFFilingLevelRef identifies a filing level by type (TRIAL, COUNTRY
// or SITE) and system id.
type ETMFFilingLevelRef struct {
	Type     string `json:"type"`
	SystemID string `json:"systemId,omitempty"`
}

// ETMFDocumentArtifact is the classification of a document.
type ETMFDocumentArtifact struct {
	ArtifactNumber           string `json:"artifactNumber"`
	ArtifactName             string `json:"artifactName"`
	SectionNumber            string `json:"sectionNumber"`
	SectionName              string `json:"sectionName"`
	ZoneNumber               string `json:"zoneNumber"`
	Zone                     string `json:"zone"`
	ReferenceModelArtifactID string `json:"referenceModelArtifactId"`
	IsUnblinded              bool   `json:"isUnblinded"`
}

// ETMFDocumentMilestone is the milestone of a document.
type ETMFDocumentMilestone struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
	ID     string `json:"id"`
}

// ETMFFile references the underlying Egnyte file.
type ETMFFile struct {
	Name    string `json:"name"`
	EntryID string `json:"entryId"`
	GroupID string `json:"groupId"`
	Path    string `json:"path"`
}

// ETMFQualityControl is the QC state of a document.
type ETMFQualityControl struct {
	DueOn string `json:"dueOn"`
	// Status is DRAFT, IN_QC, APPROVED or REJECTED.
	Status string `json:"status"`
}

// ETMFDocument is one filed document.
type ETMFDocument struct {
	StudyID        string                 `json:"studyId"`
	ID             string                 `json:"id"`
	Artifact       *ETMFDocumentArtifact  `json:"artifact"`
	Milestone      *ETMFDocumentMilestone `json:"milestone"`
	FilingLevel    *ETMFFilingLevelRef    `json:"filingLevel"`
	File           *ETMFFile              `json:"file"`
	QualityControl *ETMFQualityControl    `json:"qualityControl"`
}

// ETMFDocumentList is a page of documents.
type ETMFDocumentList struct {
	Documents []ETMFDocument `json:"documents"`
	Offset    int            `json:"offset"`
	HasMore   bool           `json:"hasMore"`
}

// ListETMFDocumentsRequest filters document listings. FilingLevel is
// required.
type ListETMFDocumentsRequest struct {
	FilingLevel ETMFFilingLevelRef `json:"filingLevel"`
	// Zone filters by zone number and name, e.g.
	// "02 Central Trial Documents".
	Zone string `json:"zone,omitempty"`
	// Artifact filters by artifact number and name.
	Artifact string `json:"artifact,omitempty"`
	// Status is DRAFT, IN_QC, APPROVED or REJECTED.
	Status string `json:"status,omitempty"`
	// Limit is the page size (default 20, max 200).
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ETMFClassification classifies a document being added.
type ETMFClassification struct {
	ReferenceModelArtifactID string `json:"referenceModelArtifactId"`
	MilestoneID              string `json:"milestoneId,omitempty"`
}

// ETMFSourceFile references the file to add by group id.
type ETMFSourceFile struct {
	GroupID string `json:"groupId"`
}

// AddETMFDocumentRequest files an existing Egnyte file into a study.
type AddETMFDocumentRequest struct {
	FilingLevel    ETMFFilingLevelRef `json:"filingLevel"`
	Classification ETMFClassification `json:"classification"`
	SourceFile     ETMFSourceFile     `json:"sourceFile"`
}

// AddedETMFDocument is the result of adding a document.
type AddedETMFDocument struct {
	ID   string    `json:"id"`
	File *ETMFFile `json:"file"`
}

// ListStudies returns the studies the user has access to.
//
// GET /pubapi/v1/etmf/studies
func (s *ETMFService) ListStudies(ctx context.Context) ([]ETMFStudy, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/etmf/studies", nil)
	if err != nil {
		return nil, nil, err
	}
	var body struct {
		Studies []ETMFStudy `json:"studies"`
	}
	resp, err := s.client.Do(req, &body)
	if err != nil {
		return nil, resp, err
	}
	return body.Studies, resp, nil
}

// ListFilingLevels returns the filing level tree of a study.
//
// GET /pubapi/v1/etmf/studies/{studyId}/filing-levels
func (s *ETMFService) ListFilingLevels(ctx context.Context, studyID string) (*ETMFFilingLevels, *Response, error) {
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/etmf/studies/"+url.PathEscape(studyID)+"/filing-levels", nil)
	if err != nil {
		return nil, nil, err
	}
	levels := new(ETMFFilingLevels)
	resp, err := s.client.Do(req, levels)
	if err != nil {
		return nil, resp, err
	}
	return levels, resp, nil
}

// ListArtifacts returns the artifacts and milestones of a filing level.
// filingLevel is "trial", "country" or "site".
//
// GET /pubapi/v1/etmf/studies/{studyId}/artifacts
func (s *ETMFService) ListArtifacts(ctx context.Context, studyID, filingLevel, filingLevelID string) (*ETMFArtifacts, *Response, error) {
	if filingLevel == "" || filingLevelID == "" {
		return nil, nil, fmt.Errorf("egnyte: filingLevel and filingLevelId are required")
	}
	q := url.Values{"filingLevel": {filingLevel}, "filingLevelId": {filingLevelID}}
	req, err := s.client.NewRequest(ctx, http.MethodGet, "v1/etmf/studies/"+url.PathEscape(studyID)+"/artifacts?"+q.Encode(), nil)
	if err != nil {
		return nil, nil, err
	}
	artifacts := new(ETMFArtifacts)
	resp, err := s.client.Do(req, artifacts)
	if err != nil {
		return nil, resp, err
	}
	return artifacts, resp, nil
}

// ListDocuments returns the documents of a filing level.
//
// POST /pubapi/v1/etmf/studies/{studyId}/documents/listing
func (s *ETMFService) ListDocuments(ctx context.Context, studyID string, list ListETMFDocumentsRequest) (*ETMFDocumentList, *Response, error) {
	if list.FilingLevel.Type == "" {
		return nil, nil, fmt.Errorf("egnyte: filingLevel.type is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/etmf/studies/"+url.PathEscape(studyID)+"/documents/listing", list)
	if err != nil {
		return nil, nil, err
	}
	documents := new(ETMFDocumentList)
	resp, err := s.client.Do(req, documents)
	if err != nil {
		return nil, resp, err
	}
	return documents, resp, nil
}

// AddDocument files an existing Egnyte file (by group id) into a study
// at the given filing level.
//
// POST /pubapi/v1/etmf/studies/{studyId}/documents
func (s *ETMFService) AddDocument(ctx context.Context, studyID string, document AddETMFDocumentRequest) (*AddedETMFDocument, *Response, error) {
	switch {
	case document.FilingLevel.Type == "", document.FilingLevel.SystemID == "",
		document.Classification.ReferenceModelArtifactID == "", document.SourceFile.GroupID == "":
		return nil, nil, fmt.Errorf("egnyte: filingLevel, classification.referenceModelArtifactId and sourceFile.groupId are required")
	}
	// The spec requires the filing level both as query parameters and in
	// the body; both are populated from the request struct.
	q := url.Values{
		"filingLevel.type":     {document.FilingLevel.Type},
		"filingLevel.systemId": {document.FilingLevel.SystemID},
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/etmf/studies/"+url.PathEscape(studyID)+"/documents?"+q.Encode(), document)
	if err != nil {
		return nil, nil, err
	}
	added := new(AddedETMFDocument)
	resp, err := s.client.Do(req, added)
	if err != nil {
		return nil, resp, err
	}
	return added, resp, nil
}
