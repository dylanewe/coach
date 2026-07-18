package planner

import (
	"github.com/dylanewe/coach/internal/analytics"
	"github.com/dylanewe/coach/internal/domain"
)

// workoutSlot describes distance ranges for one workout category.
type workoutSlot struct {
	Category string
	MinDist  int // meters
	MaxDist  int // meters
}

// phaseTemplate defines distance ranges for each workout category in a periodization phase.
type phaseTemplate struct {
	Phase  analytics.Phase
	Slots  map[string]workoutSlot
	Length int // weeks
}

var templates = map[analytics.Phase]phaseTemplate{
	analytics.PhaseBase: {
		Phase:  analytics.PhaseBase,
		Length: 8,
		Slots: map[string]workoutSlot{
			domain.WorkoutEasy:          {Category: domain.WorkoutEasy, MinDist: 5000, MaxDist: 7000},
			domain.WorkoutTempoInterval: {Category: domain.WorkoutTempoInterval, MinDist: 4000, MaxDist: 6000},
			domain.WorkoutLongRun:       {Category: domain.WorkoutLongRun, MinDist: 8000, MaxDist: 12000},
		},
	},
	analytics.PhaseBuild: {
		Phase:  analytics.PhaseBuild,
		Length: 8,
		Slots: map[string]workoutSlot{
			domain.WorkoutEasy:          {Category: domain.WorkoutEasy, MinDist: 5000, MaxDist: 7000},
			domain.WorkoutTempoInterval: {Category: domain.WorkoutTempoInterval, MinDist: 6000, MaxDist: 10000},
			domain.WorkoutLongRun:       {Category: domain.WorkoutLongRun, MinDist: 12000, MaxDist: 16000},
		},
	},
	analytics.PhasePeak: {
		Phase:  analytics.PhasePeak,
		Length: 6,
		Slots: map[string]workoutSlot{
			domain.WorkoutEasy:          {Category: domain.WorkoutEasy, MinDist: 5000, MaxDist: 6000},
			domain.WorkoutTempoInterval: {Category: domain.WorkoutTempoInterval, MinDist: 8000, MaxDist: 12000},
			domain.WorkoutLongRun:       {Category: domain.WorkoutLongRun, MinDist: 14000, MaxDist: 18000},
		},
	},
	analytics.PhaseTaper: {
		Phase:  analytics.PhaseTaper,
		Length: 2,
		Slots: map[string]workoutSlot{
			domain.WorkoutEasy:          {Category: domain.WorkoutEasy, MinDist: 4000, MaxDist: 5000},
			domain.WorkoutTempoInterval: {Category: domain.WorkoutTempoInterval, MinDist: 5000, MaxDist: 6000},
			domain.WorkoutLongRun:       {Category: domain.WorkoutLongRun, MinDist: 10000, MaxDist: 12000},
		},
	},
	analytics.PhaseRace: {
		Phase:  analytics.PhaseRace,
		Length: 2,
		Slots: map[string]workoutSlot{
			domain.WorkoutEasy:          {Category: domain.WorkoutEasy, MinDist: 3000, MaxDist: 4000},
			domain.WorkoutTempoInterval: {Category: domain.WorkoutTempoInterval, MinDist: 0, MaxDist: 2000},
			domain.WorkoutLongRun:       {Category: domain.WorkoutLongRun, MinDist: 21097, MaxDist: 21097},
		},
	},
}

// slotFor returns the template slot for a given workout category in a phase.
func slotFor(phase analytics.Phase, category string) (workoutSlot, bool) {
	tmpl, ok := templates[phase]
	if !ok {
		return workoutSlot{}, false
	}
	slot, ok := tmpl.Slots[category]
	return slot, ok
}
