package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestSignService_ListTemplates(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/egnyte-sign/templates", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("status") != "ACTIVE" || q.Get("sort_by") != "name" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"count":1,"results":[{"template_id":"a0c989ce","name":"Agreement",
			"status":"ACTIVE","created_by":12,"full_name":"John Doe",
			"created_at":"2026-01-02T11:33:12Z"}]}`))
	})

	list, _, err := client.Sign.ListTemplates(context.Background(), &SignTemplateListOptions{
		Status: "ACTIVE",
		SortBy: "name",
	})
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if list.Count != 1 || list.Results[0].FullName != "John Doe" {
		t.Errorf("list = %+v", list)
	}
}

func TestSignService_GetTemplate(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/egnyte-sign/templates/54680c8d", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"template_id":"54680c8d","name":"NDA","status":"ACTIVE",
			"target_location":"/Shared/Documents","is_ordered_signing":false,
			"expiration_days":30,
			"template_participants":[{"template_participant_id":"357abea8",
				"participant_type":"SIGNER","order":0,"type":"ROLE","role_name":"Client"}],
			"template_documents":[{"template_document_id":"td1","document_name":"nda.pdf",
				"document_type":"pdf","document_order":1}]}`))
	})

	detail, _, err := client.Sign.GetTemplate(context.Background(), "54680c8d")
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if detail.ExpirationDays != 30 || len(detail.TemplateParticipants) != 1 ||
		detail.TemplateParticipants[0].RoleName != "Client" {
		t.Errorf("detail = %+v", detail)
	}
}

func TestSignService_CreateSignatureRequest(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/egnyte-sign/signature-requests", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		recipients, _ := body["recipients"].([]any)
		if body["template_id"] != "016d7399" || body["signature_request_name"] != "NDA of 2026A" ||
			len(recipients) != 1 {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"results":[{"agreement_id":"agr1","agreement_name":"NDA of 2026A",
			"status":"INPROGRESS","participants":[{"participant_id":"p1","role":"SIGNER",
			"status":"PENDING","is_internal":false}],
			"documents":[{"document_id":"d1","entry_id":"e1","document_order":1}]}]}`))
	})

	agreements, _, err := client.Sign.CreateSignatureRequest(context.Background(), CreateSignatureRequest{
		TemplateID:           "016d7399",
		SignatureRequestName: "NDA of 2026A",
		EmailMessage:         "Hi, Please sign this document",
		Recipients: []SignRecipient{
			{TemplateParticipantID: "357abea8", Email: "john.doe@externalmail.com"},
		},
	})
	if err != nil {
		t.Fatalf("CreateSignatureRequest: %v", err)
	}
	if len(agreements) != 1 || agreements[0].AgreementID != "agr1" || len(agreements[0].Participants) != 1 {
		t.Errorf("agreements = %+v", agreements)
	}
}

func TestSignService_CreateSignatureRequest_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Sign.CreateSignatureRequest(context.Background(), CreateSignatureRequest{
		TemplateID: "x",
	}); err == nil {
		t.Error("missing signature_request_name should fail before sending")
	}
}

func TestSignService_ListSentRequests(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/egnyte-sign/signature-requests/sent-requests", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("status") != "INPROGRESS" || q.Get("limit") != "10" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"count":1,"results":[{"agreement_id":"agr1",
			"signature_request_name":"NDA of 2026A","status":"INPROGRESS",
			"participants":[{"participant_id":"p1","email":"a@b.com","status":"PENDING","is_in_queue":true}],
			"template_id":"016d7399","template_name":"NDA"}]}`))
	})

	list, _, err := client.Sign.ListSentRequests(context.Background(), &SignRequestListOptions{
		Status: "INPROGRESS",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListSentRequests: %v", err)
	}
	if list.Count != 1 || list.Results[0].TemplateName != "NDA" ||
		!list.Results[0].Participants[0].IsInQueue {
		t.Errorf("list = %+v", list)
	}
}

func TestSignService_ListMyRequests(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/egnyte-sign/signature-requests/my-requests", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"count":1,"results":[{"agreement_id":"agr2",
			"signature_request_name":"Contract","sent_by_name":"Jane Doe",
			"sent_by_email":"jane@x.com","link_url":"https://test.egnyte.com/sign/abc",
			"status":"INPROGRESS"}]}`))
	})

	list, _, err := client.Sign.ListMyRequests(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListMyRequests: %v", err)
	}
	if list.Count != 1 || list.Results[0].SentByName != "Jane Doe" {
		t.Errorf("list = %+v", list)
	}
}

func TestSignService_CancelSignatureRequest(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/egnyte-sign/signature-requests/agr1/cancel", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["message"] != "Document no longer required" {
			t.Errorf("body = %v", body)
		}
	})

	if _, err := client.Sign.CancelSignatureRequest(context.Background(), "agr1",
		"Document no longer required"); err != nil {
		t.Fatalf("CancelSignatureRequest: %v", err)
	}
}

func TestSignService_GetTemplate_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/egnyte-sign/templates/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Template not found"}`))
	})

	_, _, err := client.Sign.GetTemplate(context.Background(), "nope")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError = %+v", apiErr)
	}
}
