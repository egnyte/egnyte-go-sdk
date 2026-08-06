package egnyte

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// AIService exposes Egnyte's AI capabilities: document Q&A and
// summaries, the AI Assistant, knowledge bases and hybrid search. AI
// endpoints have separate, lower rate limits.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/ai-api
// OAuth scope: Egnyte.ai.
type AIService service

// ChatMessage is one message of a conversation history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatHistory carries previous conversation messages.
type ChatHistory struct {
	Messages []ChatMessage `json:"messages"`
}

// AIText is the nested text payload of AI responses.
type AIText struct {
	Text string `json:"text"`
}

// CitationChunk is one cited text fragment.
type CitationChunk struct {
	ChunkID string `json:"chunkId"`
	// SourceText is the cited passage (the live API sends sourceText,
	// not the spec's text).
	SourceText string `json:"sourceText"`
}

// Citation references a source document backing an AI answer.
type Citation struct {
	Filename string          `json:"filename"`
	EntryID  string          `json:"entryId"`
	ChunkID  string          `json:"chunkId"`
	Text     string          `json:"text"`
	Chunks   []CitationChunk `json:"chunks"`
}

// AskOptions are the optional parts of an ask request.
type AskOptions struct {
	IncludeCitations bool
	ChatHistory      *ChatHistory
}

// DocumentAnswer is the response to AskDocument.
type DocumentAnswer struct {
	Response  AIText     `json:"response"`
	Citations []Citation `json:"citations"`
}

