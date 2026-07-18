package mongo

import (
	"context"
	"time"

	"github.com/dylanewe/coach/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Repository implements ports.Repository.
type Repository struct {
	db       *DB
	raceDate time.Time
}

// NewRepository creates a new MongoDB-backed repository.
func NewRepository(db *DB, raceDate time.Time) *Repository {
	if raceDate.IsZero() {
		raceDate = time.Date(2026, 12, 12, 0, 0, 0, 0, time.UTC)
	}
	return &Repository{db: db, raceDate: raceDate}
}

func (r *Repository) activities() *mongo.Collection {
	return r.db.DB.Collection("activities")
}

func (r *Repository) analyses() *mongo.Collection {
	return r.db.DB.Collection("analysis")
}

func (r *Repository) reports() *mongo.Collection {
	return r.db.DB.Collection("weekly_reports")
}

func (r *Repository) summaries() *mongo.Collection {
	return r.db.DB.Collection("weekly_summaries")
}

func (r *Repository) profiles() *mongo.Collection {
	return r.db.DB.Collection("athlete_profile")
}

// SaveActivity upserts an activity.
func (r *Repository) SaveActivity(ctx context.Context, a domain.Activity) error {
	filter := bson.M{"_id": a.ID}
	update := bson.M{"$set": a}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.activities().UpdateOne(ctx, filter, update, opts)
	return err
}

// GetActivitiesSince returns activities after a given time, sorted newest first.
func (r *Repository) GetActivitiesSince(ctx context.Context, since time.Time) ([]domain.Activity, error) {
	filter := bson.M{"start_date_local": bson.M{"$gte": since}}
	opts := options.Find().SetSort(bson.M{"start_date_local": -1})
	cur, err := r.activities().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []domain.Activity
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ActivityExists checks if an activity is already stored.
func (r *Repository) ActivityExists(ctx context.Context, id string) (bool, error) {
	count, err := r.activities().CountDocuments(ctx, bson.M{"_id": id}, options.Count().SetLimit(1))
	return count > 0, err
}

// SaveAnalysis stores a run analysis.
func (r *Repository) SaveAnalysis(ctx context.Context, a domain.RunAnalysis) error {
	_, err := r.analyses().InsertOne(ctx, a)
	return err
}

// AnalysisExists checks if an analysis for the given activity already exists.
func (r *Repository) AnalysisExists(ctx context.Context, activityID string) (bool, error) {
	count, err := r.analyses().CountDocuments(ctx, bson.M{"activity_id": activityID}, options.Count().SetLimit(1))
	return count > 0, err
}

// SaveWeeklySummary upserts a weekly summary.
func (r *Repository) SaveWeeklySummary(ctx context.Context, s domain.WeekSummary) error {
	s.CreatedAt = time.Now()
	filter := bson.M{"_id": s.WeekStart}
	update := bson.M{"$set": s}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.summaries().UpdateOne(ctx, filter, update, opts)
	return err
}

// GetWeeklySummaries returns weekly summaries on or after since, sorted oldest first.
func (r *Repository) GetWeeklySummaries(ctx context.Context, since time.Time) ([]domain.WeekSummary, error) {
	filter := bson.M{"_id": bson.M{"$gte": since}}
	opts := options.Find().SetSort(bson.M{"_id": 1})
	cur, err := r.summaries().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []domain.WeekSummary
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SaveWeeklyReport stores a weekly report.
func (r *Repository) SaveWeeklyReport(ctx context.Context, rpt domain.WeeklyReport) error {
	_, err := r.reports().InsertOne(ctx, rpt)
	return err
}

// GetLatestWeeklyReport returns the most recent weekly report.
func (r *Repository) GetLatestWeeklyReport(ctx context.Context) (*domain.WeeklyReport, error) {
	opts := options.FindOne().SetSort(bson.M{"generated_at": -1})
	var out domain.WeeklyReport
	err := r.reports().FindOne(ctx, bson.M{}, opts).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAthleteProfile returns the singleton athlete profile.
// If none exists, a default profile is returned.
func (r *Repository) GetAthleteProfile(ctx context.Context) (*domain.AthleteProfile, error) {
	var out domain.AthleteProfile
	err := r.profiles().FindOne(ctx, bson.M{"_id": "athlete"}).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return r.defaultProfile(), nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SaveAthleteProfile upserts the singleton athlete profile.
func (r *Repository) SaveAthleteProfile(ctx context.Context, p domain.AthleteProfile) error {
	p.ID = "athlete"
	p.UpdatedAt = time.Now()
	filter := bson.M{"_id": p.ID}
	update := bson.M{"$set": p}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.profiles().UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *Repository) defaultProfile() *domain.AthleteProfile {
	return &domain.AthleteProfile{
		ID:       "athlete",
		RaceDate: r.raceDate,
		WeeklyTemplate: []domain.ScheduledDay{
			{DayOfWeek: 0, Type: domain.WorkoutLongRun},
			{DayOfWeek: 3, Type: domain.WorkoutEasy},
			{DayOfWeek: 5, Type: domain.WorkoutTempoInterval},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
