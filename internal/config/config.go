package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Intervals struct {
		APIKey    string
		AthleteID string
	}
	LLM struct {
		Provider string
		APIKey   string
		BaseURL  string
		Model    string
	}
	MongoDB struct {
		URI string
		DB  string
	}
	Email struct {
		SMTPHost string
		SMTPPort int
		SMTPUser string
		SMTPPass string
		To       string
		From     string
	}
	Notify struct {
		BaseURL string
		APIKey  string
		Channel string
	}
	App struct {
		SyncCron    string
		WeeklyCron  string
		Location    *time.Location
		HistoryDays int // how many days of history to pass to analyzer
	}
	Profile struct {
		RaceDate time.Time
	}
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config

	cfg.Intervals.APIKey = os.Getenv("INTERVALS_API_KEY")
	cfg.Intervals.AthleteID = os.Getenv("INTERVALS_ATHLETE_ID")

	cfg.LLM.Provider = getEnvDefault("LLM_PROVIDER", "ollama")
	cfg.LLM.APIKey = os.Getenv("LLM_API_KEY")

	switch cfg.LLM.Provider {
	case "ollama":
		cfg.LLM.BaseURL = getEnvDefault("LLM_BASE_URL", "http://localhost:11434")
		cfg.LLM.Model = getEnvDefault("LLM_MODEL", "gemma4:12b")
	case "kimi":
		cfg.LLM.BaseURL = getEnvDefault("LLM_BASE_URL", "https://api.moonshot.cn/v1")
		cfg.LLM.Model = getEnvDefault("LLM_MODEL", "moonshot-v1-128k")
	default:
		return nil, fmt.Errorf("unsupported LLM_PROVIDER %q (use 'ollama' or 'kimi')", cfg.LLM.Provider)
	}

	cfg.MongoDB.URI = getEnvDefault("MONGODB_URI", "mongodb://localhost:27017")
	cfg.MongoDB.DB = getEnvDefault("MONGODB_DB", "coach")

	cfg.Email.SMTPHost = getEnvDefault("SMTP_HOST", "smtp.gmail.com")
	cfg.Email.SMTPPort = mustAtoi(getEnvDefault("SMTP_PORT", "587"))
	cfg.Email.SMTPUser = os.Getenv("SMTP_USER")
	cfg.Email.SMTPPass = os.Getenv("SMTP_PASS")
	cfg.Email.To = os.Getenv("EMAIL_TO")
	cfg.Email.From = os.Getenv("EMAIL_FROM")

	cfg.Notify.BaseURL = getEnvDefault("NOTIFY_BASE_URL", "http://localhost:8080")
	cfg.Notify.APIKey = os.Getenv("NOTIFY_API_KEY")
	cfg.Notify.Channel = getEnvDefault("NOTIFY_CHANNEL", "email")

	cfg.App.SyncCron = getEnvDefault("SYNC_CRON", "0 23 * * *")
	cfg.App.WeeklyCron = getEnvDefault("WEEKLY_CRON", "0 23 * * 6")
	cfg.App.HistoryDays = mustAtoi(getEnvDefault("HISTORY_DAYS", "30"))

	raceDateStr := getEnvDefault("RACE_DATE", "2026-12-12")
	raceDate, err := time.Parse("2006-01-02", raceDateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RACE_DATE %q: %w", raceDateStr, err)
	}
	cfg.Profile.RaceDate = raceDate

	tz := getEnvDefault("TZ", "Asia/Kuala_Lumpur")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid TZ %q: %w", tz, err)
	}
	cfg.App.Location = loc

	if cfg.Intervals.APIKey == "" || cfg.Intervals.AthleteID == "" {
		return nil, fmt.Errorf("INTERVALS_API_KEY and INTERVALS_ATHLETE_ID are required")
	}
	if cfg.LLM.Provider == "kimi" && cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required for kimi provider")
	}
	if cfg.Notify.APIKey == "" {
		return nil, fmt.Errorf("NOTIFY_API_KEY is required")
	}
	if cfg.Notify.Channel != "email" && cfg.Notify.Channel != "telegram" {
		return nil, fmt.Errorf("NOTIFY_CHANNEL must be 'email' or 'telegram'")
	}

	return &cfg, nil
}

// RaceDate returns the configured race date in UTC.
func (c *Config) RaceDate() time.Time {
	return c.Profile.RaceDate.UTC()
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
