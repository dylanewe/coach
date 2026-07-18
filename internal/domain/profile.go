package domain

import (
	"fmt"
	"time"
)

// Workout schedule categories.
const (
	WorkoutEasy          = "Easy"
	WorkoutTempoInterval = "Tempo/Interval"
	WorkoutLongRun       = "Long Run"
)

// validWorkoutCategories lists the allowed values for ScheduledDay.Type.
var validWorkoutCategories = map[string]bool{
	WorkoutEasy:          true,
	WorkoutTempoInterval: true,
	WorkoutLongRun:       true,
}

// AthleteProfile holds static/semi-static athlete configuration.
// It is stored as a singleton document with _id = "athlete".
type AthleteProfile struct {
	ID              string         `bson:"_id" json:"id"`
	RaceDate        time.Time      `bson:"race_date" json:"race_date"`
	MaxHR           int            `bson:"max_hr" json:"max_hr"`
	ThresholdHR     int            `bson:"threshold_hr" json:"threshold_hr"`
	ThresholdPaceMs float64        `bson:"threshold_pace_ms" json:"threshold_pace_ms"`
	RestingHR       int            `bson:"resting_hr" json:"resting_hr"`
	WeeklyTemplate  []ScheduledDay `bson:"weekly_template" json:"weekly_template"`
	CreatedAt       time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `bson:"updated_at" json:"updated_at"`
}

// ScheduledDay maps a weekday to its workout category.
// DayOfWeek follows Go's time.Weekday: 0=Sunday, 1=Monday, ... 6=Saturday.
// Type must be one of the Workout* constants.
type ScheduledDay struct {
	DayOfWeek int    `bson:"day_of_week" json:"day_of_week"`
	Type      string `bson:"type" json:"type"`
}

// ValidateWeeklySchedule checks that WeeklyTemplate defines a supported schedule:
//   - 1 or 2 easy runs
//   - exactly 1 tempo/interval run
//   - exactly 1 long run
//   - no duplicate weekdays and no unknown categories.
func (p AthleteProfile) ValidateWeeklySchedule() error {
	if len(p.WeeklyTemplate) == 0 {
		return fmt.Errorf("weekly_template is empty")
	}

	seen := make(map[int]bool)
	counts := map[string]int{
		WorkoutEasy:          0,
		WorkoutTempoInterval: 0,
		WorkoutLongRun:       0,
	}

	for _, d := range p.WeeklyTemplate {
		if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
			return fmt.Errorf("invalid day_of_week %d (must be 0-6)", d.DayOfWeek)
		}
		if seen[d.DayOfWeek] {
			return fmt.Errorf("duplicate day_of_week %d", d.DayOfWeek)
		}
		seen[d.DayOfWeek] = true

		if !validWorkoutCategories[d.Type] {
			return fmt.Errorf("invalid workout type %q (must be one of %s, %s, %s)",
				d.Type, WorkoutEasy, WorkoutTempoInterval, WorkoutLongRun)
		}
		counts[d.Type]++
	}

	if counts[WorkoutEasy] < 1 || counts[WorkoutEasy] > 2 {
		return fmt.Errorf("schedule must have 1 or 2 %s days, got %d", WorkoutEasy, counts[WorkoutEasy])
	}
	if counts[WorkoutTempoInterval] != 1 {
		return fmt.Errorf("schedule must have exactly 1 %s day, got %d", WorkoutTempoInterval, counts[WorkoutTempoInterval])
	}
	if counts[WorkoutLongRun] != 1 {
		return fmt.Errorf("schedule must have exactly 1 %s day, got %d", WorkoutLongRun, counts[WorkoutLongRun])
	}

	return nil
}

// ScheduleByWeekday returns a map of weekday -> workout category.
func (p AthleteProfile) ScheduleByWeekday() map[int]string {
	out := make(map[int]string, len(p.WeeklyTemplate))
	for _, d := range p.WeeklyTemplate {
		out[d.DayOfWeek] = d.Type
	}
	return out
}
