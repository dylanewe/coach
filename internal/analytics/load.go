package analytics

import (
	"math"
	"time"

	"github.com/dylanewe/coach/internal/domain"
)

// LoadMetrics holds the computed load/recovery metrics for a point in time.
type LoadMetrics struct {
	Acwr        float64
	Ctl         float64
	Atl         float64
	Tsb         float64
	AcuteLoad   float64
	ChronicLoad float64
}

// DailyLoad is the training load assigned to a single calendar day.
type DailyLoad struct {
	Date time.Time
	Load float64
}

// ComputeLoadMetrics calculates ACWR, CTL, ATL, and TSB as of the given end date.
//
// ACWR uses the rolling-average model: sum of daily load for the past 7 days
// divided by the average of weekly totals for the past 4 weeks.
//
// CTL/ATL use exponentially weighted moving averages with 42-day and 7-day
// time constants respectively.
func ComputeLoadMetrics(activities []domain.Activity, endDate time.Time) LoadMetrics {
	daily := dailyLoads(activities, endDate)

	acute := sumLastNDays(daily, endDate, 7)
	weeklyTotals := lastNWeekTotals(daily, endDate, 4)

	var chronic float64
	if len(weeklyTotals) > 0 {
		for _, w := range weeklyTotals {
			chronic += w
		}
		chronic /= float64(len(weeklyTotals))
	}

	var acwr float64
	if chronic > 0 {
		acwr = acute / chronic
	}

	ctl := ewma(daily, endDate, 42)
	atl := ewma(daily, endDate, 7)

	return LoadMetrics{
		Acwr:        round(acwr, 2),
		Ctl:         round(ctl, 2),
		Atl:         round(atl, 2),
		Tsb:         round(ctl-atl, 2),
		AcuteLoad:   round(acute, 2),
		ChronicLoad: round(chronic, 2),
	}
}

// dailyLoads collapses activities into a per-day load map keyed by UTC date.
// If multiple activities occur on the same day, their loads are summed.
func dailyLoads(activities []domain.Activity, endDate time.Time) map[time.Time]float64 {
	loads := make(map[time.Time]float64)
	for _, a := range activities {
		if a.StartDateLocal.IsZero() {
			continue
		}
		d := a.StartDateLocal.UTC().Truncate(24 * time.Hour)
		loads[d] += a.ICULoad
	}
	return loads
}

// sumLastNDays returns the total load for the N days ending on endDate (inclusive).
func sumLastNDays(loads map[time.Time]float64, endDate time.Time, n int) float64 {
	end := endDate.UTC().Truncate(24 * time.Hour)
	var sum float64
	for i := 0; i < n; i++ {
		d := end.AddDate(0, 0, -i)
		sum += loads[d]
	}
	return sum
}

// lastNWeekTotals returns the total load for each of the last N complete Monday-Sunday weeks
// ending on or before endDate. The most recent week is first.
func lastNWeekTotals(loads map[time.Time]float64, endDate time.Time, n int) []float64 {
	end := endDate.UTC().Truncate(24 * time.Hour)
	// Find the Sunday of the current week (Go weekday: 0=Sunday).
	daysSinceSunday := int(end.Weekday()) // 0..6
	sunday := end.AddDate(0, 0, -daysSinceSunday)

	out := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		weekEnd := sunday.AddDate(0, 0, -i*7)
		weekStart := weekEnd.AddDate(0, 0, -6)
		var total float64
		for d := weekStart; !d.After(weekEnd); d = d.AddDate(0, 0, 1) {
			total += loads[d]
		}
		out = append(out, total)
	}
	return out
}

// ewma computes an exponentially weighted moving average of daily load as of endDate.
// The decay constant is derived from the supplied timeConstant in days.
func ewma(loads map[time.Time]float64, endDate time.Time, timeConstant int) float64 {
	if len(loads) == 0 {
		return 0
	}
	end := endDate.UTC().Truncate(24 * time.Hour)

	// Find the earliest date in the data so we know where to start.
	var earliest time.Time
	for d := range loads {
		if earliest.IsZero() || d.Before(earliest) {
			earliest = d
		}
	}

	alpha := 1.0 - math.Exp(-1.0/float64(timeConstant))

	// Seed with the average daily load over the time-constant window or available history.
	seedWindow := timeConstant
	if seedWindow > int(end.Sub(earliest).Hours()/24)+1 {
		seedWindow = int(end.Sub(earliest).Hours()/24) + 1
	}
	if seedWindow < 1 {
		seedWindow = 1
	}
	avg := sumLastNDays(loads, end, seedWindow) / float64(seedWindow)
	value := avg

	// Walk forward from earliest to end, applying EWMA.
	for d := earliest; !d.After(end); d = d.AddDate(0, 0, 1) {
		load := loads[d]
		value = value + alpha*(load-value)
	}
	return value
}

func round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}
