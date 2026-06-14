package planner

import "github.com/dylanewe/coach/internal/analytics"

// workoutSlot describes one of the three fixed weekly sessions.
type workoutSlot struct {
	DayOfWeek int
	Type      string
	MinDist   int // meters
	MaxDist   int // meters
}

// phaseTemplate defines the 3-day structure for a periodization phase.
type phaseTemplate struct {
	Phase  analytics.Phase
	Slots  []workoutSlot
	Length int // weeks
}

var templates = map[analytics.Phase]phaseTemplate{
	analytics.PhaseBase: {
		Phase:  analytics.PhaseBase,
		Length: 8,
		Slots: []workoutSlot{
			{DayOfWeek: 3, Type: "Easy Run", MinDist: 5000, MaxDist: 7000},
			{DayOfWeek: 5, Type: "Easy Run with Strides", MinDist: 4000, MaxDist: 6000},
			{DayOfWeek: 0, Type: "Long Run", MinDist: 8000, MaxDist: 12000},
		},
	},
	analytics.PhaseBuild: {
		Phase:  analytics.PhaseBuild,
		Length: 8,
		Slots: []workoutSlot{
			{DayOfWeek: 3, Type: "Easy Run", MinDist: 5000, MaxDist: 7000},
			{DayOfWeek: 5, Type: "Tempo or Intervals", MinDist: 6000, MaxDist: 10000},
			{DayOfWeek: 0, Type: "Long Run", MinDist: 12000, MaxDist: 16000},
		},
	},
	analytics.PhasePeak: {
		Phase:  analytics.PhasePeak,
		Length: 6,
		Slots: []workoutSlot{
			{DayOfWeek: 3, Type: "Easy Run", MinDist: 5000, MaxDist: 6000},
			{DayOfWeek: 5, Type: "HM Pace Workout", MinDist: 8000, MaxDist: 12000},
			{DayOfWeek: 0, Type: "Long Run with HM Segments", MinDist: 14000, MaxDist: 18000},
		},
	},
	analytics.PhaseTaper: {
		Phase:  analytics.PhaseTaper,
		Length: 2,
		Slots: []workoutSlot{
			{DayOfWeek: 3, Type: "Easy Run", MinDist: 4000, MaxDist: 5000},
			{DayOfWeek: 5, Type: "Short Tempo or Strides", MinDist: 5000, MaxDist: 6000},
			{DayOfWeek: 0, Type: "Moderate Long Run", MinDist: 10000, MaxDist: 12000},
		},
	},
	analytics.PhaseRace: {
		Phase:  analytics.PhaseRace,
		Length: 2,
		Slots: []workoutSlot{
			{DayOfWeek: 3, Type: "Easy Run with Strides", MinDist: 3000, MaxDist: 4000},
			{DayOfWeek: 5, Type: "Shakeout or Rest", MinDist: 0, MaxDist: 2000},
			{DayOfWeek: 0, Type: "Race Day", MinDist: 21097, MaxDist: 21097},
		},
	},
}

// slotFor returns the template slot for a given day of week in a phase.
func slotFor(phase analytics.Phase, dayOfWeek int) (workoutSlot, bool) {
	tmpl, ok := templates[phase]
	if !ok {
		return workoutSlot{}, false
	}
	for _, s := range tmpl.Slots {
		if s.DayOfWeek == dayOfWeek {
			return s, true
		}
	}
	return workoutSlot{}, false
}
