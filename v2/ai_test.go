package egnyte

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestAIService_AskDocument(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/document/535083b1/ask", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["question"] != "What is the renewal date?" || body["includeCitations"] != true {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"response":{"text":"The renewal date is June 1."},
			"citations":[{"filename":"Policy.pdf","entryId":"535083b1",
			"chunks":[{"chunkId":"0","sourceText":"Policy Renewal Migration"}]}]}`))
	})

	answer, _, err := client.AI.AskDocument(context.Background(), "535083b1",
		"What is the renewal date?", &AskOptions{IncludeCitations: true})
	if err != nil {
		t.Fatalf("AskDocument: %v", err)
	}
	if answer.Response.Text != "The renewal date is June 1." || len(answer.Citations) != 1 {
		t.Errorf("answer = %+v", answer)
	}
	if answer.Citations[0].Chunks[0].SourceText != "Policy Renewal Migration" {
		t.Errorf("citation chunks = %+v", answer.Citations[0].Chunks)
	}
}

func TestAIService_SummarizeDocument(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/document/535083b1/summary", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		w.Write([]byte(`{"response":{"text":"Summary of the document is here"}}`))
	})

	summary, _, err := client.AI.SummarizeDocument(context.Background(), "535083b1", nil)
	if err != nil {
		t.Fatalf("SummarizeDocument: %v", err)
	}
	if summary != "Summary of the document is here" {
		t.Errorf("summary = %q", summary)
	}
}

func TestAIService_AskAssistantAndStatus(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/assistant/ask", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		selected, _ := body["selectedItems"].(map[string]any)
		files, _ := selected["files"].([]any)
		if body["question"] != "Describe the file" || len(files) != 1 {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"conversationId":"2862a232","executionId":"10cd770f",
			"executionStatus":"IN_PROGRESS","truncated":false,"actions":[]}`))
	})
	mux.HandleFunc("/pubapi/v1/ai/assistant/10cd770f/status", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		if got := r.URL.Query().Get("includeCitations"); got != "true" {
			t.Errorf("includeCitations = %q", got)
		}
		w.Write([]byte(`{"status":"COMPLETED","responseText":"Several premium examples",
			"citations":[{"filename":"Policy.pdf","entryId":"535083b1",
				"chunks":[{"chunkId":"c590a032","sourceText":"Text"}]}],
			"intent":"qna","lastUpdated":1780660757000,"numToolCalls":1,
			"toolCalls":{"call_abc123":{"name":"search_documents",
				"toolDisplayName":"Document Search","status":"success",
				"output":"Found 3 relevant documents"}}}`))
	})

	ctx := context.Background()
	execution, _, err := client.AI.AskAssistant(ctx, AssistantAskRequest{
		Question: "Describe the file",
		SelectedItems: &AISelectedItems{
			Files: []AIFileRef{{EntryID: "308314dd"}},
		},
		IncludeCitations: true,
	})
	if err != nil {
		t.Fatalf("AskAssistant: %v", err)
	}
	if execution.ExecutionID != "10cd770f" || execution.ExecutionStatus != "IN_PROGRESS" {
		t.Errorf("execution = %+v", execution)
	}

	status, _, err := client.AI.AssistantStatus(ctx, execution.ExecutionID, true)
	if err != nil {
		t.Fatalf("AssistantStatus: %v", err)
	}
	if status.Status != "COMPLETED" || status.ResponseText != "Several premium examples" {
		t.Errorf("status = %+v", status)
	}
	if call, ok := status.ToolCalls["call_abc123"]; !ok || call.Name != "search_documents" {
		t.Errorf("toolCalls = %+v", status.ToolCalls)
	}
	if len(status.Citations) != 1 || status.Citations[0].Chunks[0].SourceText != "Text" {
		t.Errorf("citations = %+v", status.Citations)
	}
}

func TestAIService_AskCopilot(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/copilot/ask", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["selectedItems"]; !ok {
			t.Error("selectedItems is required for copilot")
		}
		w.Write([]byte(`{"response":{"text":"Answer"},"conversationId":"c655a318",
			"deprecationNotice":"This endpoint is deprecated"}`))
	})

	answer, _, err := client.AI.AskCopilot(context.Background(), "Describe the file",
		AISelectedItems{Folders: []AIFolderRef{{ID: "d57858be"}}}, nil)
	if err != nil {
		t.Fatalf("AskCopilot: %v", err)
	}
	if answer.ConversationID != "c655a318" || answer.DeprecationNotice == "" {
		t.Errorf("answer = %+v", answer)
	}
}

