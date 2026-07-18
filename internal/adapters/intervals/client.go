package intervals

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dylanewe/coach/internal/adapters/intervals/gen"
	"github.com/dylanewe/coach/internal/domain"
)

// Client wraps the generated oapi-codegen client.
type Client struct {
	apiKey    string
	athleteID string
	client    *gen.ClientWithResponses
}

// NewClient creates a new Intervals.icu client.
func NewClient(apiKey, athleteID string) (*Client, error) {
	client, err := gen.NewClientWithResponses("https://intervals.icu", gen.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		auth := base64.StdEncoding.EncodeToString([]byte("API_KEY:" + apiKey))
		req.Header.Set("Authorization", "Basic "+auth)
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("intervals client: %w", err)
	}
	return &Client{
		apiKey:    apiKey,
		athleteID: athleteID,
		client:    client,
	}, nil
}

// CheckConnection verifies the API key and athlete ID by making a minimal request.
func (c *Client) CheckConnection(ctx context.Context) error {
	params := &gen.ListActivitiesParams{
		Oldest: time.Now().AddDate(0, 0, -7).Format(iso8601Local),
		Limit:  ptr[int32](1),
	}
	resp, err := c.client.ListActivitiesWithResponse(ctx, c.athleteID, params)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	if resp.StatusCode() == http.StatusForbidden {
		return fmt.Errorf("403 Forbidden: API key may be invalid, or athlete ID %q is not accessible (try 0 for your own athlete)", c.athleteID)
	}
	if resp.StatusCode() != http.StatusOK {
		body := string(resp.Body)
		if body == "" {
			body = "<empty body>"
		}
		return fmt.Errorf("connect: %s — %s", resp.Status(), body)
	}
	return nil
}

// FetchActivities retrieves activities since the given time.
func (c *Client) FetchActivities(ctx context.Context, athleteID string, since time.Time) ([]domain.Activity, error) {
	oldest := since.Format(iso8601Local)
	params := &gen.ListActivitiesParams{
		Oldest: oldest,
		Limit:  ptr[int32](100),
	}
	resp, err := c.client.ListActivitiesWithResponse(ctx, athleteID, params)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	if resp.StatusCode() == http.StatusForbidden {
		return nil, fmt.Errorf("list activities: 403 Forbidden for athlete %s (verify API key and athlete ID)", athleteID)
	}
	if resp.StatusCode() != http.StatusOK {
		body := string(resp.Body)
		if body == "" {
			body = "<empty body>"
		}
		return nil, fmt.Errorf("list activities: %s — %s", resp.Status(), body)
	}

	var items []gen.Activity
	if err := json.Unmarshal(resp.Body, &items); err != nil {
		return nil, fmt.Errorf("unmarshal activities: %w", err)
	}

	out := make([]domain.Activity, 0, len(items))
	for _, a := range items {
		if a.Type != nil && *a.Type == "Run" {
			out = append(out, mapActivity(a))
		}
	}
	return out, nil
}

// CreateWorkout pushes a planned workout to the athlete's calendar.
func (c *Client) CreateWorkout(ctx context.Context, athleteID string, w domain.Workout) error {
	start := time.Now().AddDate(0, 0, w.Day).Format("2006-01-02")
	name := w.Name
	desc := w.Description
	cat := gen.EventExCategory("WORKOUT")
	typ := w.Type
	movTime := int32(w.MovingTime)
	event := gen.EventEx{
		StartDateLocal: &start,
		Name:           &name,
		Description:    &desc,
		Category:       &cat,
		Type:           &typ,
		MovingTime:     &movTime,
		Distance:       ptr[float32](float32(w.Distance)),
	}
	if w.Target != "" {
		t := gen.EventExTarget(w.Target)
		event.Target = &t
	}
	params := &gen.CreateEventParams{
		UpsertOnUid: false,
	}
	resp, err := c.client.CreateEventWithResponse(ctx, athleteID, params, event)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("create event: %s", resp.Status())
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

// iso8601Local is the date format Intervals.icu expects for the oldest/newest params.
// The API docs specify "Local ISO-8601 date or date and time e.g. 2019-07-22T16:18:49".
const iso8601Local = "2006-01-02T15:04:05"

func mapActivity(a gen.Activity) domain.Activity {
	var start time.Time
	if a.StartDateLocal != nil {
		// Intervals.icu returns start_date_local as "2026-06-14T07:03:32" (no timezone).
		// Parse it as UTC so the local calendar date is preserved for grouping.
		start, _ = time.ParseInLocation(iso8601Local, *a.StartDateLocal, time.UTC)
	}

	source := safeSource(a.Source)
	cadence := safeFloat(a.AverageCadence)
	// COROS reports cadence as steps per foot per minute in some exports;
	// Intervals.icu surfaces this as ~half the true SPM value.
	if source == "COROS" && cadence > 0 {
		cadence *= 2
	}

	return domain.Activity{
		ID:                 safeString(a.Id),
		Name:               safeString(a.Name),
		Type:               safeString(a.Type),
		SubType:            safeSubType(a.SubType),
		StartDateLocal:     start,
		Distance:           safeFloat(a.Distance),
		MovingTime:         int(safeInt32(a.MovingTime)),
		ElapsedTime:        int(safeInt32(a.ElapsedTime)),
		TotalElevationGain: safeFloat(a.TotalElevationGain),
		AverageSpeed:       safeFloat(a.AverageSpeed),
		AverageHeartrate:   safeFloatFromInt32(a.AverageHeartrate),
		MaxHeartrate:       safeFloatFromInt32(a.MaxHeartrate),
		AverageCadence:     cadence,
		ICULoad:            safeFloatFromInt32(a.IcuTrainingLoad),
		ICUCTL:             safeFloat(a.IcuCtl),
		ICUATL:             safeFloat(a.IcuAtl),
		PaceAtHRZ2Ms:       safeFloat(a.IcuPowerHrZ2),
		Description:        safeString(a.Description),
		CreatedAt:          time.Now(),
		SyncedAt:           time.Now(),
	}
}

func safeSubType(s *gen.ActivitySubType) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func safeSource(s *gen.ActivitySource) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeFloat(f *float32) float64 {
	if f == nil {
		return 0
	}
	return float64(*f)
}

func safeFloatFromInt32(i *int32) float64 {
	if i == nil {
		return 0
	}
	return float64(*i)
}

func safeInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}
