# Coach — Personal Running Coach Daemon

A Go daemon that pulls completed runs from [Intervals.icu](https://intervals.icu), analyzes them with a local LLM via [Ollama](https://ollama.com) (or Kimi/Moonshot), and emails coaching feedback. Every Saturday it generates a weekly report and pushes planned workouts back to Intervals.icu.

The coach tracks periodization phase, weekly load, ACWR, CTL/ATL/TSB, and uses those guardrails to inform the LLM's training plan.

## Schedule
- **Daily @ 23:00** — Sync new runs, analyze, email report
- **Saturday @ 23:00** — Weekly summary + next week's plan

Hard constraints in the coaching prompt:
- **Sunday**: Long Run
- **Wednesday**: Easy Run
- **Friday**: Tempo / Interval / Speed Work

## Quick Start

1. Copy `.env.example` to `.env` and fill in your credentials.
   - For Intervals.icu, use `INTERVALS_ATHLETE_ID=0` when using a personal API key.
   - For MongoDB with auth, include credentials directly in `MONGODB_URI`.
   - For LLM, default is local Ollama (`gemma4:12b`). Set `LLM_PROVIDER=kimi` to use Moonshot cloud.
2. Ensure MongoDB is running and Ollama is running with your chosen model (e.g. `ollama run gemma4:12b`).
3. Build and run:
   ```bash
   make build
   make run
   ```

## Manual runs (no cron wait)

You can trigger either job immediately without waiting for the scheduler:

```bash
# Sync new runs now
./bin/coachd -sync

# Generate weekly report + next week's plan now
./bin/coachd -weekly

# Run the daemon with cron scheduling (default)
./bin/coachd
# or explicitly
./bin/coachd -daemon
```

## Architecture
Hexagonal / Ports & Adapters:
- **Adapters**: Intervals.icu API, Kimi LLM, MongoDB, SMTP
- **Ports**: Interfaces for swapping LLM, DB, or notifier later
- **App**: Use-cases (daily sync, weekly report)

## Deployment
Built as a single binary with env-based config. Container-ready for Proxmox/Docker:
```bash
make docker-build
```

## Requirements
- Go 1.26+
- MongoDB
- Intervals.icu API key
- Kimi / Moonshot API key
- SMTP credentials for email delivery
