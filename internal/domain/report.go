package domain

import "time"

// WeekSummary aggregates a week's worth of runs for the analyzer.
// It is persisted to MongoDB so load trends are available for future planning.
type WeekSummary struct {
	WeekStart     time.Time  `bson:"_id" json:"week_start"`
	WeekEnd        time.Time  `bson:"week_end" json:"week_end"`
	Activities     []Activity `bson:"activities" json:"activities"`
	TotalDistance  float64    `bson:"total_distance" json:"total_distance"`
	TotalTime      int        `bson:"total_time" json:"total_time"`
	TotalLoad      float64    `bson:"total_load" json:"total_load"`
	Acwr           float64    `bson:"acwr" json:"acwr"`
	Ctl            float64    `bson:"ctl" json:"ctl"`
	Atl            float64    `bson:"atl" json:"atl"`
	Tsb            float64    `bson:"tsb" json:"tsb"`
	Phase          string     `bson:"phase" json:"phase"`
	WeekOfPhase    int        `bson:"week_of_phase" json:"week_of_phase"`
	WeeksToRace    int        `bson:"weeks_to_race" json:"weeks_to_race"`
	CreatedAt      time.Time  `bson:"created_at" json:"created_at"`
}

// WeeklyReport is the LLM-generated weekly summary and plan.
type WeeklyReport struct {
	WeekStart        time.Time `bson:"week_start" json:"week_start"`
	WeekEnd          time.Time `bson:"week_end" json:"week_end"`
	Summary          string    `bson:"summary" json:"summary"`
	Recommendations  []string  `bson:"recommendations" json:"recommendations"`
	NextWeekWorkouts []Workout `bson:"next_week_workouts" json:"next_week_workouts"`
	GeneratedAt      time.Time `bson:"generated_at" json:"generated_at"`
}
