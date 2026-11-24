package maintenance

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"

	"spacegame-backend/internal/store"
)

type CleanupConfig struct {
	Interval  time.Duration `env:"CLEANUP_INTERVAL"`
	Retention time.Duration `env:"EXPIRED_SESSION_RETENTION"`
	BatchSize int32         `env:"CLEANUP_BATCH_SIZE"`
}

func CleanupConfigFromEnv() (CleanupConfig, error) {
	config := CleanupConfig{Interval: 5 * time.Minute, Retention: 24 * time.Hour, BatchSize: 1000}
	if err := env.Parse(&config); err != nil {
		return config, err
	}
	return config, config.Validate()
}

func (c CleanupConfig) Validate() error {
	if c.Interval < time.Second || c.Retention < 0 || c.BatchSize < 1 || c.BatchSize > 10000 {
		return errors.New("cleanup interval must be at least 1s, retention must be non-negative, and batch size must be 1 to 10000")
	}
	return nil
}

// sweepBatches bounds the database work done by one sweep.
const sweepBatches = 10

// defaultDrainDelay separates follow-up sweeps while a backlog is drained.
const defaultDrainDelay = time.Second

type Cleaner struct {
	Queries store.Querier
	Config  CleanupConfig
	Logger  *slog.Logger
	// Observe receives each sweep's result. Saturated reports that the sweep
	// reached its batch cap, so more expired sessions are likely waiting.
	Observe func(deleted int64, saturated bool, err error)
	// DrainDelay is the pause before a follow-up sweep after a saturated one.
	DrainDelay time.Duration
}

// Sweep removes at most ten batches of expired, unfinished sessions. It reports
// saturated when every batch was full and the backlog may not be empty.
func (c Cleaner) Sweep(ctx context.Context, now time.Time) (total int64, saturated bool, err error) {
	for range sweepBatches {
		deleted, err := c.Queries.DeleteExpiredSessions(ctx, store.DeleteExpiredSessionsParams{
			Cutoff: now.Add(-c.Config.Retention), BatchSize: c.Config.BatchSize,
		})
		total += deleted
		if err != nil {
			return total, false, err
		}
		if deleted < int64(c.Config.BatchSize) {
			return total, false, nil
		}
	}
	return total, true, nil
}

// Run sweeps immediately and then every interval. When a sweep reaches its
// batch cap, further bounded sweeps follow after DrainDelay until the backlog
// is drained, so sustained abandoned-session traffic cannot outgrow cleanup.
func (c Cleaner) Run(ctx context.Context) {
	ticker := time.NewTicker(c.Config.Interval)
	defer ticker.Stop()
	drainDelay := c.DrainDelay
	if drainDelay <= 0 {
		drainDelay = defaultDrainDelay
	}
	for {
		if ctx.Err() != nil {
			return
		}
		sweepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		deleted, saturated, err := c.Sweep(sweepCtx, time.Now().UTC())
		cancel()
		if c.Observe != nil {
			c.Observe(deleted, saturated, err)
		}
		if err != nil && ctx.Err() == nil {
			c.Logger.Error("expired session cleanup failed", "error", err)
		} else if deleted > 0 {
			c.Logger.Info("expired sessions deleted", "count", deleted, "backlog", saturated)
		}
		if !c.wait(ctx, ticker.C, saturated, drainDelay) {
			return
		}
	}
}

// wait blocks until the next sweep is due and reports false on cancellation.
func (c Cleaner) wait(ctx context.Context, tick <-chan time.Time, saturated bool, drainDelay time.Duration) bool {
	if saturated {
		tick = time.After(drainDelay)
	}
	select {
	case <-ctx.Done():
		return false
	case <-tick:
		return true
	}
}
