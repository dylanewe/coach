package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dylanewe/coach/internal/adapters/llm/prompts"
	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/ports"
)

// Client implements ports.Analyzer for a local Ollama server.
type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

// NewClient creates a new Ollama analyzer.
func NewClient(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "gemma4:12b"
	}
	return &Client{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 5 * time.Minute},
	}
}

// AnalyzeRun generates coaching feedback for a single run.
func (c *Client) AnalyzeRun(ctx context.Context, activity domain.Activity, history []domain.Activity, profile domain.AthleteProfile) (domain.RunAnalysis, error) {
	userPrompt := prompts.BuildAnalyzePrompt(activity, history, profile)
	resp, err := c.chatWithRetry(ctx, userPrompt)
	if err != nil {
		return domain.RunAnalysis{}, fmt.Errorf("ollama chat: %w", err)
	}
	return prompts.ParseAnalysis(resp)
}

// GenerateWeeklyPlan creates a weekly summary and upcoming workout plan.
func (c *Client) GenerateWeeklyPlan(ctx context.Context, week domain.WeekSummary, profile domain.AthleteProfile) (ports.WeeklyPlan, error) {
	userPrompt := prompts.BuildWeeklyPrompt(week, profile)
	resp, err := c.chatWithRetry(ctx, userPrompt)
	if err != nil {
		return ports.WeeklyPlan{}, fmt.Errorf("ollama chat: %w", err)
	}
	return prompts.ParseWeeklyPlan(resp)
}

// chatWithRetry attempts the chat once, and if the response looks malformed,
// retries with an explicit instruction to produce valid JSON.
func (c *Client) chatWithRetry(ctx context.Context, prompt string) (string, error) {
	resp, err := c.chat(ctx, prompt)
	if err != nil {
		return "", err
	}
	if looksMalformed(resp) {
		fixPrompt := prompt + "\n\nIMPORTANT: Your previous response was malformed. Respond with VALID JSON ONLY, no markdown, no extra text."
		resp, err = c.chat(ctx, fixPrompt)
		if err != nil {
			return "", err
		}
	}
	return resp, nil
}

// looksMalformed does a lightweight check for common JSON issues.
func looksMalformed(s string) bool {
	s = trimSpaceAndCode(s)
	if s == "" {
		return true
	}
	if s[0] != '{' && s[0] != '[' {
		return true
	}
	// Very long responses are sometimes truncated mid-object.
	if len(s) > 10 && s[len(s)-1] != '}' && s[len(s)-1] != ']' {
		return true
	}
	return false
}

func trimSpaceAndCode(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
	Format   string    `json:"format"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message message `json:"message"`
	Done    bool    `json:"done"`
}

func (c *Client) chat(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: prompts.SystemPrompt},
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Format: "json",
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama %d: %s", resp.StatusCode, string(body))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}