// AskDocument answers a question from the contents of the document
// version with the given entry id.
//
// POST /pubapi/v1/ai/document/{entry-id}/ask
func (s *AIService) AskDocument(ctx context.Context, entryID, question string, opts *AskOptions) (*DocumentAnswer, *Response, error) {
	if question == "" {
		return nil, nil, fmt.Errorf("egnyte: question is required")
	}
	body := struct {
		Question         string       `json:"question"`
		IncludeCitations bool         `json:"includeCitations,omitempty"`
		ChatHistory      *ChatHistory `json:"chatHistory,omitempty"`
	}{Question: question}
	if opts != nil {
		body.IncludeCitations = opts.IncludeCitations
		body.ChatHistory = opts.ChatHistory
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/ai/document/"+url.PathEscape(entryID)+"/ask", body)
	if err != nil {
		return nil, nil, err
	}
	answer := new(DocumentAnswer)
	resp, err := s.client.Do(req, answer)
	if err != nil {
		return nil, resp, err
	}
	return answer, resp, nil
}

// SummarizeDocument returns a concise summary of the document version
// with the given entry id. chatHistory optionally provides context.
//
// POST /pubapi/v1/ai/document/{entry-id}/summary
func (s *AIService) SummarizeDocument(ctx context.Context, entryID string, chatHistory *ChatHistory) (string, *Response, error) {
	var body any
	if chatHistory != nil {
		body = struct {
			ChatHistory *ChatHistory `json:"chatHistory"`
		}{chatHistory}
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/ai/document/"+url.PathEscape(entryID)+"/summary", body)
	if err != nil {
		return "", nil, err
	}
	var summary struct {
		Response AIText `json:"response"`
	}
	resp, err := s.client.Do(req, &summary)
	if err != nil {
		return "", resp, err
	}
	return summary.Response.Text, resp, nil
}

// AIFolderRef and AIFileRef scope assistant questions to content.
type AIFolderRef struct {
	ID string `json:"id"`
}

// AIFileRef references a file by entry id.
type AIFileRef struct {
	EntryID string `json:"entryId"`
}

// AISelectedItems scope the assistant's context.
type AISelectedItems struct {
	Folders []AIFolderRef `json:"folders,omitempty"`
	Files   []AIFileRef   `json:"files,omitempty"`
	// AllEgnyteSearch searches across all content in the domain.
	AllEgnyteSearch *bool `json:"allEgnyteSearch,omitempty"`
	// WebSearch includes web search results.
	WebSearch *bool `json:"webSearch,omitempty"`
}

// AIModelDetails override the model used for a request.
type AIModelDetails struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"` // e.g. gemini-2.5-flash
}

// AssistantAskRequest is the body for the async AI Assistant.
type AssistantAskRequest struct {
	Question         string           `json:"question"`
	SelectedItems    *AISelectedItems `json:"selectedItems,omitempty"`
	IncludeCitations bool             `json:"includeCitations,omitempty"`
	ChatHistory      *ChatHistory     `json:"chatHistory,omitempty"`
	// ConversationID continues a previous conversation.
	ConversationID string `json:"conversationId,omitempty"`
	// MCPSelectionID selects an MCP server configuration.
	MCPSelectionID string          `json:"mcpSelectionId,omitempty"`
	ModelDetails   *AIModelDetails `json:"modelDetails,omitempty"`
}

// AssistantExecution is the async acknowledgement of an assistant ask.
type AssistantExecution struct {
	ConversationID string `json:"conversationId"`
	// ExecutionID is used to poll AssistantStatus.
	ExecutionID     string           `json:"executionId"`
	ExecutionStatus string           `json:"executionStatus"` // e.g. IN_PROGRESS
	Truncated       bool             `json:"truncated"`
	Actions         []map[string]any `json:"actions"`
}

// AskAssistant submits a question to the AI Assistant and returns an
// execution to poll with AssistantStatus. Replaces the deprecated
// AskCopilot; the answer is retrieved asynchronously.
//
// POST /pubapi/v1/ai/assistant/ask
func (s *AIService) AskAssistant(ctx context.Context, ask AssistantAskRequest) (*AssistantExecution, *Response, error) {
	if ask.Question == "" {
		return nil, nil, fmt.Errorf("egnyte: question is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/ai/assistant/ask", ask)
	if err != nil {
		return nil, nil, err
	}
	execution := new(AssistantExecution)
	resp, err := s.client.Do(req, execution)
	if err != nil {
		return nil, resp, err
	}
	return execution, resp, nil
}

// AssistantCitationChunk is one cited fragment of an assistant answer.
type AssistantCitationChunk struct {
	ChunkID    string `json:"chunkId"`
	SourceText string `json:"sourceText"`
}

// AssistantCitation references a source document of an assistant answer.
type AssistantCitation struct {
	Filename   string                   `json:"filename"`
	EntryID    string                   `json:"entryId"`
	ObjectID   string                   `json:"objectId"`
	PreviewURL string                   `json:"previewUrl"`
	Chunks     []AssistantCitationChunk `json:"chunks"`
}

// AIToolCall describes one tool invocation made by the assistant.
type AIToolCall struct {
	Name              string         `json:"name"`
	SourceDisplayName string         `json:"sourceDisplayName"`
	ToolDisplayName   string         `json:"toolDisplayName"`
	Thought           string         `json:"thought"`
	Thoughts          []string       `json:"thoughts"`
	Input             map[string]any `json:"input"`
	Output            any            `json:"output"`
	Status            string         `json:"status"`
	Timestamp         string         `json:"timestamp"`
	Metadata          map[string]any `json:"metadata"`
}

// AIPendingAction is an action awaiting user confirmation.
type AIPendingAction struct {
	Type       string `json:"type"`
	ServerID   string `json:"serverId"`
	ToolCallID string `json:"toolCallId"`
}

// AssistantStatus is the state (and, once completed, the answer) of an
// assistant execution.
type AssistantStatus struct {
	// Status is IN_PROGRESS, COMPLETED, FAILED or
	// AWAITING_USER_CONFIRMATION.
	Status string `json:"status"`
	// ResponseText is the answer, populated when Status is COMPLETED.
	ResponseText string              `json:"responseText"`
	Thoughts     []string            `json:"thoughts"`
	Citations    []AssistantCitation `json:"citations"`
	Actions      []map[string]any    `json:"actions"`
	Intent       string              `json:"intent"`
	// LastUpdated is Unix epoch milliseconds.
	LastUpdated    int64                 `json:"lastUpdated"`
	NumToolCalls   int                   `json:"numToolCalls"`
	PendingActions []AIPendingAction     `json:"pendingActions"`
	ToolCalls      map[string]AIToolCall `json:"toolCalls"`
}

// AssistantStatus polls the result of an AskAssistant execution. Poll
// until Status is no longer IN_PROGRESS.
//
// GET /pubapi/v1/ai/assistant/{executionId}/status
func (s *AIService) AssistantStatus(ctx context.Context, executionID string, includeCitations bool) (*AssistantStatus, *Response, error) {
	urlPath := "v1/ai/assistant/" + url.PathEscape(executionID) + "/status?" +
		url.Values{"includeCitations": {strconv.FormatBool(includeCitations)}}.Encode()
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	status := new(AssistantStatus)
	resp, err := s.client.Do(req, status)
	if err != nil {
		return nil, resp, err
	}
	return status, resp, nil
}

// CopilotCitation references a source document of a copilot answer.
type CopilotCitation struct {
	Filename   string          `json:"filename"`
	EntryID    string          `json:"entryId"`
	ObjectID   string          `json:"objectId"`
	PreviewURL string          `json:"previewUrl"`
	Chunks     []CitationChunk `json:"chunks"`
}

// CopilotAnswer is the synchronous copilot response.
type CopilotAnswer struct {
	Response          AIText            `json:"response"`
	Citations         []CopilotCitation `json:"citations"`
	ConversationID    string            `json:"conversationId"`
	DeprecationNotice string            `json:"deprecationNotice"`
}

// AskCopilot asks Egnyte Copilot a question scoped to the selected
// folders and files, returning the answer synchronously.
//
// Deprecated: the endpoint will be removed on September 30, 2026 — use
// AskAssistant with AssistantStatus polling instead.
//
// POST /pubapi/v1/ai/copilot/ask
func (s *AIService) AskCopilot(ctx context.Context, question string, selectedItems AISelectedItems, opts *AskOptions) (*CopilotAnswer, *Response, error) {
	if question == "" {
		return nil, nil, fmt.Errorf("egnyte: question is required")
	}
	body := struct {
		Question         string          `json:"question"`
		SelectedItems    AISelectedItems `json:"selectedItems"`
		IncludeCitations bool            `json:"includeCitations,omitempty"`
		ChatHistory      *ChatHistory    `json:"chatHistory,omitempty"`
	}{Question: question, SelectedItems: selectedItems}
	if opts != nil {
		body.IncludeCitations = opts.IncludeCitations
		body.ChatHistory = opts.ChatHistory
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/ai/copilot/ask", body)
	if err != nil {
		return nil, nil, err
	}
	answer := new(CopilotAnswer)
	resp, err := s.client.Do(req, answer)
	if err != nil {
		return nil, resp, err
	}
	return answer, resp, nil
}

// KnowledgeBasePath is one folder included in a knowledge base.
type KnowledgeBasePath struct {
	ID         string `json:"id"`
	FolderID   string `json:"folderId"`
	Path       string `json:"path"`
	Permission string `json:"permission"`
	Status     string `json:"status"`
}

// KnowledgeBaseUser identifies the creator of a knowledge base.
type KnowledgeBaseUser struct {
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	UserName          string `json:"userName"`
	UserID            int    `json:"userId"`
	AvatarEOSObjectID string `json:"avatarEosObjectId"`
}

// KnowledgeBase is a curated content collection for AI Q&A.
type KnowledgeBase struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Paths             []KnowledgeBasePath `json:"paths"`
	Status            string              `json:"status"` // ACTIVE, DELETED or CREATED
	Type              string              `json:"type"`
	CreatedBy         string              `json:"createdBy"`
	CreatedByUser     *KnowledgeBaseUser  `json:"createdByUser"`
	CreatedOn         int64               `json:"createdOn"`
	NoResponseMessage string              `json:"noResponseMessage"`
	IconName          string              `json:"iconName"`
	SubType           string              `json:"subType"`
	Progress          int                 `json:"progress"`
	LastProcessedAt   int64               `json:"lastProcessedAt"`
	PathCount         int                 `json:"pathCount"`
	Prompts           []map[string]any    `json:"prompts"`
	SubStatus         string              `json:"subStatus"`
}

// KnowledgeBaseListRequest filters and paginates knowledge base
// listings. SortBy, SortDirection and Status are required by the API.
type KnowledgeBaseListRequest struct {
	SortBy                      []string `json:"sortBy"`        // createdOn, name
	SortDirection               []string `json:"sortDirection"` // ASC, DESC
	Status                      []string `json:"status"`        // ACTIVE, DELETED, CREATED
	Page                        int      `json:"page,omitempty"`
	Size                        int      `json:"size,omitempty"`
	CreatedBy                   int      `json:"createdBy,omitempty"`
	CreatedAfter                int64    `json:"createdAfter,omitempty"`
	CreatedBefore               int64    `json:"createdBefore,omitempty"`
	IncludePlaceholderData      bool     `json:"includePlaceholderData,omitempty"`
	IncludeProcessingStatistics bool     `json:"includeProcessingStatistics,omitempty"`
	IncludePrompts              bool     `json:"includePrompts,omitempty"`
}

// KnowledgeBaseList is a page of knowledge bases.
type KnowledgeBaseList struct {
	Content          []KnowledgeBase `json:"content"`
	Number           int             `json:"number"`
	Size             int             `json:"size"`
	First            bool            `json:"first"`
	Last             bool            `json:"last"`
	Empty            bool            `json:"empty"`
	NumberOfElements int             `json:"numberOfElements"`
	TotalElements    int             `json:"totalElements"`
	TotalPages       int             `json:"totalPages"`
	FileLimit        int             `json:"fileLimit"`
}

// ListKnowledgeBases returns the knowledge bases in the domain.
//
// POST /pubapi/v1/ai/kb/list
func (s *AIService) ListKnowledgeBases(ctx context.Context, list KnowledgeBaseListRequest) (*KnowledgeBaseList, *Response, error) {
	if len(list.SortBy) == 0 || len(list.SortDirection) == 0 || len(list.Status) == 0 {
		return nil, nil, fmt.Errorf("egnyte: sortBy, sortDirection and status are required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/ai/kb/list", list)
	if err != nil {
		return nil, nil, err
	}
	result := new(KnowledgeBaseList)
	resp, err := s.client.Do(req, result)
	if err != nil {
		return nil, resp, err
	}
	return result, resp, nil
}

// KnowledgeBaseAnswer is the response to AskKnowledgeBase.
type KnowledgeBaseAnswer struct {
	Response       AIText            `json:"response"`
	Citations      []CopilotCitation `json:"citations"`
	ConversationID string            `json:"conversationId"`
}

// AskKnowledgeBase answers a question from the contents of a knowledge
// base.
//
// POST /pubapi/v1/ai/kb/{kb-id}/ask
func (s *AIService) AskKnowledgeBase(ctx context.Context, kbID, question string, opts *AskOptions) (*KnowledgeBaseAnswer, *Response, error) {
	if question == "" {
		return nil, nil, fmt.Errorf("egnyte: question is required")
	}
	body := struct {
		Question         string       `json:"question"`
		IncludeCitations bool         `json:"includeCitations,omitempty"`
		ChatHistory      *ChatHistory `json:"chatHistory,omitempty"`
	}{Question: question}
	if opts != nil {
		body.IncludeCitations = opts.IncludeCitations
		body.ChatHistory = opts.ChatHistory
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/ai/kb/"+url.PathEscape(kbID)+"/ask", body)
	if err != nil {
		return nil, nil, err
	}
	answer := new(KnowledgeBaseAnswer)
	resp, err := s.client.Do(req, answer)
	if err != nil {
		return nil, resp, err
	}
	return answer, resp, nil
}

// HybridSearchRequest combines semantic and keyword retrieval. The
// endpoint rejects unknown properties, so every field here is validated
// against the live API.
type HybridSearchRequest struct {
	// Query is the search phrase, 3-250 characters. Required.
	Query string `json:"query"`
	// SemanticWeight balances semantic vs keyword search: 0 = keyword
	// only, 1 = semantic only. Optional.
	SemanticWeight *float64 `json:"semanticWeight,omitempty"`
	// Limit caps the number of results, 1-1000 (default 100).
	Limit int `json:"limit,omitempty"`
	// CollectionId restricts the search to a collection.
	CollectionID string `json:"collectionId,omitempty"`
	// CreatedBy filters by the uploading user's username.
	CreatedBy string `json:"createdBy,omitempty"`
	// CreatedAfter / CreatedBefore bound the upload time (epoch ms).
	CreatedAfter  int64 `json:"createdAfter,omitempty"`
	CreatedBefore int64 `json:"createdBefore,omitempty"`
	// FolderPaths restricts the search to these folders; FolderPath is
	// the single-folder equivalent (both are accepted by the API).
	FolderPaths []string `json:"folderPaths,omitempty"`
	FolderPath  string   `json:"folderPath,omitempty"`
	// PreferredFolderPath boosts results from a folder without
	// excluding others.
	PreferredFolderPath string `json:"preferredFolderPath,omitempty"`
	// ExcludeFolderPaths removes folders from the search scope.
	ExcludeFolderPaths []string `json:"excludeFolderPaths,omitempty"`
	// EntryIDs restricts the search to specific file versions.
	EntryIDs []string `json:"entryIds,omitempty"`
}

// HybridSearchChunk is one ranked text fragment.
type HybridSearchChunk struct {
	ChunkID string `json:"chunkId"`
	// ChunkText is the matched passage.
	ChunkText string `json:"chunkText"`
	// Type is TEXT or TABLE.
	Type  string  `json:"type"`
	Score float64 `json:"score"`
}

// HybridSearchResult is one matching file with its ranked chunks.
type HybridSearchResult struct {
	// Filename is the full path of the file.
	Filename string `json:"filename"`
	EntryID  string `json:"entryId"`
	GroupID  string `json:"groupId"`
	// UploadedTimestamp is the upload time in epoch milliseconds.
	UploadedTimestamp int64               `json:"uploadedTimestamp"`
	Chunks            []HybridSearchChunk `json:"chunks"`
}

// HybridSearchResults is the hybrid search response.
type HybridSearchResults struct {
	Results []HybridSearchResult `json:"results"`
	// Error is null on success; its shape on partial failure is not
	// documented, so the raw JSON is preserved.
	Error json.RawMessage `json:"error"`
}

// HybridSearch performs a combined semantic and keyword search across
// the domain's AI-indexed content.
//
// POST /pubapi/v1/hybrid-search
func (s *AIService) HybridSearch(ctx context.Context, search HybridSearchRequest) (*HybridSearchResults, *Response, error) {
	if n := len([]rune(search.Query)); n < 3 || n > 250 {
		return nil, nil, fmt.Errorf("egnyte: query must be 3-250 characters, got %d", n)
	}
	if search.Limit < 0 || search.Limit > 1000 {
		return nil, nil, fmt.Errorf("egnyte: limit must be 1-1000, got %d", search.Limit)
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/hybrid-search", search)
	if err != nil {
		return nil, nil, err
	}
	results := new(HybridSearchResults)
	resp, err := s.client.Do(req, results)
	if err != nil {
		return nil, resp, err
	}
	return results, resp, nil
}
