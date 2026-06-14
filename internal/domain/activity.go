package domain

import "time"

// Activity represents a completed run from Intervals.icu.
type Activity struct {
	ID                 string    `bson:"_id" json:"id"`
	Name               string    `bson:"name" json:"name"`
	Type               string    `bson:"type" json:"type"`
	SubType            string    `bson:"sub_type" json:"sub_type"`
	StartDateLocal     time.Time `bson:"start_date_local" json:"start_date_local"`
	Distance           float64   `bson:"distance" json:"distance"`                       // meters
	MovingTime         int       `bson:"moving_time" json:"moving_time"`                 // seconds
	ElapsedTime        int       `bson:"elapsed_time" json:"elapsed_time"`               // seconds
	TotalElevationGain float64   `bson:"total_elevation_gain" json:"total_elevation_gain"`
	AverageSpeed       float64   `bson:"average_speed" json:"average_speed"`             // m/s
	AverageHeartrate   float64   `bson:"average_heartrate" json:"average_heartrate"`
	MaxHeartrate       float64   `bson:"max_heartrate" json:"max_heartrate"`
	AverageCadence     float64   `bson:"average_cadence" json:"average_cadence"`
	ICULoad            float64   `bson:"icu_load" json:"icu_load"`                       // Intervals.icu training load (proxy for TSS)
	ICUCTL             float64   `bson:"icu_ctl" json:"icu_ctl"`                         // Intervals.icu chronic training load
	ICUATL             float64   `bson:"icu_atl" json:"icu_atl"`                         // Intervals.icu acute training load
	PaceAtHRZ2Ms       float64   `bson:"pace_at_hr_z2_ms" json:"pace_at_hr_z2_ms"`       // Avg pace while in HR Z2 (m/s)
	Description        string    `bson:"description" json:"description"`
	CreatedAt          time.Time `bson:"created_at" json:"created_at"`
	SyncedAt           time.Time `bson:"synced_at" json:"synced_at"`
}
