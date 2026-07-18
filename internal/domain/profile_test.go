package domain

import (
	"strings"
	"testing"
)

func TestValidateWeeklyScheduleValidThreeDay(t *testing.T) {
	p := AthleteProfile{
		WeeklyTemplate: []ScheduledDay{
			{DayOfWeek: 0, Type: WorkoutLongRun},
			{DayOfWeek: 3, Type: WorkoutEasy},
			{DayOfWeek: 5, Type: WorkoutTempoInterval},
		},
	}
	if err := p.ValidateWeeklySchedule(); err != nil {
		t.Fatalf("expected valid schedule, got %v", err)
	}
}

func TestValidateWeeklyScheduleValidFourDay(t *testing.T) {
	p := AthleteProfile{
		WeeklyTemplate: []ScheduledDay{
			{DayOfWeek: 0, Type: WorkoutLongRun},
			{DayOfWeek: 2, Type: WorkoutEasy},
			{DayOfWeek: 4, Type: WorkoutTempoInterval},
			{DayOfWeek: 5, Type: WorkoutEasy},
		},
	}
	if err := p.ValidateWeeklySchedule(); err != nil {
		t.Fatalf("expected valid schedule, got %v", err)
	}
}

func TestValidateWeeklyScheduleTooManyEasy(t *testing.T) {
	p := AthleteProfile{
		WeeklyTemplate: []ScheduledDay{
			{DayOfWeek: 0, Type: WorkoutLongRun},
			{DayOfWeek: 2, Type: WorkoutEasy},
			{DayOfWeek: 3, Type: WorkoutEasy},
			{DayOfWeek: 4, Type: WorkoutTempoInterval},
			{DayOfWeek: 5, Type: WorkoutEasy},
		},
	}
	if err := p.ValidateWeeklySchedule(); err == nil {
		t.Fatal("expected error for too many easy days")
	} else if !strings.Contains(err.Error(), "1 or 2") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateWeeklyScheduleMissingTempo(t *testing.T) {
	p := AthleteProfile{
		WeeklyTemplate: []ScheduledDay{
			{DayOfWeek: 0, Type: WorkoutLongRun},
			{DayOfWeek: 3, Type: WorkoutEasy},
			{DayOfWeek: 5, Type: WorkoutEasy},
		},
	}
	if err := p.ValidateWeeklySchedule(); err == nil {
		t.Fatal("expected error for missing tempo day")
	}
}

func TestValidateWeeklyScheduleDuplicateDay(t *testing.T) {
	p := AthleteProfile{
		WeeklyTemplate: []ScheduledDay{
			{DayOfWeek: 0, Type: WorkoutLongRun},
			{DayOfWeek: 0, Type: WorkoutEasy},
			{DayOfWeek: 5, Type: WorkoutTempoInterval},
		},
	}
	if err := p.ValidateWeeklySchedule(); err == nil {
		t.Fatal("expected error for duplicate weekday")
	}
}

func TestValidateWeeklyScheduleUnknownType(t *testing.T) {
	p := AthleteProfile{
		WeeklyTemplate: []ScheduledDay{
			{DayOfWeek: 0, Type: WorkoutLongRun},
			{DayOfWeek: 3, Type: WorkoutEasy},
			{DayOfWeek: 5, Type: "Speed"},
		},
	}
	if err := p.ValidateWeeklySchedule(); err == nil {
		t.Fatal("expected error for unknown workout type")
	}
}

func TestScheduleByWeekday(t *testing.T) {
	p := AthleteProfile{
		WeeklyTemplate: []ScheduledDay{
			{DayOfWeek: 0, Type: WorkoutLongRun},
			{DayOfWeek: 3, Type: WorkoutEasy},
			{DayOfWeek: 5, Type: WorkoutTempoInterval},
		},
	}
	m := p.ScheduleByWeekday()
	if m[0] != WorkoutLongRun || m[3] != WorkoutEasy || m[5] != WorkoutTempoInterval {
		t.Fatalf("unexpected schedule map: %v", m)
	}
}
