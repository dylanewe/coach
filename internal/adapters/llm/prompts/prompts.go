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
		activity.StartDateLocal.Weekday(), IntendedWorkoutType(activity.StartDateLocal.Weekday(), profile)))
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
	b.WriteString("\n\nRespond with JSON matching this schema. Use the exact keys shown. Example:\n")
	b.WriteString(`{
  "summary": "Solid aerobic run with good pace control throughout.",
  "positives": ["Kept heart rate in Zone 2 for most of the run", "Consistent cadence", "Felt relaxed"],
  "areas_for_improvement": ["First km was a bit fast", "Could hydrate better"],
  "suggested_next_session": "Easy 45 min recovery run tomorrow",
  "fatigue_score": 3
}`)
	return b.String()
}

// IntendedWorkoutType returns the expected workout category for a given weekday
// based on the athlete's configured weekly schedule.
func IntendedWorkoutType(wd time.Weekday, profile domain.AthleteProfile) string {
	for _, d := range profile.WeeklyTemplate {
		if d.DayOfWeek == int(wd) {
			switch d.Type {
			case domain.WorkoutEasy:
				return "Easy Run"
			case domain.WorkoutTempoInterval:
				return "Tempo, Interval, or Speed Work"
			case domain.WorkoutLongRun:
				return "Long Run"
			}
		}
	}
	return "Rest day (no run planned)"
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
	b.WriteString(scheduleConstraints(profile))
	b.WriteString("- Every 4th week should be a recovery/down week at ~70% volume\n")

	b.WriteString("\nRespond with JSON matching this schema. Use the exact keys shown. Example:\n")
	b.WriteString(buildWeeklyExample(profile))
	b.WriteString("\n\nNote: `day` is the number of days from today. Use the exact day values listed above for each scheduled workout.")
	return b.String()
}

// scheduleConstraints builds the "IMPORTANT CONSTRAINTS" section from the
// athlete's configured weekly schedule, including pre-computed day offsets.
func scheduleConstraints(profile domain.AthleteProfile) string {
	schedule := effectiveSchedule(profile)
	var b strings.Builder
	for _, d := range schedule {
		offset := dayOffsetFromToday(d.DayOfWeek)
		b.WriteString(fmt.Sprintf("- %s (day=%d): %s\n", time.Weekday(d.DayOfWeek), offset, d.Type))
	}

	days := make([]string, 0, 7)
	for wd := 0; wd < 7; wd++ {
		found := false
		for _, d := range schedule {
			if d.DayOfWeek == wd {
				found = true
				break
			}
		}
		if !found {
			days = append(days, time.Weekday(wd).String())
		}
	}
	if len(days) > 0 {
		b.WriteString(fmt.Sprintf("- No running on %s\n", strings.Join(days, ", ")))
	}
	return b.String()
}

// buildWeeklyExample generates a JSON example that matches the athlete's
// configured schedule and uses the correct day offsets from today.
func buildWeeklyExample(profile domain.AthleteProfile) string {
	schedule := effectiveSchedule(profile)

	var workouts []string
	for _, d := range schedule {
		name, dist := exampleForCategory(d.Type)
		workouts = append(workouts, fmt.Sprintf(`    {
      "day": %d,
      "name": %q,
      "description": %q,
      "type": "Run",
      "sub_type": "NONE",
      "moving_time": %d,
      "distance": %d,
      "target": "PACE",
      "tags": [%s]
    }`, dayOffsetFromToday(d.DayOfWeek), name, fmt.Sprintf("%s — target %.1f km", name, float64(dist)/1000), planner.EstimateMovingTime(dist), dist, exampleTags(d.Type)))
	}

	return fmt.Sprintf(`{
  "summary": "Good week with a solid long run and controlled easy pace.",
  "recommendations": ["Keep the long run easy", "Add one more recovery day if legs feel heavy"],
  "next_week_workouts": [
%s
  ]
}`, strings.Join(workouts, ",\n"))
}

func exampleForCategory(category string) (string, int) {
	switch category {
	case domain.WorkoutEasy:
		return "Easy Run", 6000
	case domain.WorkoutTempoInterval:
		return "Tempo Run", 8000
	case domain.WorkoutLongRun:
		return "Long Run", 12000
	default:
		return "Run", 5000
	}
}

func exampleTags(category string) string {
	switch category {
	case domain.WorkoutEasy:
		return `"easy", "base"`
	case domain.WorkoutTempoInterval:
		return `"tempo", "build"`
	case domain.WorkoutLongRun:
		return `"long", "base"`
	default:
		return `"run"`
	}
}

// effectiveSchedule returns the profile's schedule if it is valid, otherwise the
// default Sunday/Wednesday/Friday schedule.
func effectiveSchedule(profile domain.AthleteProfile) []domain.ScheduledDay {
	if profile.ValidateWeeklySchedule() == nil {
		return profile.WeeklyTemplate
	}
	return []domain.ScheduledDay{
		{DayOfWeek: 0, Type: domain.WorkoutLongRun},
		{DayOfWeek: 3, Type: domain.WorkoutEasy},
		{DayOfWeek: 5, Type: domain.WorkoutTempoInterval},
	}
}

// dayOffsetFromToday returns the number of days from today until the next
// occurrence of the given weekday (0=Sunday). If today is that weekday, it
// returns 7 so the plan targets next week.
func dayOffsetFromToday(dayOfWeek int) int {
	diff := dayOfWeek - int(time.Now().Weekday())
	if diff <= 0 {
		diff += 7
	}
	return diff
}

// ParseAnalysis parses the LLM response into a RunAnalysis.
func ParseAnalysis(text string) (domain.RunAnalysis, error) {
	text = stripMarkdown(text)
	var a domain.RunAnalysis
	if err := json.Unmarshal([]byte(text), &a); err != nil {
		return domain.RunAnalysis{}, fmt.Errorf("parse analysis: %w (raw: %s)", err, truncate(text, 500))
	}
	if a.Summary == "" && len(a.Positives) == 0 && a.FatigueScore == 0 {
		return domain.RunAnalysis{}, fmt.Errorf("analysis has empty fields (raw: %s)", truncate(text, 500))
	}
	a.GeneratedAt = time.Now()
	return a, nil
}

// ParseWeeklyPlan parses the LLM response into a WeeklyPlan.
// The LLM provides `day` as the number of days from today; this is stored
// directly in Workout.Day.
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
		// day is already expressed as days from today; keep it as-is.
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
