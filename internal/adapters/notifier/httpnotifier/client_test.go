package httpnotifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dylanewe/coach/internal/domain"
)

func TestSendRunAnalysis(t *testing.T) {
	var receivedReq *http.Request
	var receivedBody notifyRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", "telegram")
	activity := domain.Activity{
		Name:           "Morning Run",
		StartDateLocal: mustParse("2026-06-30"),
		Distance:       5000,
		MovingTime:     1800,
	}
	analysis := domain.RunAnalysis{
		Summary:              "Good run.",
		Positives:            []string{"steady pace"},
		AreasForImprovement:  []string{"cadence"},
		SuggestedNextSession: "Easy 5k",
		FatigueScore:         3,
	}

	err := client.SendRunAnalysis(context.Background(), "to@example.com", analysis, activity)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if receivedReq.URL.Path != "/notify" {
		t.Fatalf("expected path /notify, got %s", receivedReq.URL.Path)
	}
	if auth := receivedReq.Header.Get("Authorization"); auth != "Bearer test-key" {
		t.Fatalf("expected Authorization 'Bearer test-key', got %q", auth)
	}
	if receivedBody.Channel != "telegram" {
		t.Fatalf("expected channel telegram, got %q", receivedBody.Channel)
	}
	if receivedBody.Title != "Run Analysis: Morning Run" {
		t.Fatalf("unexpected title: %q", receivedBody.Title)
	}
}

func mustParse(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}
