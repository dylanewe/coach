package domain

import "time"

// RunAnalysis is the LLM-generated feedback for a single run.
type RunAnalysis struct {
	ActivityID          string    `bson:"activity_id" json:"activity_id"`
	Summary             string    `bson:"summary" json:"summary"`
	Positives           []string  `bson:"positives" json:"positives"`
	AreasForImprovement []string  `bson:"areas_for_improvement" json:"areas_for_improvement"`
	SuggestedNextSession string   `bson:"suggested_next_session" json:"suggested_next_session"`
	FatigueScore        int       `bson:"fatigue_score" json:"fatigue_score"` // 1-10
	GeneratedAt         time.Time `bson:"generated_at" json:"generated_at"`
}
