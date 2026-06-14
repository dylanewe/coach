package app

import (
	"context"
	"fmt"
	"time"

	"github.com/dylanewe/coach/internal/analytics"
	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/planner"
	"github.com/dylanewe/coach/internal/ports"
	"go.uber.org/zap"
)

// Weekly generates weekly summaries and plans.
type Weekly struct {
	provider ports.ActivityProvider
	repo     ports.Repository
	analyzer ports.Analyzer
	notifier ports.Notifier
	cfg      Config
	log      *zap.Logger
}

// NewWeekly creates a new Weekly generator.
func NewWeekly(provider ports.ActivityProvider, repo ports.Repository, analyzer ports.Analyzer, notifier ports.Notifier, cfg Config, log *zap.Logger) *Weekly {
	return &Weekly{
		provider: provider,
		repo:     repo,
		analyzer: analyzer,
		notifier: notifier,
		cfg:      cfg,
		log:      log,
	}
}

// GenerateReport creates the weekly report and pushes workouts.
func (w *Weekly) GenerateReport(ctx context.Context) error {
	weekEnd := time.Now()
	weekStart := weekEnd.AddDate(0, 0, -7)

	profile, err := w.repo.GetAthleteProfile(ctx)
	if err != nil {
		return fmt.Errorf("fetch athlete profile: %w", err)
	}

	activities, err := w.repo.GetActivitiesSince(ctx, weekStart.AddDate(0, 0, -56))
	if err != nil {
		return fmt.Errorf("fetch activities: %w", err)
	}

	var weekActivities []domain.Activity
	for _, a := range activities {
		if !a.StartDateLocal.Before(weekStart) {
			weekActivities = append(weekActivities, a)
		}
	}

	var totalDist float64
	var totalTime int
	var totalLoad float64
	for _, a := range weekActivities {
		totalDist += a.Distance
		totalTime += a.MovingTime
		totalLoad += a.ICULoad
	}

	phaseInfo := analytics.PhaseForDate(profile.RaceDate, weekEnd)
	loads := analytics.ComputeLoadMetrics(activities, weekEnd)

	summary := domain.WeekSummary{
		WeekStart:     weekStart,
		WeekEnd:       weekEnd,
		Activities:    weekActivities,
		TotalDistance: totalDist,
		TotalTime:     totalTime,
		TotalLoad:     totalLoad,
		Acwr:          loads.Acwr,
		Ctl:           loads.Ctl,
		Atl:           loads.Atl,
		Tsb:           loads.Tsb,
		Phase:         string(phaseInfo.Phase),
		WeekOfPhase:   phaseInfo.WeekOfPhase,
		WeeksToRace:   phaseInfo.WeeksToRace,
	}

	if err := w.repo.SaveWeeklySummary(ctx, summary); err != nil {
		w.log.Error("save weekly summary", zap.Error(err))
	}

	plan, err := w.generatePlan(ctx, summary, *profile)
	if err != nil {
		return fmt.Errorf("generate plan: %w", err)
	}

	report := plan.Report
	if err := w.repo.SaveWeeklyReport(ctx, report); err != nil {
		return fmt.Errorf("save weekly report: %w", err)
	}

	if err := w.notifier.SendWeeklyReport(ctx, w.cfg.EmailTo, report); err != nil {
		w.log.Error("send weekly report", zap.Error(err))
	}

	for _, workout := range plan.Workouts {
		if err := w.provider.CreateWorkout(ctx, w.cfg.AthleteID, workout); err != nil {
			w.log.Error("create workout", zap.String("name", workout.Name), zap.Error(err))
			continue
		}
		w.log.Info("created workout", zap.String("name", workout.Name))
	}

	w.log.Info("weekly report generated",
		zap.Time("week_start", report.WeekStart),
		zap.String("phase", summary.Phase),
		zap.Float64("acwr", summary.Acwr),
		zap.Int("workouts", len(plan.Workouts)),
	)
	return nil
}

// generatePlan asks the LLM for a plan, validates it, and falls back to templates if needed.
func (w *Weekly) generatePlan(ctx context.Context, summary domain.WeekSummary, profile domain.AthleteProfile) (ports.WeeklyPlan, error) {
	llmPlan, err := w.analyzer.GenerateWeeklyPlan(ctx, summary, profile)
	if err != nil {
		w.log.Warn("LLM plan generation failed, using fallback", zap.Error(err))
		return planner.FallbackPlan(summary, profile), nil
	}

	cleaned, warnings, err := planner.ValidateAndNormalize(llmPlan, summary, profile)
	if err != nil {
		w.log.Warn("LLM plan invalid, using fallback", zap.Error(err), zap.Strings("warnings", warnings))
		fallback := planner.FallbackPlan(summary, profile)
		fallback.Report.Recommendations = append(fallback.Report.Recommendations,
			fmt.Sprintf("LLM plan was invalid (%v); used fallback template.", err))
		return fallback, nil
	}

	if len(warnings) > 0 {
		cleaned.Report.Recommendations = append(cleaned.Report.Recommendations, warnings...)
		w.log.Info("LLM plan normalized", zap.Strings("warnings", warnings))
	}

	return cleaned, nil
}
