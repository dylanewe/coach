package app

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"go.uber.org/zap"
)

// Scheduler wraps gocron and holds jobs.
type Scheduler struct {
	sched    gocron.Scheduler
	syncer   *Syncer
	weekly   *Weekly
	location *time.Location
	log      *zap.Logger
}

// NewScheduler builds a scheduler with the given cron expressions.
func NewScheduler(syncCron, weeklyCron string, loc *time.Location, syncer *Syncer, weekly *Weekly, log *zap.Logger) (*Scheduler, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return nil, fmt.Errorf("new scheduler: %w", err)
	}

	if _, err := s.NewJob(
		gocron.CronJob(syncCron, false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := syncer.DailySync(ctx, false); err != nil {
				log.Error("daily sync failed", zap.Error(err))
			}
		}),
		gocron.WithName("daily-sync"),
	); err != nil {
		return nil, fmt.Errorf("register daily sync: %w", err)
	}

	if _, err := s.NewJob(
		gocron.CronJob(weeklyCron, false),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if err := weekly.GenerateReport(ctx); err != nil {
				log.Error("weekly report failed", zap.Error(err))
			}
		}),
		gocron.WithName("weekly-report"),
	); err != nil {
		return nil, fmt.Errorf("register weekly report: %w", err)
	}

	return &Scheduler{
		sched:    s,
		syncer:   syncer,
		weekly:   weekly,
		location: loc,
		log:      log,
	}, nil
}

// Start begins the scheduler.
func (s *Scheduler) Start() {
	s.sched.Start()
	s.log.Info("scheduler started", zap.String("location", s.location.String()))
}

// Shutdown stops the scheduler gracefully.
func (s *Scheduler) Shutdown() error {
	return s.sched.Shutdown()
}
