package domain

import "time"

// AthleteProfile holds static/semi-static athlete configuration.
// It is stored as a singleton document with _id = "athlete".
type AthleteProfile struct {
	ID              string         `bson:"_id" json:"id"`
	RaceDate        time.Time      `bson:"race_date" json:"race_date"`
	MaxHR           int            `bson:"max_hr" json:"max_hr"`
	ThresholdHR     int            `bson:"threshold_hr" json:"threshold_hr"`
	ThresholdPaceMs float64        `bson:"threshold_pace_ms" json:"threshold_pace_ms"`
	RestingHR       int            `bson:"resting_hr" json:"resting_hr"`
	WeeklyTemplate  []ScheduledDay `bson:"weekly_template" json:"weekly_template"`
	CreatedAt       time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `bson:"updated_at" json:"updated_at"`
}

// ScheduledDay maps a weekday to its default workout type.
// DayOfWeek follows Go's time.Weekday: 0=Sunday, 3=Wednesday, 5=Friday.
type ScheduledDay struct {
	DayOfWeek int    `bson:"day_of_week" json:"day_of_week"`
	Type      string `bson:"type" json:"type"`
}
