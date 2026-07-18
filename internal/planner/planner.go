package planner

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylanewe/coach/internal/analytics"
	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/ports"
)

// defaultSchedule is used when an athlete profile does not define a valid schedule.
func defaultSchedule() []domain.ScheduledDay {
	return []domain.ScheduledDay{
		{DayOfWeek: 0, Type: domain.WorkoutLongRun},
		{DayOfWeek: 3, Type: domain.WorkoutEasy},
		{DayOfWeek: 5, Type: domain.WorkoutTempoInterval},
	}
}

// scheduleForProfile returns the configured weekly schedule, falling back to the
// default schedule if the profile template is missing or invalid.
func scheduleForProfile(profile domain.AthleteProfile) []domain.ScheduledDay {
	if profile.ValidateWeeklySchedule() == nil {
		return profile.WeeklyTemplate
	}
	return defaultSchedule()
}

// ValidateAndNormalize checks an LLM-generated plan for basic correctness and
// clamps distances to safe ranges. It returns the cleaned plan and a list of
// warnings. If the plan is unusable, it returns an error so the caller can fall
// back to the template planner.
func ValidateAndNormalize(plan ports.WeeklyPlan, summary domain.WeekSummary, profile domain.AthleteProfile) (ports.WeeklyPlan, []string, error) {
	phase := analytics.Phase(summary.Phase)
	var warnings []string

	if len(plan.Workouts) == 0 {
		return plan, warnings, fmt.Errorf("no workouts in plan")
	}

	schedule := scheduleForProfile(profile)
	expected := make(map[int]string, len(schedule))
	for _, d := range schedule {
		expected[d.DayOfWeek] = d.Type
	}

	seen := make(map[int]bool)
	var cleaned []domain.Workout
	var totalDist float64

	now := time.Now()

	for _, w := range plan.Workouts {
		// day is an absolute offset from today (as produced by ParseWeeklyPlan and FallbackPlan).
		targetDate := now.AddDate(0, 0, w.Day)
		wd := int(targetDate.Weekday())

		category, ok := expected[wd]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("Ignoring workout on disallowed day %s", targetDate.Weekday()))
			continue
		}
		if seen[wd] {
			warnings = append(warnings, fmt.Sprintf("Duplicate workout on %s; keeping first", targetDate.Weekday()))
			continue
		}
		seen[wd] = true

		slot, ok := slotFor(phase, category)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("No template for %s in %s phase", category, phase))
			cleaned = append(cleaned, w)
			continue
		}

		// Clamp distance to template range.
		if w.Distance < float64(slot.MinDist) {
			warnings = append(warnings, fmt.Sprintf("%s distance %.0f m below template min %d m; clamping", category, w.Distance, slot.MinDist))
			w.Distance = float64(slot.MinDist)
		} else if w.Distance > float64(slot.MaxDist) {
			warnings = append(warnings, fmt.Sprintf("%s distance %.0f m above template max %d m; clamping", category, w.Distance, slot.MaxDist))
			w.Distance = float64(slot.MaxDist)
		}

		// Ensure type is reasonable.
		if strings.TrimSpace(w.Type) == "" {
			w.Type = "Run"
		}
		if strings.TrimSpace(w.Name) == "" {
			w.Name = defaultNameForCategory(category)
		}

		totalDist += w.Distance
		cleaned = append(cleaned, w)
	}

	// Must have every configured day.
	for _, d := range schedule {
		if !seen[d.DayOfWeek] {
			return plan, warnings, fmt.Errorf("missing required workout on %s (%s)", time.Weekday(d.DayOfWeek), d.Type)
		}
	}

	plan.Workouts = cleaned

	// ACWR safety check.
	if summary.Acwr > 1.3 && totalDist > summary.TotalDistance*0.9 {
		warnings = append(warnings, fmt.Sprintf(
			"ACWR %.2f is high; generated plan total %.1f km is close to current week %.1f km. Consider reducing.",
			summary.Acwr, totalDist/1000, summary.TotalDistance/1000,
		))
	}

	return plan, warnings, nil
}