func TestAIService_ListKnowledgeBases(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/kb/list", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		sortBy, _ := body["sortBy"].([]any)
		if len(sortBy) != 1 || sortBy[0] != "name" {
			t.Errorf("body = %v", body)
		}
		w.Write([]byte(`{"content":[{"id":"a691dcbf","name":"KB","status":"ACTIVE",
			"paths":[{"id":"8804124f","folderId":"0f53980d","path":"/Shared/Documents",
				"permission":"Owner","status":"ACTIVE"}],
			"createdByUser":{"firstName":"Ankesh","userId":1},
			"createdOn":1738570114257,"progress":100,"pathCount":1}],
			"totalElements":1,"totalPages":1,"first":true,"last":true,"size":200}`))
	})

	list, _, err := client.AI.ListKnowledgeBases(context.Background(), KnowledgeBaseListRequest{
		SortBy:        []string{"name"},
		SortDirection: []string{"ASC"},
		Status:        []string{"ACTIVE"},
	})
	if err != nil {
		t.Fatalf("ListKnowledgeBases: %v", err)
	}
	if list.TotalElements != 1 || list.Content[0].Name != "KB" || len(list.Content[0].Paths) != 1 {
		t.Errorf("list = %+v", list)
	}
}

func TestAIService_AskKnowledgeBase(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/kb/a691dcbf/ask", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		w.Write([]byte(`{"response":{"text":"42 documents"},"conversationId":"conv1"}`))
	})

	answer, _, err := client.AI.AskKnowledgeBase(context.Background(), "a691dcbf",
		"How many documents are stored in this KB?", nil)
	if err != nil {
		t.Fatalf("AskKnowledgeBase: %v", err)
	}
	if answer.Response.Text != "42 documents" {
		t.Errorf("answer = %+v", answer)
	}
}

func TestAIService_HybridSearch(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/hybrid-search", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["query"] != "quarterly revenue report" || body["semanticWeight"] != 0.7 ||
			body["createdBy"] != "jsmith" || body["limit"] != float64(5) {
			t.Errorf("body = %v", body)
		}
		folderPaths, _ := body["folderPaths"].([]any)
		if len(folderPaths) != 1 || folderPaths[0] != "/Shared/Finance" {
			t.Errorf("folderPaths = %v", folderPaths)
		}
		// Response shape captured from the live API.
		w.Write([]byte(`{"results":[{"uploadedTimestamp":1589869846507,
			"filename":"/Shared/Finance/Q4-Revenue.pdf","entryId":"535083b1","groupId":"223ba602",
			"chunks":[{"chunkId":"abc123","chunkText":"Total Q4 revenue was $4.2M",
			"type":"TEXT","score":87.843216}]}],"error":null}`))
	})

	weight := 0.7
	results, _, err := client.AI.HybridSearch(context.Background(), HybridSearchRequest{
		Query:          "quarterly revenue report",
		SemanticWeight: &weight,
		FolderPaths:    []string{"/Shared/Finance"},
		CreatedBy:      "jsmith",
		Limit:          5,
	})
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	if len(results.Results) != 1 {
		t.Fatalf("results = %+v", results)
	}
	r := results.Results[0]
	if r.Filename != "/Shared/Finance/Q4-Revenue.pdf" || r.GroupID != "223ba602" ||
		r.UploadedTimestamp != 1589869846507 {
		t.Errorf("result = %+v", r)
	}
	c := r.Chunks[0]
	if c.ChunkText != "Total Q4 revenue was $4.2M" || c.Type != "TEXT" || c.Score != 87.843216 {
		t.Errorf("chunk = %+v", c)
	}
}

func TestAIService_validation(t *testing.T) {
	client, _ := setup(t)
	ctx := context.Background()
	if _, _, err := client.AI.AskDocument(ctx, "e1", "", nil); err == nil {
		t.Error("empty question should fail before sending")
	}
	if _, _, err := client.AI.AskAssistant(ctx, AssistantAskRequest{}); err == nil {
		t.Error("empty assistant question should fail before sending")
	}
	if _, _, err := client.AI.ListKnowledgeBases(ctx, KnowledgeBaseListRequest{}); err == nil {
		t.Error("missing sort/status should fail before sending")
	}
	if _, _, err := client.AI.HybridSearch(ctx, HybridSearchRequest{}); err == nil {
		t.Error("empty query should fail before sending")
	}
	// The live API enforces query length 3-250 and limit 1-1000; the
	// client validates the same bounds before sending.
	if _, _, err := client.AI.HybridSearch(ctx, HybridSearchRequest{Query: "ab"}); err == nil {
		t.Error("2-char query should fail before sending")
	}
	if _, _, err := client.AI.HybridSearch(ctx, HybridSearchRequest{
		Query: strings.Repeat("q", 251)}); err == nil {
		t.Error("251-char query should fail before sending")
	}
	if _, _, err := client.AI.HybridSearch(ctx, HybridSearchRequest{
		Query: "valid query", Limit: 1001}); err == nil {
		t.Error("limit over 1000 should fail before sending")
	}
}

func TestAIService_AskDocument_rateLimited(t *testing.T) {
	client, mux := setup(t)
	mux.HandleFunc("/pubapi/v1/ai/document/e1/ask", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"AI rate limit exceeded"}`))
	})

	_, _, err := client.AI.AskDocument(context.Background(), "e1", "question?", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if !apiErr.IsRateLimit() || apiErr.RetryAfter == 0 {
		t.Errorf("APIError = %+v", apiErr)
	}
}
