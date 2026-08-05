package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestETMFService_ListStudies(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/etmf/studies", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"studies":[{"systemId":"eee92526","id":"DEM-81",
			"name":"Demo 81 Study","status":"ACTIVE"}]}`))
	})

	studies, _, err := client.ETMF.ListStudies(context.Background())
	if err != nil {
		t.Fatalf("ListStudies: %v", err)
	}
	if len(studies) != 1 || studies[0].ID != "DEM-81" {
		t.Errorf("studies = %+v", studies)
	}
}

func TestETMFService_ListFilingLevels(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/etmf/studies/eee92526/filing-levels", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"trial":{"systemId":"2f47005e","status":"ACTIVE",
			"countries":[{"systemId":"5ccfd52a","code":"POL","status":"ACTIVE",
			"sites":[{"systemId":"ab650679","id":"WAW_01","status":"ACTIVE"}]}]}}`))
	})

	levels, _, err := client.ETMF.ListFilingLevels(context.Background(), "eee92526")
	if err != nil {
		t.Fatalf("ListFilingLevels: %v", err)
	}
	if levels.Trial == nil || len(levels.Trial.Countries) != 1 ||
		levels.Trial.Countries[0].Sites[0].ID != "WAW_01" {
		t.Errorf("levels = %+v", levels)
	}
}

func TestETMFService_ListArtifacts(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/etmf/studies/eee92526/artifacts", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("filingLevel") != "trial" || q.Get("filingLevelId") != "2f47005e" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"artifacts":[{"referenceModelArtifactId":"001","number":"01.01.01",
			"name":"Trial Master File Plan","isUnblinded":false,"milestoneNumber":2}],
			"milestones":[{"id":"aa9b50b7","name":"Clinical Infrastructure Ready",
			"number":2,"status":"ACTIVE"}]}`))
	})

	artifacts, _, err := client.ETMF.ListArtifacts(context.Background(), "eee92526", "trial", "2f47005e")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts.Artifacts) != 1 || len(artifacts.Milestones) != 1 ||
		artifacts.Milestones[0].Number != 2 {
		t.Errorf("artifacts = %+v", artifacts)
	}
}

func TestETMFService_ListDocuments(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/etmf/studies/eee92526/documents/listing", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fl, _ := body["filingLevel"].(map[string]any)
		if fl["type"] != "SITE" || body["status"] != "DRAFT" || body["limit"] != float64(100) {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"documents":[{"studyId":"ab650679","id":"doc1",
			"artifact":{"artifactNumber":"02.01.01","artifactName":"Investigator's Brochure"},
			"milestone":{"name":"Clinical Infrastructure Ready","number":2,"id":"45f8cb40"},
			"filingLevel":{"type":"TRIAL"},
			"file":{"name":"IB v1.docx","entryId":"17c79408","groupId":"aa5f47a4"},
			"qualityControl":{"dueOn":"2025-04-12T14:08:14.812Z","status":"DRAFT"}}],
			"offset":0,"hasMore":true}`))
	})

	documents, _, err := client.ETMF.ListDocuments(context.Background(), "eee92526", ListETMFDocumentsRequest{
		FilingLevel: ETMFFilingLevelRef{Type: "SITE", SystemID: "b53884cf"},
		Status:      "DRAFT",
		Limit:       100,
	})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if !documents.HasMore || len(documents.Documents) != 1 ||
		documents.Documents[0].QualityControl.Status != "DRAFT" {
		t.Errorf("documents = %+v", documents)
	}
}

func TestETMFService_AddDocument(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/etmf/studies/eee92526/documents", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		q := r.URL.Query()
		if q.Get("filingLevel.type") != "TRIAL" || q.Get("filingLevel.systemId") != "aa9b50b7" {
			t.Errorf("query = %v", q)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		classification, _ := body["classification"].(map[string]any)
		source, _ := body["sourceFile"].(map[string]any)
		if classification["referenceModelArtifactId"] != "001" || source["groupId"] != "35ed2e11" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"id":"aa9b50b7","file":{"groupId":"35ed2e11","entryId":"5c8598b6",
			"path":"/Shared/__eTMF__/example.pdf"}}`))
	})

	added, _, err := client.ETMF.AddDocument(context.Background(), "eee92526", AddETMFDocumentRequest{
		FilingLevel:    ETMFFilingLevelRef{Type: "TRIAL", SystemID: "aa9b50b7"},
		Classification: ETMFClassification{ReferenceModelArtifactID: "001", MilestoneID: "c8cfa0dd"},
		SourceFile:     ETMFSourceFile{GroupID: "35ed2e11"},
	})
	if err != nil {
		t.Fatalf("AddDocument: %v", err)
	}
	if added.ID != "aa9b50b7" || added.File.EntryID != "5c8598b6" {
		t.Errorf("added = %+v", added)
	}
}

func TestETMFService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.ETMF.ListArtifacts(ctx, "s1", "", "id"); err == nil {
		t.Error("missing filingLevel should fail before sending")
	}
	if _, _, err := client.ETMF.ListDocuments(ctx, "s1", ListETMFDocumentsRequest{}); err == nil {
		t.Error("missing filingLevel.type should fail before sending")
	}
	if _, _, err := client.ETMF.AddDocument(ctx, "s1", AddETMFDocumentRequest{}); err == nil {
		t.Error("empty add document request should fail before sending")
	}
}

func TestETMFService_ListStudies_forbidden(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/etmf/studies", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Missing Egnyte.etmf scope"}`))
	})

	_, _, err := client.ETMF.ListStudies(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("APIError = %+v", apiErr)
	}
}
