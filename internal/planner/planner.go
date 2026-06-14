package planner

import (
	"fmt"
	"strings"
	"time"

	"github.com/dylanewe/coach/internal/analytics"
	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/ports"
)

// Allowed days of week for workouts: Sunday(0), Wednesday(3), Friday(5).
var allowedDays = map[int]bool{0: true, 3: true, 5: true}

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

	seen := make(map[int]bool)
	var cleaned []domain.Workout
	var totalDist float64

	now := time.Now()

	for _, w := range plan.Workouts {
		// day is an absolute offset from today (as produced by parseWeeklyPlan and FallbackPlan).
		targetDate := now.AddDate(0, 0, w.Day)
		wd := int(targetDate.Weekday())

		if !allowedDays[wd] {
			warnings = append(warnings, fmt.Sprintf("Ignoring workout on disallowed day %s", targetDate.Weekday()))
			continue
		}
		if seen[wd] {
			warnings = append(warnings, fmt.Sprintf("Duplicate workout on %s; keeping first", targetDate.Weekday()))
			continue
		}
		seen[wd] = true

		slot, ok := slotFor(phase, wd)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("No template for %s in %s phase", targetDate.Weekday(), phase))
			cleaned = append(cleaned, w)
			continue
		}

		// Clamp distance to template range.
		if w.Distance < float64(slot.MinDist) {
			warnings = append(warnings, fmt.Sprintf("%s distance %.0f m below template min %d m; clamping", slot.Type, w.Distance, slot.MinDist))
			w.Distance = float64(slot.MinDist)
		} else if w.Distance > float64(slot.MaxDist) {
			warnings = append(warnings, fmt.Sprintf("%s distance %.0f m above template max %d m; clamping", slot.Type, w.Distance, slot.MaxDist))
			w.Distance = float64(slot.MaxDist)
		}

		// Ensure type is reasonable.
		if strings.TrimSpace(w.Type) == "" {
			w.Type = "Run"
		}
		if strings.TrimSpace(w.Name) == "" {
			w.Name = slot.Type
		}

		totalDist += w.Distance
		cleaned = append(cleaned, w)
	}

	// Must have all three required days.
	for d, name := range map[int]string{0: "Sunday Long Run", 3: "Wednesday Easy", 5: "Friday Tempo/Interval"} {
		if !seen[d] {
			return plan, warnings, fmt.Errorf("missing required workout: %s", name)
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

	now := time.Now()
	workouts := make([]domain.Workout, 0, len(tmpl.Slots))
	for _, slot := range tmpl.Slots {
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

		targetDate := nextMonday.AddDate(0, 0, slot.DayOfWeek-int(time.Monday))
		// Adjust because Monday=1 in Go; our offset from nextMonday should be slot.DayOfWeek-1.
		dayOffset := slot.DayOfWeek - int(time.Monday)
		if dayOffset < 0 {
			dayOffset += 7
		}
		targetDate = nextMonday.AddDate(0, 0, dayOffset)

		workouts = append(workouts, domain.Workout{
			AthleteID:   profile.ID,
			Name:        slot.Type,
			Description: fmt.Sprintf("%s — target %.1f km", slot.Type, float64(dist)/1000),
			Type:        "Run",
			SubType:     "NONE",
			Day:         DaysBetween(now, targetDate),
			Distance:    float64(dist),
			MovingTime:  estimateMovingTime(dist),
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

// estimateMovingTime uses a conservative 6:00/km pace for planning.
func estimateMovingTime(distM int) int {
	paceSecPerKm := 360 // 6:00/km
	return distM * paceSecPerKm / 1000
}
