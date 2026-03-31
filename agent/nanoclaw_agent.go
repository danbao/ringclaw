package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultNanoClawEndpoint = "http://127.0.0.1:3929/message"

type NanoClawAgent struct {
	name         string
	endpoint     string
	apiKey       string
	headers      map[string]string
	model        string
	systemPrompt string
	cwd          string
	groupJID     string
	sender       string
	contextMode  string
	httpClient   *http.Client
	mu           sync.Mutex
}

type NanoClawAgentConfig struct {
	Name         string
	Endpoint     string
	APIKey       string
	Headers      map[string]string
	Model        string
	SystemPrompt string
	Cwd          string
	GroupJID     string
	Sender       string
	ContextMode  string
}

type nanoClawRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
	GroupJID       string `json:"group_jid,omitempty"`
	Sender         string `json:"sender,omitempty"`
	ContextMode    string `json:"context_mode,omitempty"`
	Cwd            string `json:"cwd,omitempty"`
	SystemPrompt   string `json:"system_prompt,omitempty"`
}

type nanoClawResponse struct {
	Reply string `json:"reply"`
}

func NewNanoClawAgent(cfg NanoClawAgentConfig) *NanoClawAgent {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultNanoClawEndpoint
	}
	contextMode := strings.TrimSpace(cfg.ContextMode)
	if contextMode == "" {
		contextMode = "group"
	}
	cwd := cfg.Cwd
	if cwd == "" {
		cwd = defaultWorkspace()
	}
	return &NanoClawAgent{
		name:         cfg.Name,
		endpoint:     endpoint,
		apiKey:       cfg.APIKey,
		headers:      cloneHeaders(cfg.Headers),
		model:        cfg.Model,
		systemPrompt: cfg.SystemPrompt,
		cwd:          cwd,
		groupJID:     strings.TrimSpace(cfg.GroupJID),
		sender:       strings.TrimSpace(cfg.Sender),
		contextMode:  contextMode,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *NanoClawAgent) Info() AgentInfo {
	return AgentInfo{
		Name:    a.name,
		Type:    "nanoclaw",
		Model:   a.model,
		Command: a.endpoint,
	}
}

func (a *NanoClawAgent) SetCwd(cwd string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cwd = cwd
}

func (a *NanoClawAgent) ResetSession(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (a *NanoClawAgent) Chat(ctx context.Context, conversationID string, message string) (string, error) {
	a.mu.Lock()
	payload := nanoClawRequest{
		ConversationID: conversationID,
		Message:        message,
		GroupJID:       a.groupJID,
		Sender:         a.sender,
		ContextMode:    a.contextMode,
		Cwd:            a.cwd,
		SystemPrompt:   a.systemPrompt,
	}
	endpoint := a.endpoint
	apiKey := a.apiKey
	headers := cloneHeaders(a.headers)
	a.mu.Unlock()

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal nanoclaw request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create nanoclaw request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("nanoclaw request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read nanoclaw response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("nanoclaw API error HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed nanoClawResponse
	if err := json.Unmarshal(respBody, &parsed); err == nil && strings.TrimSpace(parsed.Reply) != "" {
		return strings.TrimSpace(parsed.Reply), nil
	}

	trimmed := strings.TrimSpace(string(respBody))
	if trimmed == "" {
		return "", fmt.Errorf("nanoclaw returned empty response")
	}
	return trimmed, nil
}

func cloneHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
