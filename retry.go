package skailar

import (
	"context"
	"math/rand"
	"time"
)

const (
	backoffBase     = 500 * time.Millisecond
	backoffCap      = 8 * time.Second
	maxRetryAfter   = 60 * time.Second
	backoffMaxShift = 20
)

// sleepBackoff waits before the next retry attempt, honouring ctx. It returns a
// [KindAborted] error if the context is cancelled mid-wait, so the retry loop
// never blocks on an un-interruptible timer.
func sleepBackoff(ctx context.Context, attempt, retryAfterSecs int) error {
	delay := backoffDelay(attempt, retryAfterSecs)
	if delay <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return abortedError(ctx.Err())
	case <-timer.C:
		return nil
	}
}

// backoffDelay computes the wait before a retry. A Retry-After value (in
// seconds) takes precedence, capped at 60 seconds; otherwise it is exponential
// with full jitter in [0, min(cap, base*2^attempt)].
func backoffDelay(attempt, retryAfterSecs int) time.Duration {
	if retryAfterSecs > 0 {
		d := time.Duration(retryAfterSecs) * time.Second
		if d > maxRetryAfter {
			return maxRetryAfter
		}
		return d
	}

	shift := attempt
	if shift > backoffMaxShift {
		shift = backoffMaxShift
	}
	window := backoffBase << shift
	if window > backoffCap {
		window = backoffCap
	}
	if window <= 0 {
		return 0
	}
	// Full jitter: a uniformly random delay in [0, window].
	return time.Duration(rand.Int63n(int64(window) + 1))
}
