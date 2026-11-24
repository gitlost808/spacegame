package maintenance

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"spacegame-backend/internal/store"
)

type cleanupStore struct {
	store.Querier
	call func(store.DeleteExpiredSessionsParams) (int64, error)
}

func (s cleanupStore) DeleteExpiredSessions(_ context.Context, p store.DeleteExpiredSessionsParams) (int64, error) {
	return s.call(p)
}

func TestCleanupBatchesAndRetention(t *testing.T) {
	now := time.Now().UTC()
	calls := 0
	cleaner := Cleaner{Config: CleanupConfig{Retention: 24 * time.Hour, BatchSize: 2}, Queries: cleanupStore{
		call: func(p store.DeleteExpiredSessionsParams) (int64, error) {
			if p.Cutoff != now.Add(-24*time.Hour) || p.BatchSize != 2 {
				t.Fatalf("unexpected cleanup parameters: %+v", p)
			}
			calls++
			if calls == 1 {
				return 2, nil
			}
			return 1, nil
		},
	}}
	if deleted, saturated, err := cleaner.Sweep(context.Background(), now); err != nil || deleted != 3 || calls != 2 || saturated {
		t.Fatalf("deleted=%d calls=%d saturated=%v error=%v", deleted, calls, saturated, err)
	}
}

func TestCleanupBudgetAndPartialFailure(t *testing.T) {
	calls := 0
	cleaner := Cleaner{Config: CleanupConfig{BatchSize: 2}, Queries: cleanupStore{
		call: func(store.DeleteExpiredSessionsParams) (int64, error) { calls++; return 2, nil },
	}}
	if deleted, saturated, err := cleaner.Sweep(context.Background(), time.Now()); err != nil || deleted != 20 || calls != 10 || !saturated {
		t.Fatalf("sweep must be bounded and report saturation: deleted=%d calls=%d saturated=%v error=%v", deleted, calls, saturated, err)
	}
	calls = 0
	want := errors.New("database unavailable")
	cleaner.Queries = cleanupStore{call: func(store.DeleteExpiredSessionsParams) (int64, error) {
		calls++
		if calls == 1 {
			return 2, nil
		}
		return 0, want
	}}
	if deleted, _, err := cleaner.Sweep(context.Background(), time.Now()); deleted != 2 || !errors.Is(err, want) {
		t.Fatalf("partial sweep: deleted=%d error=%v", deleted, err)
	}
}

func TestRunDrainsBacklogBeyondOneSweep(t *testing.T) {
	const backlog = 55 // more than one sweep of ten batches of two
	var mu sync.Mutex
	remaining := int64(backlog)
	results := make(chan bool, 10)
	cleaner := Cleaner{
		Config:     CleanupConfig{Interval: time.Hour, BatchSize: 2},
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		DrainDelay: time.Millisecond,
		Queries: cleanupStore{call: func(p store.DeleteExpiredSessionsParams) (int64, error) {
			mu.Lock()
			defer mu.Unlock()
			n := min(remaining, int64(p.BatchSize))
			remaining -= n
			return n, nil
		}},
		Observe: func(_ int64, saturated bool, err error) {
			if err != nil {
				t.Errorf("unexpected cleanup error: %v", err)
			}
			results <- saturated
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { cleaner.Run(ctx); close(done) }()
	want := []bool{true, true, false} // 20 + 20 + 15 rows, without waiting an hour
	for i, saturated := range want {
		select {
		case got := <-results:
			if got != saturated {
				t.Fatalf("sweep %d saturated=%v, want %v", i, got, saturated)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("sweep %d did not run; saturated sweeps must schedule follow-ups", i)
		}
	}
	select {
	case <-results:
		t.Fatal("an unsaturated sweep must wait for the next interval")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	<-done
	if remaining != 0 {
		t.Fatalf("%d expired sessions left behind", remaining)
	}
}

func TestCleanupConfig(t *testing.T) {
	for _, key := range []string{"CLEANUP_INTERVAL", "EXPIRED_SESSION_RETENTION", "CLEANUP_BATCH_SIZE"} {
		t.Setenv(key, "")
	}
	c, err := CleanupConfigFromEnv()
	if err != nil || c.Interval != 5*time.Minute || c.Retention != 24*time.Hour || c.BatchSize != 1000 {
		t.Fatalf("defaults=%+v error=%v", c, err)
	}
	t.Setenv("CLEANUP_INTERVAL", "10s")
	t.Setenv("EXPIRED_SESSION_RETENTION", "0s")
	t.Setenv("CLEANUP_BATCH_SIZE", "25")
	if got, err := CleanupConfigFromEnv(); err != nil || got.Interval != 10*time.Second || got.Retention != 0 || got.BatchSize != 25 {
		t.Fatalf("overrides=%+v error=%v", got, err)
	}
	for _, tt := range []struct{ name, value string }{
		{"CLEANUP_INTERVAL", "invalid"},
		{"CLEANUP_INTERVAL", "100ms"},
		{"EXPIRED_SESSION_RETENTION", "-1s"},
		{"CLEANUP_BATCH_SIZE", "not-an-integer"},
		{"CLEANUP_BATCH_SIZE", "2147483648"},
		{"CLEANUP_BATCH_SIZE", "0"},
		{"CLEANUP_BATCH_SIZE", "10001"},
	} {
		t.Run(tt.name+"="+tt.value, func(t *testing.T) {
			t.Setenv(tt.name, tt.value)
			if _, err := CleanupConfigFromEnv(); err == nil {
				t.Fatalf("%s=%s must fail", tt.name, tt.value)
			}
		})
	}
}
