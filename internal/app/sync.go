package app

import (
	"context"
	"fmt"
	"time"

	"github.com/dylanewe/coach/internal/domain"
	"github.com/dylanewe/coach/internal/ports"
	"go.uber.org/zap"
)

// Syncer orchestrates daily activity fetching and analysis.
type Syncer struct {
	provider ports.ActivityProvider
	repo     ports.Repository
	analyzer ports.Analyzer
	notifier ports.Notifier
	cfg      Config
	log      *zap.Logger
}

// Config holds sync-related settings.
type Config struct {
	AthleteID   string
	EmailTo     string
	HistoryDays int
}

// NewSyncer creates a new Syncer.
func NewSyncer(provider ports.ActivityProvider, repo ports.Repository, analyzer ports.Analyzer, notifier ports.Notifier, cfg Config, log *zap.Logger) *Syncer {
	return &Syncer{
		provider: provider,
		repo:     repo,
		analyzer: analyzer,
		notifier: notifier,
		cfg:      cfg,
		log:      log,
	}
}

// DailySync fetches new runs and analyzes any that are new or missing an analysis.
// If force is true, existing analyses are re-analyzed.
func (s *Syncer) DailySync(ctx context.Context, force bool) error {
	profile, err := s.repo.GetAthleteProfile(ctx)
	if err != nil {
		return fmt.Errorf("fetch athlete profile: %w", err)
	}

	since := time.Now().AddDate(0, 0, -s.cfg.HistoryDays)
	// Try to find the most recent activity to narrow the window
	latest, err := s.repo.GetLatestWeeklyReport(ctx)
	if err != nil {
		s.log.Warn("failed to get latest weekly report", zap.Error(err))
	}
	if latest != nil {
		since = latest.WeekStart.AddDate(0, 0, -7)
	}

	activities, err := s.provider.FetchActivities(ctx, s.cfg.AthleteID, since)
	if err != nil {
		return fmt.Errorf("fetch activities: %w", err)
	}

	s.log.Info("fetched activities", zap.Int("count", len(activities)))

	var analyzed, skipped, failed int
	for _, act := range activities {
		if err := s.repo.SaveActivity(ctx, act); err != nil {
			s.log.Error("save activity", zap.String("id", act.ID), zap.Error(err))
			failed++
			continue
		}

		if !force {
			exists, err := s.repo.AnalysisExists(ctx, act.ID)
			if err != nil {
				s.log.Error("check analysis exists", zap.String("id", act.ID), zap.Error(err))
				failed++
				continue
			}
			if exists {
				skipped++
				continue
			}
		}

		history, err := s.repo.GetActivitiesSince(ctx, time.Now().AddDate(0, 0, -s.cfg.HistoryDays))
		if err != nil {
			s.log.Error("fetch history", zap.Error(err))
			history = nil
		}
		// Exclude the current activity from its own history
		filtered := make([]domain.Activity, 0, len(history))
		for _, h := range history {
			if h.ID != act.ID {
				filtered = append(filtered, h)
			}
		}

		analysis, err := s.analyzer.AnalyzeRun(ctx, act, filtered, *profile)
		if err != nil {
			s.log.Error("analyze run", zap.String("id", act.ID), zap.Error(err))
			failed++
			continue
		}
		analysis.ActivityID = act.ID

		if err := s.repo.SaveAnalysis(ctx, analysis); err != nil {
			s.log.Error("save analysis", zap.String("id", act.ID), zap.Error(err))
			failed++
			continue
		}

		if err := s.notifier.SendRunAnalysis(ctx, s.cfg.EmailTo, analysis, act); err != nil {
			s.log.Error("send run analysis", zap.String("id", act.ID), zap.Error(err))
			failed++
			continue
		}

		s.log.Info("analyzed and notified", zap.String("activity_id", act.ID), zap.String("name", act.Name))
		analyzed++
	}

	s.log.Info("daily sync complete", zap.Int("fetched", len(activities)), zap.Int("analyzed", analyzed), zap.Int("skipped", skipped), zap.Int("failed", failed))
	return nil
}
