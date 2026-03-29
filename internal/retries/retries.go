package retries

import (
	"context"
	"errors"
	"time"

	"github.com/LittleAksMax/bids-service/internal/logging"
)

// RetryFunc is the function shape supported by Retry.
type RetryFunc func(context.Context) error

// Retry decorates a RetryFunc with retry behaviour.
func Retry(totalAttempts int, wait time.Duration, logger *logging.Logger, wrappedFunc RetryFunc) RetryFunc {
	if wrappedFunc == nil {
		panic("retries: wrapped function must not be nil")
	}

	return func(ctx context.Context) error {
		for attempt := 1; attempt <= totalAttempts; attempt++ {
			err := wrappedFunc(ctx)
			if err == nil {
				return nil
			}

			if attempt == totalAttempts || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			if ctx != nil && ctx.Err() != nil {
				return err
			}

			if logger != nil {
				logger.Warnf("attempt %d/%d failed: %v; retrying in %s", attempt, totalAttempts, err, wait)
			}

			if wait <= 0 {
				continue
			}

			timer := time.NewTimer(wait)
			if ctx == nil {
				<-timer.C
				continue
			}

			select {
			case <-ctx.Done():
				timer.Stop()
				return err
			case <-timer.C:
			}
		}

		panic("retries: unreachable")
	}
}
