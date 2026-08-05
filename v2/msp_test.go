package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestMSPService_WebhookLifecycle(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/msp/webhooks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`[{"status":"enabled","webhook_id":"4091c6d2"}]`))
		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			events, _ := body["eventTypes"].([]any)
			if body["webhook_url"] != "https://www.example.com/egnyte-events" || len(events) != 2 {
				t.Errorf("body = %v", body)
			}
			w.Write([]byte(`{"status":"enabled","webhook_id":"a63790bf"}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	mux.HandleFunc("/pubapi/v1/msp/webhooks/a63790bf", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"webhook_id":"a63790bf","event_types":["RESELLER_MRR_CHANGED"],
				"webhook_url":"https://www.example.com/egnyte-events","auth_header":"X-TOKEN",
				"auth_header_value":"a234","status":"enabled"}`))
		case http.MethodPut:
			w.Write([]byte(`{"status":"enabled","webhook_id":"a63790bf"}`))
		case http.MethodDelete:
			w.Write([]byte(`{"success":true}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	ctx := context.Background()
	webhooks, _, err := client.MSP.ListWebhooks(ctx)
	if err != nil {
		t.Fatalf("ListWebhooks: %v", err)
	}
	if len(webhooks) != 1 || webhooks[0].WebhookID != "4091c6d2" {
		t.Errorf("webhooks = %+v", webhooks)
	}

	created, _, err := client.MSP.CreateWebhook(ctx, MSPWebhookRequest{
		WebhookURL: "https://www.example.com/egnyte-events",
		EventTypes: []string{"RESELLER_CREATED_CONNECT_TRIAL", "RESELLER_CONVERTED_CONNECT_TRIAL_TO_PAID"},
		AuthHeader: "AUTH-TOKEN", AuthHeaderValue: "XYZ",
	})
	if err != nil {
		t.Fatalf("CreateWebhook: %v", err)
	}
	if created.WebhookID != "a63790bf" {
		t.Errorf("created = %+v", created)
	}

	details, _, err := client.MSP.GetWebhook(ctx, "a63790bf")
	if err != nil {
		t.Fatalf("GetWebhook: %v", err)
	}
	if len(details.EventTypes) != 1 || details.AuthHeader != "X-TOKEN" {
		t.Errorf("details = %+v", details)
	}

	if _, _, err := client.MSP.UpdateWebhook(ctx, "a63790bf", MSPWebhookRequest{
		WebhookURL: "https://www.example.com/updated",
		EventTypes: []string{"RESELLER_MRR_CHANGED"},
	}); err != nil {
		t.Fatalf("UpdateWebhook: %v", err)
	}
	if _, err := client.MSP.DeleteWebhook(ctx, "a63790bf"); err != nil {
		t.Fatalf("DeleteWebhook: %v", err)
	}
}

func TestMSPService_DomainsAndTenants(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/msp/domains", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("planName"); got != "Platform Enterprise" {
			t.Errorf("planName = %q", got)
		}
		w.Write([]byte(`{"next":null,"previous":null,"count":1,"results":[
			{"domain":"testDomain","planId":1,"planName":"Platform Enterprise","status":"active",
			 "availablePuLicences":10,"usedPuLicences":1,"unusedPuLicences":9,
			 "availableStorage":2000,"allocatedStorage":1000}]}`))
	})
	mux.HandleFunc("/pubapi/v1/msp/domains/testDomain", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"domain":"testDomain","planId":1,"status":"active","monthlyCost":99.5,
			"features":[{"name":"olc","title":"Turbo or Storage Sync","type":"flat","numPurchased":2}],
			"sources":[{"name":"data_security_mgmt_gmail_users","title":"Gmail Users","numPurchased":0}]}`))
	})
	mux.HandleFunc("/pubapi/v1/msp/tenants/protecttest", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tenant":"protecttest","domain":"testdomain3","planId":20202,
			"planName":"Protect","status":"active","cupUsed":8649,"totalCupAvailable":11005,
			"monthlyCost":400.99}`))
	})

	ctx := context.Background()
	domains, _, err := client.MSP.ListDomains(ctx, "Platform Enterprise")
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if domains.Count != 1 || domains.Results[0].AvailablePULicences != 10 {
		t.Errorf("domains = %+v", domains)
	}

	domain, _, err := client.MSP.GetDomain(ctx, "testDomain")
	if err != nil {
		t.Fatalf("GetDomain: %v", err)
	}
	if domain.MonthlyCost != 99.5 || len(domain.Features) != 1 || domain.Features[0].NumPurchased != 2 {
		t.Errorf("domain = %+v", domain)
	}

	tenant, _, err := client.MSP.GetTenant(ctx, "protecttest")
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if tenant.CupUsed != 8649 || tenant.MonthlyCost != 400.99 {
		t.Errorf("tenant = %+v", tenant)
	}
}

func TestMSPService_PlansAndPowerUsers(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/msp/plans", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"count":1,"results":[{"id":1,"planId":10,"planName":"Example Plan",
			"maxPlanPuLicences":5,"costPerUser":10,"features":[{"name":"f1","type":"flat"}]}]}`))
	})
	mux.HandleFunc("/pubapi/v1/msp/plans/10/power-users/quote/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]int
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["powerUsers"] != 50 {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"previousPowerUsers":25,"newPowerUsers":50,"deltaPowerUsers":25,
			"deltaMonthly":"250.00","proratedCost":"125.50","proratedDaysLeft":15,"proratedMonthsLeft":0}`))
	})
	mux.HandleFunc("/pubapi/v1/msp/domains/testDomain/power-users/allocate", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		w.Write([]byte(`{"domain":"testDomain","previousPowerUsers":5,"newPowerUsers":10,"poolAvailableAfter":40}`))
	})

	ctx := context.Background()
	plans, _, err := client.MSP.ListPlans(ctx)
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if plans.Count != 1 || plans.Results[0].CostPerUser != 10 {
		t.Errorf("plans = %+v", plans)
	}

	quote, _, err := client.MSP.PowerUsersQuote(ctx, 10, 50)
	if err != nil {
		t.Fatalf("PowerUsersQuote: %v", err)
	}
	if quote.DeltaPowerUsers != 25 || quote.ProratedCost != "125.50" {
		t.Errorf("quote = %+v", quote)
	}

	allocation, _, err := client.MSP.AllocatePowerUsers(ctx, "testDomain", 10)
	if err != nil {
		t.Fatalf("AllocatePowerUsers: %v", err)
	}
	if allocation.PoolAvailableAfter != 40 {
		t.Errorf("allocation = %+v", allocation)
	}
}

func TestMSPService_Trials(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/msp/trials/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["domain"] != "acmetrial" || body["id"] != float64(42) || body["region"] != "us" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"domain":"acmetrial","domainUrl":"https://acmetrial.egnyte.com",
			"trialStart":"2026-06-01","trialEnd":"2026-06-30","powerUsers":25,"storageGB":500}`))
	})
	mux.HandleFunc("/pubapi/v1/msp/trials/acmetrial/activate/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]int
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["planId"] != 42 || body["storageGB"] != 1000 {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"domain":"acmetrial","domainUrl":"https://acmetrial.egnyte.com",
			"activationDate":"2026-07-01"}`))
	})

	ctx := context.Background()
	trial, resp, err := client.MSP.CreateTrial(ctx, MSPCreateTrialRequest{
		Domain: "acmetrial", ID: 42, PowerUsers: 25, StorageGB: 500, Region: "us",
	})
	if err != nil {
		t.Fatalf("CreateTrial: %v", err)
	}
	if resp.StatusCode != http.StatusCreated || trial.TrialEnd != "2026-06-30" {
		t.Errorf("trial = %+v, status = %d", trial, resp.StatusCode)
	}

	activation, _, err := client.MSP.ActivateTrial(ctx, "acmetrial", 42, 25, 1000)
	if err != nil {
		t.Fatalf("ActivateTrial: %v", err)
	}
	if activation.ActivationDate != "2026-07-01" {
		t.Errorf("activation = %+v", activation)
	}
}

func TestMSPService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.MSP.CreateWebhook(ctx, MSPWebhookRequest{WebhookURL: "https://x.com"}); err == nil {
		t.Error("missing eventTypes should fail before sending")
	}
	if _, _, err := client.MSP.PowerUsersQuote(ctx, 10, 0); err == nil {
		t.Error("zero powerUsers should fail before sending")
	}
	if _, _, err := client.MSP.CreateTrial(ctx, MSPCreateTrialRequest{Domain: "x"}); err == nil {
		t.Error("incomplete trial request should fail before sending")
	}
	if _, _, err := client.MSP.ActivateTrial(ctx, "d", 0, 1, 1); err == nil {
		t.Error("zero planId should fail before sending")
	}
}

func TestMSPService_CreateWebhook_limitExceeded(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/msp/webhooks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message":"Max webhook limit exceeded.","code":"webhook_limit_exceeded_error"}`))
	})

	_, _, err := client.MSP.CreateWebhook(context.Background(), MSPWebhookRequest{
		WebhookURL: "https://www.example.com/hook",
		EventTypes: []string{"RESELLER_MRR_CHANGED"},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Message != "Max webhook limit exceeded." {
		t.Errorf("APIError = %+v", apiErr)
	}
}
