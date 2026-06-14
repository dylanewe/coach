package analytics

import "time"

// Phase represents a periodization phase.
type Phase string

const (
	PhaseBase  Phase = "Base"
	PhaseBuild Phase = "Build"
	PhasePeak  Phase = "Peak"
	PhaseTaper Phase = "Taper"
	PhaseRace  Phase = "Race"
)

// TotalMacrocycleWeeks is the length of the half-marathon plan.
const TotalMacrocycleWeeks = 26

// PhaseInfo describes where the athlete is in the macrocycle.
type PhaseInfo struct {
	Phase       Phase
	WeekOfPhase int
	WeeksToRace int
}

// PhaseForDate returns the training phase for a given date relative to the race date.
func PhaseForDate(raceDate, today time.Time) PhaseInfo {
	// Use UTC date boundaries for consistent week math.
	race := raceDate.UTC().Truncate(24 * time.Hour)
	now := today.UTC().Truncate(24 * time.Hour)

	// Macrocycle starts 26 weeks before the race.
	start := race.AddDate(0, 0, -7*TotalMacrocycleWeeks)

	daysSinceStart := int(now.Sub(start).Hours() / 24)
	weekOfMacrocycle := daysSinceStart/7 + 1
	if weekOfMacrocycle < 1 {
		weekOfMacrocycle = 1
	}
	if weekOfMacrocycle > TotalMacrocycleWeeks {
		weekOfMacrocycle = TotalMacrocycleWeeks
	}

	weeksToRace := TotalMacrocycleWeeks - weekOfMacrocycle
	if weeksToRace < 0 {
		weeksToRace = 0
	}

	var phase Phase
	var weekOfPhase int
	switch {
	case weekOfMacrocycle <= 8:
		phase = PhaseBase
		weekOfPhase = weekOfMacrocycle
	case weekOfMacrocycle <= 16:
		phase = PhaseBuild
		weekOfPhase = weekOfMacrocycle - 8
	case weekOfMacrocycle <= 22:
		phase = PhasePeak
		weekOfPhase = weekOfMacrocycle - 16
	case weekOfMacrocycle <= 24:
		phase = PhaseTaper
		weekOfPhase = weekOfMacrocycle - 22
	default:
		phase = PhaseRace
		weekOfPhase = weekOfMacrocycle - 24
	}

	return PhaseInfo{
		Phase:       phase,
		WeekOfPhase: weekOfPhase,
		WeeksToRace: weeksToRace,
	}
}
