package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestWebhooksService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/webhooks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`[{"webhookId":"c9ac0519","expires":2100000000,"status":"enabled"},
			{"webhookId":"d2fcb85e","expires":2100000000,"status":"disabled"}]`))
	})

	webhooks, _, err := client.Webhooks.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(webhooks) != 2 || webhooks[0].WebhookID != "c9ac0519" || webhooks[1].Status != "disabled" {
		t.Errorf("webhooks = %+v", webhooks)
	}
}

func TestWebhooksService_Register(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/webhooks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["url"] != "https://example.com/webhook" ||
			body["path"] != "/Shared/Documents/Invoices" {
			t.Errorf("body = %v", body)
		}
		events, _ := body["eventType"].([]any)
		if len(events) != 1 || events[0] != "fs:add_file" {
			t.Errorf("eventType = %v", body["eventType"])
		}
		if _, ok := body["status"]; ok {
			t.Error("unset status should be omitted on register")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"webhookId":"c9ac0519","expires":2100000000,"status":"enabled"}`))
	})

	webhook, resp, err := client.Webhooks.Register(context.Background(), WebhookRequest{
		URL:       "https://example.com/webhook",
		EventType: []string{"fs:add_file"},
		Path:      "/Shared/Documents/Invoices",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || webhook.WebhookID != "c9ac0519" {
		t.Errorf("webhook = %+v, status = %d", webhook, resp.StatusCode)
	}
}

func TestWebhooksService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.Webhooks.Register(ctx, WebhookRequest{}); err == nil {
		t.Error("missing url should fail Register before sending")
	}
	if _, err := client.Webhooks.Update(ctx, "id1", WebhookRequest{}); err == nil {
		t.Error("missing url should fail Update before sending")
	}
	if _, _, err := client.Webhooks.SetStatus(ctx, "id1", "paused"); err == nil {
		t.Error("invalid status should fail before sending")
	}
}

func TestWebhooksService_UpdateAndDelete(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/webhooks/c9ac0519", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["url"] != "https://example.com/hook2" || body["status"] != "disabled" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(`{}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	if _, err := client.Webhooks.Update(ctx, "c9ac0519", WebhookRequest{
		URL:    "https://example.com/hook2",
		Status: "disabled",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, err := client.Webhooks.Delete(ctx, "c9ac0519"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestWebhooksService_StatusAndSetStatus(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/webhooks/c9ac0519/status", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"webhookId":"c9ac0519","expires":2100000000,"status":"enabled"}`))
		case http.MethodPost:
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["status"] != "disabled" {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(`{"webhookId":"c9ac0519","expires":2100000000,"status":"disabled"}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	webhook, _, err := client.Webhooks.Status(ctx, "c9ac0519")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if webhook.Status != "enabled" {
		t.Errorf("status = %q", webhook.Status)
	}

	webhook, _, err = client.Webhooks.SetStatus(ctx, "c9ac0519", "disabled")
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if webhook.Status != "disabled" {
		t.Errorf("status = %q", webhook.Status)
	}
}

func TestWebhooksService_Details(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/webhooks/c9ac0519/details", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"webhookId":"c9ac0519","expires":2100000000,"status":"enabled",
			"eventType":["fs:add_file","link:create"],
			"path":"/path/dummypath,/path/samples","authHeader":"AuthHeader",
			"url":"https://myidealurl/?qwer"}`))
	})

	details, _, err := client.Webhooks.Details(context.Background(), "c9ac0519")
	if err != nil {
		t.Fatalf("Details: %v", err)
	}
	if len(details.EventType) != 2 || details.URL != "https://myidealurl/?qwer" {
		t.Errorf("details = %+v", details)
	}
}

func TestWebhooksService_WhoAmI(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/whoami", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"clientId":"hweb8adyh3gumiypa45qdgkm","username":"johndoe",
			"domain":"acme","scopes":["fs"]}`))
	})

	who, _, err := client.Webhooks.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	if who.ClientID != "hweb8adyh3gumiypa45qdgkm" || who.Domain != "acme" || len(who.Scopes) != 1 {
		t.Errorf("whoami = %+v", who)
	}
}

func TestWebhooksService_Register_quotaExceeded(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/webhooks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"code":"QUOTA_EXCEEDED","message":"Webhook quota exceeded"}`))
	})

	_, _, err := client.Webhooks.Register(context.Background(), WebhookRequest{URL: "https://example.com/webhook"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Message != "Webhook quota exceeded" {
		t.Errorf("APIError = %+v", apiErr)
	}
}