// FallbackPlan generates a conservative default plan from phase templates.
// It applies down-week and ACWR reductions automatically.
func FallbackPlan(summary domain.WeekSummary, profile domain.AthleteProfile) ports.WeeklyPlan {
	phase := analytics.Phase(summary.Phase)
	weekOfPhase := summary.WeekOfPhase

	tmpl, ok := templates[phase]
	if !ok {
		tmpl = templates[analytics.PhaseBase]
	}

	nextMonday := summary.WeekEnd.AddDate(0, 0, 1)
	for nextMonday.Weekday() != time.Monday {
		nextMonday = nextMonday.AddDate(0, 0, 1)
	}

	downWeek := isDownWeek(summary.WeekStart, profile.RaceDate)
	acwrReduction := 1.0
	if summary.Acwr > 1.3 {
		acwrReduction = 0.85
	}

	schedule := scheduleForProfile(profile)
	now := time.Now()
	workouts := make([]domain.Workout, 0, len(schedule))
	for _, day := range schedule {
		slot, ok := tmpl.Slots[day.Type]
		if !ok {
			continue
		}

		// Progress distance through the phase.
		dist := progressiveDistance(slot.MinDist, slot.MaxDist, weekOfPhase, tmpl.Length)

		if downWeek {
			dist = int(float64(dist) * 0.7)
		}
		dist = int(float64(dist) * acwrReduction)

		// Never go below minimum except in race week shakeout.
		if dist < slot.MinDist && phase != analytics.PhaseRace {
			dist = slot.MinDist
		}

		dayOffset := day.DayOfWeek - int(time.Monday)
		if dayOffset < 0 {
			dayOffset += 7
		}
		targetDate := nextMonday.AddDate(0, 0, dayOffset)

		workouts = append(workouts, domain.Workout{
			AthleteID:   profile.ID,
			Name:        defaultNameForCategory(day.Type),
			Description: fmt.Sprintf("%s — target %.1f km", defaultNameForCategory(day.Type), float64(dist)/1000),
			Type:        "Run",
			SubType:     "NONE",
			Day:         DaysBetween(now, targetDate),
			Distance:    float64(dist),
			MovingTime:  EstimateMovingTime(dist),
			Target:      "PACE",
			Tags:        []string{strings.ToLower(string(phase))},
			CreatedAt:   now,
		})
	}

	report := domain.WeeklyReport{
		WeekStart:        nextMonday,
		WeekEnd:          nextMonday.AddDate(0, 0, 6),
		Summary:          fmt.Sprintf("Conservative %s plan generated from templates.", phase),
		Recommendations:  []string{"This is a fallback plan because the LLM plan was invalid or unavailable."},
		NextWeekWorkouts: workouts,
		GeneratedAt:      now,
	}

	return ports.WeeklyPlan{Report: report, Workouts: workouts}
}

// defaultNameForCategory returns a human-readable workout name for a category.
func defaultNameForCategory(category string) string {
	switch category {
	case domain.WorkoutEasy:
		return "Easy Run"
	case domain.WorkoutTempoInterval:
		return "Tempo or Intervals"
	case domain.WorkoutLongRun:
		return "Long Run"
	default:
		return category
	}
}

// progressiveDistance linearly increases distance from min to max over the phase.
func progressiveDistance(min, max, weekOfPhase, phaseLength int) int {
	if phaseLength <= 1 {
		return max
	}
	if weekOfPhase < 1 {
		weekOfPhase = 1
	}
	if weekOfPhase > phaseLength {
		weekOfPhase = phaseLength
	}
	ratio := float64(weekOfPhase-1) / float64(phaseLength-1)
	return min + int(float64(max-min)*ratio)
}

// isDownWeek returns true every 4th week of the macrocycle.
func isDownWeek(weekStart, raceDate time.Time) bool {
	macrocycleStart := raceDate.AddDate(0, 0, -7*analytics.TotalMacrocycleWeeks)
	days := int(weekStart.Sub(macrocycleStart).Hours() / 24)
	weekNum := days/7 + 1
	return weekNum%4 == 0
}

// DaysBetween returns the number of calendar days from a to b (can be negative).
func DaysBetween(a, b time.Time) int {
	aUTC := a.UTC().Truncate(24 * time.Hour)
	bUTC := b.UTC().Truncate(24 * time.Hour)
	return int(bUTC.Sub(aUTC).Hours() / 24)
}

// EstimateMovingTime uses a conservative 6:00/km pace for planning.
func EstimateMovingTime(distM int) int {
	paceSecPerKm := 360 // 6:00/km
	return distM * paceSecPerKm / 1000
}
