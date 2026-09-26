package workerpool

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRunPool_TimeoutDoesNotBlockPool(t *testing.T) {
	withTimeout(t, 3*time.Second, func() {
		jobs := make(chan Job, 3)
		jobs <- Job{ID: "slow-1", Fetch: slowFetch(500 * time.Millisecond)}
		jobs <- Job{ID: "fast-1", Fetch: func(ctx context.Context) (int, error) {
			select {
			case <-time.After(20 * time.Millisecond):
				return 11, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}}
		jobs <- Job{ID: "fast-2", Fetch: func(ctx context.Context) (int, error) {
			select {
			case <-time.After(20 * time.Millisecond):
				return 22, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}}
		close(jobs)

		start := time.Now()
		results := RunPool(jobs, 3, 100*time.Millisecond)

		seen := map[string]bool{}
		for r := range results {
			seen[r.JobID] = true
			if r.JobID == "slow-1" {
				if r.Err != context.DeadlineExceeded {
					t.Fatalf("slow-1: got err %v, want %v", r.Err, context.DeadlineExceeded)
				}
				continue
			}
			if r.Err != nil {
				t.Fatalf("%s: unexpected err: %v", r.JobID, r.Err)
			}
			if r.Size == 0 {
				t.Fatalf("%s: size should not be zero for successful job", r.JobID)
			}
		}

		if len(seen) != 3 {
			t.Fatalf("got %d results, want 3", len(seen))
		}
		if time.Since(start) > time.Second {
			t.Fatalf("pool took too long: %v; timed-out job blocked the pool", time.Since(start))
		}
	})
}

func TestRunPool_TimeoutReturnsDeadlineExceeded(t *testing.T) {
	withTimeout(t, 3*time.Second, func() {
		jobs := make(chan Job, 1)
		jobs <- Job{ID: "job-timeout", Fetch: slowFetch(300 * time.Millisecond)}
		close(jobs)

		results := RunPool(jobs, 1, 50*time.Millisecond)
		for r := range results {
			if r.JobID != "job-timeout" {
				t.Fatalf("unexpected job id %q", r.JobID)
			}
			if r.Err == nil {
				t.Fatal("expected timeout error, got nil")
			}
			if r.Err != context.DeadlineExceeded {
				t.Fatalf("err = %v, want %v", r.Err, context.DeadlineExceeded)
			}
			if r.Size != 0 {
				t.Fatalf("size = %d, want 0 for timeout", r.Size)
			}
		}
	})
}

func TestRunPool_UsesAllWorkers(t *testing.T) {
	withTimeout(t, 3*time.Second, func() {
		const numWorkers = 3
		jobs := make(chan Job, 6)
		for i := 0; i < 6; i++ {
			jobs <- Job{ID: fmt.Sprintf("job-%d", i), Fetch: func(ctx context.Context) (int, error) {
				select {
				case <-time.After(20 * time.Millisecond):
					return 1, nil
				case <-ctx.Done():
					return 0, ctx.Err()
				}
			}}
		}
		close(jobs)

		results := RunPool(jobs, numWorkers, time.Second)
		count := 0
		for range results {
			count++
		}
		if count != 6 {
			t.Fatalf("got %d results, want 6", count)
		}
	})
}
