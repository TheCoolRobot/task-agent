// Package ai provides a unified interface for multiple AI providers.
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Provider defines an AI provider's metadata.
type Provider struct {
	ID           string
	Name         string
	Models       []string
	DefaultModel string
	EnvKey       string
	BaseURL      string
}

// Providers lists all registered providers.
var Providers = []Provider{
	{
		ID:           "anthropic",
		Name:         "Anthropic",
		Models:       []string{"claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5-20251001"},
		DefaultModel: "claude-sonnet-4-6",
		EnvKey:       "ANTHROPIC_API_KEY",
		BaseURL:      "https://api.anthropic.com/v1",
	},
	{
		ID:           "openai",
		Name:         "OpenAI",
		Models:       []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1", "o3-mini"},
		DefaultModel: "gpt-4o",
		EnvKey:       "OPENAI_API_KEY",
		BaseURL:      "https://api.openai.com/v1",
	},
	{
		ID:           "groq",
		Name:         "Groq",
		Models:       []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768", "gemma2-9b-it"},
		DefaultModel: "llama-3.3-70b-versatile",
		EnvKey:       "GROQ_API_KEY",
		BaseURL:      "https://api.groq.com/openai/v1",
	},
	{
		ID:           "moonshot",
		Name:         "Moonshot (Kimi)",
		Models:       []string{"kimi-k2-0711-preview", "moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"},
		DefaultModel: "kimi-k2-0711-preview",
		EnvKey:       "MOONSHOT_API_KEY",
		BaseURL:      "https://api.moonshot.cn/v1",
	},
	{
		ID:           "ollama",
		Name:         "Ollama (Local)",
		Models:       []string{"llama3.3", "llama3.1", "qwen2.5-coder", "mistral", "codellama", "phi4","gemma3:1b"},
		DefaultModel: "llama3.3",
		EnvKey:       "",
		BaseURL:      "http://localhost:11434/v1",
	},
}

