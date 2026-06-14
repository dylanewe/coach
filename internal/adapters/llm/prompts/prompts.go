package prompts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dylanewe/coach/internal/analytics"
	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/planner"
	"github.com/dylanewe/coach/internal/ports"
)

const SystemPrompt = `You are an experienced running coach. Analyze the athlete's training data and provide concise, actionable feedback. You must respond with VALID JSON ONLY. Do not include markdown code blocks, explanations, or any text outside the JSON object. Ensure all strings are properly quoted and there are no trailing commas.`

// BuildAnalyzePrompt creates the prompt for daily run analysis.
func BuildAnalyzePrompt(activity domain.Activity, history []domain.Activity, profile domain.AthleteProfile) string {
	var b strings.Builder
	phase := analytics.PhaseForDate(profile.RaceDate, activity.StartDateLocal)
	b.WriteString("Analyze the following run and provide feedback.\n\n")
	b.WriteString(fmt.Sprintf("Race date: %s (%d weeks away). Current phase: %s, week %d.\n",
		profile.RaceDate.Format("2006-01-02"), phase.WeeksToRace, phase.Phase, phase.WeekOfPhase))
	b.WriteString(fmt.Sprintf("Intended workout type for %s: %s\n",
		activity.StartDateLocal.Weekday(), IntendedWorkoutType(activity.StartDateLocal.Weekday())))
	b.WriteString("\nRecent activity history (last 30 days):\n")
	for _, h := range history {
		b.WriteString(fmt.Sprintf("- %s: %s, %.2f km, %s, %.0f bpm avg HR, load %.0f\n",
			h.StartDateLocal.Format("2006-01-02"),
			h.Name,
			h.Distance/1000,
			FormatDuration(time.Duration(h.MovingTime)*time.Second),
			h.AverageHeartrate,
			h.ICULoad,
		))
	}
	b.WriteString("\nRun to analyze:\n")
	data, _ := json.Marshal(activity)
	b.Write(data)
	b.WriteString("\n\nRespond with JSON matching this schema:\n")
	b.WriteString(`{
  "summary": "string",
  "positives": ["string"],
  "areas_for_improvement": ["string"],
  "suggested_next_session": "string",
  "fatigue_score": 1
}`)
	return b.String()
}

// IntendedWorkoutType returns the expected workout type for a given weekday.
func IntendedWorkoutType(wd time.Weekday) string {
	switch wd {
	case time.Sunday:
		return "Long Run"
	case time.Wednesday:
		return "Easy Run"
	case time.Friday:
		return "Tempo, Interval, or Speed Work"
	default:
		return "Rest day (no run planned)"
	}
}

