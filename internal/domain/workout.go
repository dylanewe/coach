package domain

import "time"

// Workout represents a planned future session to push to Intervals.icu.
type Workout struct {
	ID          int       `bson:"id,omitempty" json:"id,omitempty"`
	AthleteID   string    `bson:"athlete_id" json:"athlete_id"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	Type        string    `bson:"type" json:"type"`
	SubType     string    `bson:"sub_type" json:"sub_type"`
	Day         int       `bson:"day" json:"day"` // day offset from plan start or absolute day
	MovingTime  int       `bson:"moving_time" json:"moving_time"`
	Distance    float64   `bson:"distance" json:"distance"`
	Target      string    `bson:"target" json:"target"`
	Tags        []string  `bson:"tags" json:"tags"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}