// GetProvider returns a provider by ID.
func GetProvider(id string) (Provider, bool) {
	for _, p := range Providers {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

// OutputFile is a single file produced by the AI.
type OutputFile struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

// TaskResult is the structured response from the AI.
type TaskResult struct {
	OutputType string       `json:"output_type"` // "markdown" | "code_folder" | "mixed"
	Summary    string       `json:"summary"`
	Files      []OutputFile `json:"files"`
	Notes      string       `json:"notes"`
}

const yoloSystemPrompt = `You are an autonomous task execution agent operating in YOLO mode.
You receive an Asana task and execute it completely and thoroughly.

## Output Format
Respond ONLY with a valid JSON object (no markdown fences, no preamble):
{
  "output_type": "markdown" | "code_folder" | "mixed",
  "summary": "Brief summary of what you did",
  "files": [
    {
      "path": "relative/path/to/file.ext",
      "content": "full file content here",
      "description": "what this file does"
    }
  ],
  "notes": "Any important notes, caveats, or follow-up suggestions"
}

## Rules
- Be thorough — never produce partial deliverables
- For code tasks: include tests, a README, and proper project structure
- For writing tasks: produce publication-ready content
- Make reasonable assumptions and document them in "notes"
- ALWAYS produce actual file content, never just describe what to do`

// Client sends tasks to AI providers.
type Client struct {
	ProviderID string
	Model      string
	APIKey     string
	httpClient *http.Client
}

// NewClient creates a new AI client.
func NewClient(providerID, model, apiKey string) *Client {
	return &Client{
		ProviderID: providerID,
		Model:      model,
		APIKey:     apiKey,
		httpClient: &http.Client{Timeout: 180 * time.Second},
	}
}

// ExecuteTask sends a task to the AI and returns structured output.
// The progress func is called with status strings during execution — these
// are sent into the TUI live log via a channel.
func (c *Client) ExecuteTask(taskMarkdown string, progress func(string)) (*TaskResult, error) {
	userContent := fmt.Sprintf(
		"## Asana Task\n\n%s\n\nExecute this task completely. Return valid JSON as specified.",
		taskMarkdown,
	)

	emit := func(s string) {
		if progress != nil {
			progress(s)
		}
	}

	emit(fmt.Sprintf("Calling %s / %s…", c.ProviderID, c.Model))

	var (
		raw string
		err error
	)

	switch c.ProviderID {
	case "anthropic":
		raw, err = c.callAnthropic(userContent, emit)
	default:
		raw, err = c.callOpenAICompat(userContent, emit)
	}
	if err != nil {
		return nil, err
	}

	emit("Parsing response…")
	result, parseErr := parseResult(raw)
	if parseErr == nil {
		emit(fmt.Sprintf("Got %d file(s) — output type: %s", len(result.Files), result.OutputType))
	}
	return result, parseErr
}

// ─── Anthropic ───────────────────────────────────────────────────────────────

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *Client) callAnthropic(userContent string, progress func(string)) (string, error) {
	progress("Sending request to Anthropic…")
	body := anthropicRequest{
		Model:     c.Model,
		MaxTokens: 8192,
		System:    yoloSystemPrompt,
		Messages:  []anthropicMessage{{Role: "user", Content: userContent}},
	}
	resp, err := c.post("https://api.anthropic.com/v1/messages", map[string]string{
		"x-api-key":         c.APIKey,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return "", err
	}
	var ar anthropicResponse
	if err := json.Unmarshal(resp, &ar); err != nil {
		return "", fmt.Errorf("unmarshal anthropic response: %w", err)
	}
	if ar.Error != nil {
		return "", fmt.Errorf("Anthropic API: %s", ar.Error.Message)
	}
	if len(ar.Content) == 0 {
		return "", fmt.Errorf("empty response from Anthropic")
	}
	progress(fmt.Sprintf("Received %d input / %d output tokens", ar.Usage.InputTokens, ar.Usage.OutputTokens))
	return ar.Content[0].Text, nil
}

// ─── OpenAI-compatible (OpenAI, Groq, Ollama) ────────────────────────────────

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (c *Client) callOpenAICompat(userContent string, progress func(string)) (string, error) {
	prov, ok := GetProvider(c.ProviderID)
	if !ok {
		return "", fmt.Errorf("unknown provider: %s", c.ProviderID)
	}
	progress(fmt.Sprintf("Sending request to %s…", prov.Name))
	body := openAIRequest{
		Model: c.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: yoloSystemPrompt},
			{Role: "user", Content: userContent},
		},
		MaxTokens:   8192,
		Temperature: 0.7,
	}
	apiKey := c.APIKey
	if c.ProviderID == "ollama" {
		apiKey = "ollama"
	}
	resp, err := c.post(prov.BaseURL+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + apiKey,
	}, body)
	if err != nil {
		return "", err
	}
	var or openAIResponse
	if err := json.Unmarshal(resp, &or); err != nil {
		return "", fmt.Errorf("unmarshal %s response: %w", c.ProviderID, err)
	}
	if or.Error != nil {
		return "", fmt.Errorf("%s API: %s", prov.Name, or.Error.Message)
	}
	if len(or.Choices) == 0 {
		return "", fmt.Errorf("empty response from %s", prov.Name)
	}
	progress(fmt.Sprintf("Received %d prompt / %d completion tokens",
		or.Usage.PromptTokens, or.Usage.CompletionTokens))
	return or.Choices[0].Message.Content, nil
}

// ─── HTTP helper ─────────────────────────────────────────────────────────────

func (c *Client) post(url string, extraHeaders map[string]string, body any) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to %s: %w", url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, string(b))
	}
	return b, nil
}

// ─── Response parser ─────────────────────────────────────────────────────────

func parseResult(raw string) (*TaskResult, error) {
	text := strings.TrimSpace(raw)
	// Strip markdown code fences
	if strings.HasPrefix(text, "```") {
		lines := strings.SplitN(text, "\n", 2)
		if len(lines) == 2 {
			text = lines[1]
		}
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
		text = strings.TrimSpace(text)
	}
	var result TaskResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		// Fallback: wrap raw as markdown
		return &TaskResult{
			OutputType: "markdown",
			Summary:    "AI response (raw — JSON parsing failed)",
			Files:      []OutputFile{{Path: "output.md", Content: raw, Description: "Raw AI output"}},
			Notes:      fmt.Sprintf("JSON parse error: %v", err),
		}, nil
	}
	return &result, nil
}