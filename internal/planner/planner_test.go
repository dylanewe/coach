package planner

import (
	"testing"
	"time"

	"github.com/dylanewe/coach/internal/analytics"
	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/ports"
)

// dayOffsetForWeekday returns the number of days from today until the next occurrence
// of the given weekday (0=Sunday). If today is that weekday, it returns 7 (next week).
func dayOffsetForWeekday(wd time.Weekday) int {
	diff := int(wd) - int(time.Now().Weekday())
	if diff <= 0 {
		diff += 7
	}
	return diff
}

func TestFallbackPlan(t *testing.T) {
	race := time.Date(2026, 12, 12, 0, 0, 0, 0, time.UTC)
	summary := domain.WeekSummary{
		WeekStart:   time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:     time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC),
		Phase:       string(analytics.PhaseBase),
		WeekOfPhase: 2,
		Acwr:        0.9,
	}
	profile := domain.AthleteProfile{RaceDate: race}

	plan := FallbackPlan(summary, profile)

	if len(plan.Workouts) != 3 {
		t.Fatalf("expected 3 workouts, got %d", len(plan.Workouts))
	}

	days := make(map[int]bool)
	for _, w := range plan.Workouts {
		targetDate := time.Now().AddDate(0, 0, w.Day)
		wd := int(targetDate.Weekday())
		if days[wd] {
			t.Errorf("duplicate workout on weekday %d", wd)
		}
		days[wd] = true
	}
	if !days[0] || !days[3] || !days[5] {
		t.Errorf("expected workouts on Sunday(0), Wednesday(3), Friday(5); got %v", days)
	}
}

func TestValidateAndNormalizeRejectsMissingDay(t *testing.T) {
	race := time.Date(2026, 12, 12, 0, 0, 0, 0, time.UTC)
	summary := domain.WeekSummary{
		WeekStart:   time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:     time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC),
		Phase:       string(analytics.PhaseBase),
		WeekOfPhase: 2,
		Acwr:        0.9,
	}
	profile := domain.AthleteProfile{RaceDate: race}

	// Only Sunday and Wednesday — missing Friday.
	plan := ports.WeeklyPlan{
		Workouts: []domain.Workout{
			{Day: dayOffsetForWeekday(time.Sunday), Name: "Long Run", Distance: 10000},
			{Day: dayOffsetForWeekday(time.Wednesday), Name: "Easy", Distance: 5000},
		},
	}

	_, _, err := ValidateAndNormalize(plan, summary, profile)
	if err == nil {
		t.Error("expected error for missing Friday workout")
	}
}

func TestValidateAndNormalizeClampsDistance(t *testing.T) {
	race := time.Date(2026, 12, 12, 0, 0, 0, 0, time.UTC)
	summary := domain.WeekSummary{
		WeekStart:   time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:     time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC),
		Phase:       string(analytics.PhaseBase),
		WeekOfPhase: 2,
		Acwr:        0.9,
	}
	profile := domain.AthleteProfile{RaceDate: race}

	// Sunday long run of 30 km should be clamped to Base max 12 km.
	plan := ports.WeeklyPlan{
		Workouts: []domain.Workout{
			{Day: dayOffsetForWeekday(time.Sunday), Name: "Long Run", Distance: 30000},
			{Day: dayOffsetForWeekday(time.Wednesday), Name: "Easy", Distance: 6000},
			{Day: dayOffsetForWeekday(time.Friday), Name: "Strides", Distance: 5000},
		},
	}

	cleaned, _, err := ValidateAndNormalize(plan, summary, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, w := range cleaned.Workouts {
		if w.Name == "Long Run" && w.Distance != 12000 {
			t.Errorf("expected long run clamped to 12000, got %.0f", w.Distance)
		}
	}
}
