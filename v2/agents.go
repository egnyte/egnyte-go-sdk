package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AgentsService runs custom AI agents configured in the domain.
//
// API documentation: https://developers.egnyte.com/integration/cfs/api-docs/agent-api
// OAuth scope: Egnyte.ai. The Agents feature must also be enabled for the
// domain.
type AgentsService service

// AIAgent is one configured agent.
type AIAgent struct {
	AgentID     string `json:"agentId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`  // DRAFT or ACTIVE
	SubType     string `json:"subType"` // e.g. TRANSLATION_AGENT, USER_DEFINED
	// Category is TASK, QUESTION or GENERIC.
	Category    string `json:"category"`
	Instruction string `json:"instruction"`
	CreatedBy   string `json:"createdBy"`
}

// AgentFileRef references a context file for an agent question.
type AgentFileRef struct {
	EntryID  string `json:"entryId,omitempty"`
	FilePath string `json:"filePath,omitempty"`
}

// AgentSelectedItems pre-selects files and folders as agent context.
type AgentSelectedItems struct {
	Files   []AgentFileRef `json:"files,omitempty"`
	Folders []string       `json:"folders,omitempty"`
}

// AgentAskRequest is the body for asking an agent a question. Question
// is required (max 35,000 characters).
type AgentAskRequest struct {
	Question string `json:"question"`
	// Instructions optionally override the system prompt.
	Instructions string `json:"instructions,omitempty"`
	// ConversationID continues a multi-turn conversation.
	ConversationID string       `json:"conversationId,omitempty"`
	ChatHistory    *ChatHistory `json:"chatHistory,omitempty"`
	// EntryIDs provides file context.
	EntryIDs      []string            `json:"entryIds,omitempty"`
	SelectedItems *AgentSelectedItems `json:"selectedItems,omitempty"`
}

// AgentAsk is the async acknowledgement of an agent question.
type AgentAsk struct {
	// RequestID is used to poll AskStatus.
	RequestID      string `json:"requestId"`
	ConversationID string `json:"conversationId"`
}

// AgentCitation references a source used in an agent answer.
type AgentCitation struct {
	PreviewURL string `json:"previewUrl"`
	Type       string `json:"type"` // e.g. WEB_SEARCH
}

// AgentAskStatus is the state (and, once completed, the answer) of an
// agent execution.
type AgentAskStatus struct {
	// Status is PENDING, RUNNING, COMPLETED or FAILED.
	Status string `json:"status"`
	// ResponseText is present only when Status is COMPLETED.
	ResponseText string          `json:"responseText"`
	Citations    []AgentCitation `json:"citations"`
	LastUpdated  string          `json:"lastUpdated"`
}

// List returns the agents configured in the domain. sortBy is "name"
// (default) or "createdOn"; sortOrder is "asc" (default) or "desc".
//
// GET /pubapi/v1/ai/agents/list
func (s *AgentsService) List(ctx context.Context, sortBy, sortOrder string) ([]AIAgent, *Response, error) {
	urlPath := "v1/ai/agents/list"
	q := url.Values{}
	if sortBy != "" {
		q.Set("sortBy", sortBy)
	}
	if sortOrder != "" {
		q.Set("sortOrder", sortOrder)
	}
	if enc := q.Encode(); enc != "" {
		urlPath += "?" + enc
	}
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	var agents []AIAgent
	resp, err := s.client.Do(req, &agents)
	if err != nil {
		return nil, resp, err
	}
	return agents, resp, nil
}

// Ask submits a question to an agent and returns identifiers to poll
// with AskStatus.
//
// POST /pubapi/v1/ai/agents/{agentId}/ask
func (s *AgentsService) Ask(ctx context.Context, agentID string, ask AgentAskRequest) (*AgentAsk, *Response, error) {
	if ask.Question == "" {
		return nil, nil, fmt.Errorf("egnyte: question is required")
	}
	req, err := s.client.NewRequest(ctx, http.MethodPost, "v1/ai/agents/"+url.PathEscape(agentID)+"/ask", ask)
	if err != nil {
		return nil, nil, err
	}
	execution := new(AgentAsk)
	resp, err := s.client.Do(req, execution)
	if err != nil {
		return nil, resp, err
	}
	return execution, resp, nil
}

// AskStatus polls the result of an Ask execution. Poll until Status is
// COMPLETED or FAILED.
//
// GET /pubapi/v1/ai/agents/{agentId}/ask/{requestId}/status
func (s *AgentsService) AskStatus(ctx context.Context, agentID, requestID string) (*AgentAskStatus, *Response, error) {
	urlPath := "v1/ai/agents/" + url.PathEscape(agentID) + "/ask/" + url.PathEscape(requestID) + "/status"
	req, err := s.client.NewRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, nil, err
	}
	status := new(AgentAskStatus)
	resp, err := s.client.Do(req, status)
	if err != nil {
		return nil, resp, err
	}
	return status, resp, nil
}
