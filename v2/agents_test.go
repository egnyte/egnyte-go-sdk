package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestAgentsService_List(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/agents/list", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("sortBy") != "createdOn" || q.Get("sortOrder") != "desc" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`[{"agentId":"f52a1253","name":"llm agent","description":"llm details",
			"status":"ACTIVE","subType":"USER_DEFINED","category":"GENERIC",
			"instruction":"llm details","createdBy":"John Millar"}]`))
	})

	agents, _, err := client.Agents.List(context.Background(), "createdOn", "desc")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(agents) != 1 || agents[0].AgentID != "f52a1253" || agents[0].Category != "GENERIC" {
		t.Errorf("agents = %+v", agents)
	}
}

func TestAgentsService_AskAndStatus(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/agents/f52a1253/ask", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		entryIDs, _ := body["entryIds"].([]any)
		if body["question"] != "Perform the data analysis" || len(entryIDs) != 1 {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"requestId":"12c02713","conversationId":"b5ec8d41"}`))
	})
	mux.HandleFunc("/pubapi/v1/ai/agents/f52a1253/ask/12c02713/status", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		w.Write([]byte(`{"status":"COMPLETED","responseText":"LLMs are a type of AI...",
			"citations":[{"previewUrl":"https://example.com/llm","type":"WEB_SEARCH"}],
			"lastUpdated":"2025-09-18T10:24:59Z"}`))
	})

	ctx := context.Background()
	execution, _, err := client.Agents.Ask(ctx, "f52a1253", AgentAskRequest{
		Question: "Perform the data analysis",
		EntryIDs: []string{"308314dd"},
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if execution.RequestID != "12c02713" || execution.ConversationID != "b5ec8d41" {
		t.Errorf("execution = %+v", execution)
	}

	status, _, err := client.Agents.AskStatus(ctx, "f52a1253", execution.RequestID)
	if err != nil {
		t.Fatalf("AskStatus: %v", err)
	}
	if status.Status != "COMPLETED" || len(status.Citations) != 1 ||
		status.Citations[0].Type != "WEB_SEARCH" {
		t.Errorf("status = %+v", status)
	}
}

func TestAgentsService_Ask_validation(t *testing.T) {
	client, _ := setup(t)
	if _, _, err := client.Agents.Ask(context.Background(), "a1", AgentAskRequest{}); err == nil {
		t.Error("empty question should fail before sending")
	}
}

func TestAgentsService_Ask_notFound(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/agents/nope/ask", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessage":"Agent not found","errorCode":"NOT_FOUND"}`))
	})

	_, _, err := client.Agents.Ask(context.Background(), "nope", AgentAskRequest{Question: "hi there"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("APIError = %+v", apiErr)
	}
}