// BuildWeeklyPrompt creates the prompt for weekly plan generation.
func BuildWeeklyPrompt(week domain.WeekSummary, profile domain.AthleteProfile) string {
	var b strings.Builder
	b.WriteString("Generate a weekly training summary and plan for the upcoming week.\n\n")
	b.WriteString(fmt.Sprintf("Race date: %s (%d weeks away)\n", profile.RaceDate.Format("2006-01-02"), week.WeeksToRace))
	b.WriteString(fmt.Sprintf("Current phase: %s, week %d of phase\n", week.Phase, week.WeekOfPhase))
	b.WriteString(fmt.Sprintf("Training load this week: %.0f TSS-equivalent (ACWR %.2f, CTL %.1f, ATL %.1f, TSB %.1f)\n",
		week.TotalLoad, week.Acwr, week.Ctl, week.Atl, week.Tsb))
	if week.Acwr > 1.3 {
		b.WriteString("\nLOAD GUARDRAIL: ACWR is above 1.3. Reduce next week's total volume by at least 15% and downgrade intensity if needed.\n")
	} else if week.Acwr < 0.8 {
		b.WriteString("\nLOAD GUARDRAIL: ACWR is below 0.8. It is safe to build volume gradually.\n")
	}
	b.WriteString("\nThis week's completed runs:\n")
	for _, a := range week.Activities {
		b.WriteString(fmt.Sprintf("- %s: %s, %.2f km, %s, %.0f bpm avg HR, load %.0f\n",
			a.StartDateLocal.Format("2006-01-02"),
			a.Name,
			a.Distance/1000,
			FormatDuration(time.Duration(a.MovingTime)*time.Second),
			a.AverageHeartrate,
			a.ICULoad,
		))
	}
	b.WriteString(fmt.Sprintf("\nWeekly totals: %.2f km, %s moving time\n", week.TotalDistance/1000, FormatDuration(time.Duration(week.TotalTime)*time.Second)))
	b.WriteString("\nIMPORTANT CONSTRAINTS:\n")
	b.WriteString("- Sunday: Long Run\n")
	b.WriteString("- Wednesday: Easy Run\n")
	b.WriteString("- Friday: Tempo, Interval, or Speed Work\n")
	b.WriteString("- No running on other days (Mon, Tue, Thu, Sat)\n")
	b.WriteString("- Every 4th week should be a recovery/down week at ~70% volume\n")
	b.WriteString("\nRespond with JSON matching this schema:\n")
	b.WriteString(`{
  "summary": "string",
  "recommendations": ["string"],
  "next_week_workouts": [
    {
      "day": 0,
      "name": "string",
      "description": "string",
      "type": "Run",
      "sub_type": "NONE",
      "moving_time": 3600,
      "distance": 10000,
      "target": "PACE",
      "tags": ["string"]
    }
  ]
}`)
	b.WriteString("\n\nNote: `day` is offset from next Monday (0=Mon, 6=Sun). Only include Sunday(6), Wednesday(3), Friday(5).")
	return b.String()
}

// ParseAnalysis parses the LLM response into a RunAnalysis.
func ParseAnalysis(text string) (domain.RunAnalysis, error) {
	text = stripMarkdown(text)
	var a domain.RunAnalysis
	if err := json.Unmarshal([]byte(text), &a); err != nil {
		return domain.RunAnalysis{}, fmt.Errorf("parse analysis: %w (raw: %s)", err, truncate(text, 500))
	}
	a.GeneratedAt = time.Now()
	return a, nil
}

// ParseWeeklyPlan parses the LLM response into a WeeklyPlan.
func ParseWeeklyPlan(text string) (ports.WeeklyPlan, error) {
	text = stripMarkdown(text)
	var wrapper struct {
		Summary          string           `json:"summary"`
		Recommendations  []string         `json:"recommendations"`
		NextWeekWorkouts []domain.Workout `json:"next_week_workouts"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err != nil {
		return ports.WeeklyPlan{}, fmt.Errorf("parse weekly plan: %w (raw: %s)", err, truncate(text, 500))
	}

	now := time.Now()
	// Align to next Monday (if today is Monday, use next Monday, not today)
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	nextMonday := now.AddDate(0, 0, daysUntilMonday)

	for i := range wrapper.NextWeekWorkouts {
		w := &wrapper.NextWeekWorkouts[i]
		w.CreatedAt = time.Now()
		// day offset from next Monday -> absolute days from today
		targetDate := nextMonday.AddDate(0, 0, w.Day)
		w.Day = planner.DaysBetween(now, targetDate)
	}

	report := domain.WeeklyReport{
		WeekStart:        nextMonday,
		WeekEnd:          nextMonday.AddDate(0, 0, 6),
		Summary:          wrapper.Summary,
		Recommendations:  wrapper.Recommendations,
		NextWeekWorkouts: wrapper.NextWeekWorkouts,
		GeneratedAt:      time.Now(),
	}

	return ports.WeeklyPlan{
		Report:   report,
		Workouts: wrapper.NextWeekWorkouts,
	}, nil
}

func stripMarkdown(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// FormatDuration formats a duration as H:MM:SS or M:SS.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
