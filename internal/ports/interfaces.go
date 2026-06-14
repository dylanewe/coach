package ports

import (
	"context"
	"time"

	"github.com/dylanewe/coach/internal/domain"
)

// ActivityProvider fetches completed activities and creates planned workouts.
type ActivityProvider interface {
	FetchActivities(ctx context.Context, athleteID string, since time.Time) ([]domain.Activity, error)
	CreateWorkout(ctx context.Context, athleteID string, w domain.Workout) error
}

// Analyzer generates coaching insights via an LLM.
type Analyzer interface {
	AnalyzeRun(ctx context.Context, activity domain.Activity, history []domain.Activity, profile domain.AthleteProfile) (domain.RunAnalysis, error)
	GenerateWeeklyPlan(ctx context.Context, week domain.WeekSummary, profile domain.AthleteProfile) (WeeklyPlan, error)
}

// WeeklyPlan is the output from GenerateWeeklyPlan.
type WeeklyPlan struct {
	Report   domain.WeeklyReport
	Workouts []domain.Workout
}

// Repository persists domain objects.
type Repository interface {
	SaveActivity(ctx context.Context, a domain.Activity) error
	GetActivitiesSince(ctx context.Context, since time.Time) ([]domain.Activity, error)
	ActivityExists(ctx context.Context, id string) (bool, error)
	SaveAnalysis(ctx context.Context, a domain.RunAnalysis) error
	AnalysisExists(ctx context.Context, activityID string) (bool, error)

	// Weekly summaries
	SaveWeeklySummary(ctx context.Context, s domain.WeekSummary) error
	GetWeeklySummaries(ctx context.Context, since time.Time) ([]domain.WeekSummary, error)

	SaveWeeklyReport(ctx context.Context, r domain.WeeklyReport) error
	GetLatestWeeklyReport(ctx context.Context) (*domain.WeeklyReport, error)

	// Athlete profile
	GetAthleteProfile(ctx context.Context) (*domain.AthleteProfile, error)
	SaveAthleteProfile(ctx context.Context, p domain.AthleteProfile) error
}

// Notifier delivers reports to the user.
type Notifier interface {
	SendRunAnalysis(ctx context.Context, to string, analysis domain.RunAnalysis, activity domain.Activity) error
	SendWeeklyReport(ctx context.Context, to string, report domain.WeeklyReport) error
}
