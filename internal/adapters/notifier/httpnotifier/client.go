package httpnotifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dylanewe/coach/internal/domain"
)

// Client sends notifications to a generic HTTP notification service.
type Client struct {
	baseURL string
	apiKey  string
	channel string
	client  *http.Client
}

// NewClient creates a notifier that POSTs to the given notification service.
func NewClient(baseURL, apiKey, channel string) *Client {
	if channel == "" {
		channel = "email"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		channel: channel,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendRunAnalysis delivers a run analysis notification.
func (c *Client) SendRunAnalysis(ctx context.Context, to string, analysis domain.RunAnalysis, activity domain.Activity) error {
	title := fmt.Sprintf("Run Analysis: %s", activity.Name)
	body := formatRunAnalysis(activity, analysis)
	return c.send(ctx, title, body)
}

// SendWeeklyReport delivers a weekly report notification.
func (c *Client) SendWeeklyReport(ctx context.Context, to string, report domain.WeeklyReport) error {
	title := fmt.Sprintf("Weekly Running Report: %s", report.WeekStart.Format("2006-01-02"))
	body := formatWeeklyReport(report)
	return c.send(ctx, title, body)
}

type notifyRequest struct {
	Channel  string `json:"channel"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Priority string `json:"priority,omitempty"`
}

func (c *Client) send(ctx context.Context, title, body string) error {
	payload := notifyRequest{
		Channel:  c.channel,
		Title:    title,
		Body:     body,
		Priority: "normal",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	url := c.baseURL + "/notify"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("notification request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned %d", resp.StatusCode)
	}
	return nil
}

func formatRunAnalysis(activity domain.Activity, analysis domain.RunAnalysis) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", activity.Name)
	fmt.Fprintf(&b, "Date: %s\n", activity.StartDateLocal.Format("2006-01-02"))
	fmt.Fprintf(&b, "Distance: %.2f km\n", activity.Distance/1000)
	fmt.Fprintf(&b, "Time: %d:%02d\n\n", activity.MovingTime/60, activity.MovingTime%60)

	fmt.Fprintf(&b, "Summary\n%s\n\n", analysis.Summary)

	if len(analysis.Positives) > 0 {
		fmt.Fprintln(&b, "Positives")
		for _, p := range analysis.Positives {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		fmt.Fprintln(&b)
	}

	if len(analysis.AreasForImprovement) > 0 {
		fmt.Fprintln(&b, "Areas for Improvement")
		for _, a := range analysis.AreasForImprovement {
			fmt.Fprintf(&b, "- %s\n", a)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "Suggested Next Session\n%s\n\n", analysis.SuggestedNextSession)
	fmt.Fprintf(&b, "Fatigue Score: %d/10", analysis.FatigueScore)
	return b.String()
}

func formatWeeklyReport(report domain.WeeklyReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Week: %s to %s\n\n", report.WeekStart.Format("2006-01-02"), report.WeekEnd.Format("2006-01-02"))
	fmt.Fprintf(&b, "Summary\n%s\n\n", report.Summary)

	if len(report.Recommendations) > 0 {
		fmt.Fprintln(&b, "Recommendations")
		for _, r := range report.Recommendations {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		fmt.Fprintln(&b)
	}

	if len(report.NextWeekWorkouts) > 0 {
		fmt.Fprintln(&b, "Next Week's Plan")
		for _, w := range report.NextWeekWorkouts {
			fmt.Fprintf(&b, "- %s: %s (%s)\n", w.Name, w.Description, w.Target)
		}
	}

	return b.String()
}
