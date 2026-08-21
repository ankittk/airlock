package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Request is a provider-neutral generation request.
type Request struct {
	Provider string
	Model    string
	Messages []Message
	Tools    []Tool
	Seed     *int64
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type Response struct {
	Text      string          `json:"text"`
	ToolCalls []ToolCall      `json:"tool_calls,omitempty"`
	RawJSON   json.RawMessage `json:"raw_json,omitempty"`
	LatencyMS int64           `json:"latency_ms"`
	Model     string          `json:"model,omitempty"`
	CostUSD   float64         `json:"cost_usd,omitempty"` // estimated; mock uses flat fee
}

type ToolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type Provider interface {
	Name() string
	Generate(ctx context.Context, req Request) (Response, error)
}

// HTTPDoer allows cassette RoundTripper injection.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 120 * time.Second}
}

// Resolve picks a provider by name; "mock" always available.
func Resolve(name string, client HTTPDoer) (Provider, error) {
	if client == nil {
		client = DefaultHTTPClient()
	}
	switch strings.ToLower(name) {
	case "", "mock":
		return &Mock{}, nil
	case "openai", "ollama":
		return NewOpenAI(client, name == "ollama"), nil
	case "anthropic":
		return NewAnthropic(client), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

// Mock returns deterministic responses for offline tests.
type Mock struct{}

func (m *Mock) Name() string { return "mock" }

func (m *Mock) Generate(ctx context.Context, req Request) (Response, error) {
	_ = ctx
	// If tools present and last user message mentions "search", emit tool call.
	user := lastUser(req.Messages)
	const mockCost = 0.0001
	if len(req.Tools) > 0 && strings.Contains(strings.ToLower(user), "order") {
		args := json.RawMessage(`{"order_id":"ORD-1","date_from":"2024-01-01","date_to":"2024-01-31"}`)
		return Response{
			Text: "",
			ToolCalls: []ToolCall{{
				Name: req.Tools[0].Name,
				Args: args,
			}},
			RawJSON: json.RawMessage(`{"mock":true,"tool":true}`),
			CostUSD: mockCost,
		}, nil
	}
	text := `{"ok":true,"answer":"mock"}`
	if strings.Contains(user, "exact:") {
		text = strings.TrimSpace(strings.SplitN(user, "exact:", 2)[1])
	}
	return Response{
		Text:    text,
		RawJSON: json.RawMessage(fmt.Sprintf(`{"mock":true,"text":%q}`, text)),
		CostUSD: mockCost,
	}, nil
}

func lastUser(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

// --- OpenAI-compatible ---

type OpenAI struct {
	client  HTTPDoer
	baseURL string
	apiKey  string
	name    string
}

func NewOpenAI(client HTTPDoer, ollama bool) *OpenAI {
	base := os.Getenv("OPENAI_BASE_URL")
	key := os.Getenv("OPENAI_API_KEY")
	name := "openai"
	if ollama {
		name = "ollama"
		if base == "" {
			base = "http://127.0.0.1:11434/v1"
		}
		if key == "" {
			key = "ollama"
		}
	}
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &OpenAI{client: client, baseURL: strings.TrimRight(base, "/"), apiKey: key, name: name}
}

func (o *OpenAI) Name() string { return o.name }

func (o *OpenAI) Generate(ctx context.Context, req Request) (Response, error) {
	if o.apiKey == "" {
		return Response{}, fmt.Errorf("OPENAI_API_KEY not set")
	}
	model := req.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	body := map[string]any{
		"model":    model,
		"messages": req.Messages,
	}
	if len(req.Tools) > 0 {
		var tools []map[string]any
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  json.RawMessage(orEmptyObj(t.Parameters)),
				},
			})
		}
		body["tools"] = tools
	}
	if req.Seed != nil {
		body["seed"] = *req.Seed
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	start := time.Now()
	resp, err := o.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("openai: %s: %s", resp.Status, truncate(data, 200))
	}
	var parsed openaiResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return Response{}, err
	}
	out := Response{RawJSON: data, LatencyMS: time.Since(start).Milliseconds(), Model: model}
	if len(parsed.Choices) > 0 {
		c := parsed.Choices[0].Message
		out.Text = c.Content
		for _, tc := range c.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				Name: tc.Function.Name,
				Args: json.RawMessage(tc.Function.Arguments),
			})
		}
	}
	return out, nil
}

type openaiResp struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// --- Anthropic ---

type Anthropic struct {
	client HTTPDoer
	apiKey string
}

func NewAnthropic(client HTTPDoer) *Anthropic {
	return &Anthropic{client: client, apiKey: os.Getenv("ANTHROPIC_API_KEY")}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Generate(ctx context.Context, req Request) (Response, error) {
	if a.apiKey == "" {
		return Response{}, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}
	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	var system string
	var msgs []map[string]string
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		msgs = append(msgs, map[string]string{"role": role, "content": m.Content})
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"messages":   msgs,
	}
	if system != "" {
		body["system"] = system
	}
	if len(req.Tools) > 0 {
		var tools []map[string]any
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": json.RawMessage(orEmptyObj(t.Parameters)),
			})
		}
		body["tools"] = tools
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	start := time.Now()
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("anthropic: %s: %s", resp.Status, truncate(data, 200))
	}
	var parsed anthropicResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return Response{}, err
	}
	out := Response{RawJSON: data, LatencyMS: time.Since(start).Milliseconds(), Model: model}
	for _, b := range parsed.Content {
		switch b.Type {
		case "text":
			out.Text += b.Text
		case "tool_use":
			out.ToolCalls = append(out.ToolCalls, ToolCall{Name: b.Name, Args: b.Input})
		}
	}
	return out, nil
}

type anthropicResp struct {
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
}

func orEmptyObj(b json.RawMessage) []byte {
	if len(b) == 0 {
		return []byte(`{"type":"object","properties":{}}`)
	}
	return b
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
