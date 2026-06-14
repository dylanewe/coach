package kimi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dylanewe/coach/internal/adapters/llm/prompts"
	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/ports"
)

// Client implements ports.Analyzer for the Kimi / Moonshot API.
type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
}

// NewClient creates a new Kimi analyzer.
func NewClient(apiKey, baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "https://api.moonshot.cn/v1"
	}
	if model == "" {
		model = "moonshot-v1-128k"
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// AnalyzeRun generates coaching feedback for a single run.
func (c *Client) AnalyzeRun(ctx context.Context, activity domain.Activity, history []domain.Activity, profile domain.AthleteProfile) (domain.RunAnalysis, error) {
	prompt := prompts.BuildAnalyzePrompt(activity, history, profile)
	resp, err := c.chat(ctx, prompt)
	if err != nil {
		return domain.RunAnalysis{}, fmt.Errorf("kimi chat: %w", err)
	}
	return prompts.ParseAnalysis(resp)
}

// GenerateWeeklyPlan creates a weekly summary and upcoming workout plan.
func (c *Client) GenerateWeeklyPlan(ctx context.Context, week domain.WeekSummary, profile domain.AthleteProfile) (ports.WeeklyPlan, error) {
	prompt := prompts.BuildWeeklyPrompt(week, profile)
	resp, err := c.chat(ctx, prompt)
	if err != nil {
		return ports.WeeklyPlan{}, fmt.Errorf("kimi chat: %w", err)
	}
	return prompts.ParseWeeklyPlan(resp)
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

func (c *Client) chat(ctx context.Context, userContent string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: prompts.SystemPrompt},
			{Role: "user", Content: userContent},
		},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("kimi %d: %s", resp.StatusCode, string(body))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return out.Choices[0].Message.Content, nil
}
