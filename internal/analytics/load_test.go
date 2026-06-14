package analytics

import (
	"testing"
	"time"

	"github.com/dylanewe/coach/internal/domain"
)

func TestComputeLoadMetrics(t *testing.T) {
	base := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	// 29 days so the last 4 complete Monday-Sunday weeks are fully populated.
	var activities []domain.Activity
	for i := 0; i < 29; i++ {
		activities = append(activities, domain.Activity{
			StartDateLocal: base.AddDate(0, 0, -i),
			ICULoad:        30,
		})
	}

	m := ComputeLoadMetrics(activities, base)

	if m.AcuteLoad != 210 {
		t.Errorf("acute load = %v, want 210", m.AcuteLoad)
	}
	if m.ChronicLoad != 210 {
		t.Errorf("chronic load = %v, want 210", m.ChronicLoad)
	}
	if m.Acwr != 1.0 {
		t.Errorf("ACWR = %v, want 1.0", m.Acwr)
	}
	if m.Atl <= 0 {
		t.Errorf("ATL should be > 0, got %v", m.Atl)
	}
	if m.Ctl <= 0 {
		t.Errorf("CTL should be > 0, got %v", m.Ctl)
	}
}

func TestPhaseForDate(t *testing.T) {
	race := time.Date(2026, 12, 12, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		date    time.Time
		phase   Phase
		weekNum int
	}{
		{time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC), PhaseBase, 1},
		{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), PhaseBase, 3},
		{time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), PhaseBuild, 1},
		{time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), PhasePeak, 1},
		{time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC), PhaseTaper, 1},
		{time.Date(2026, 12, 6, 0, 0, 0, 0, time.UTC), PhaseRace, 2},
	}

	for _, tt := range tests {
		info := PhaseForDate(race, tt.date)
		if info.Phase != tt.phase {
			t.Errorf("date %s: phase = %s, want %s", tt.date.Format("2006-01-02"), info.Phase, tt.phase)
		}
		if info.WeekOfPhase != tt.weekNum {
			t.Errorf("date %s: weekOfPhase = %d, want %d", tt.date.Format("2006-01-02"), info.WeekOfPhase, tt.weekNum)
		}
	}
}
