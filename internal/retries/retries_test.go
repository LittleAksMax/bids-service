package retries

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	retried := Retry(2, 0, nil, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("boom")
		}

		return nil
	})

	if err := retried(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryStopsAfterMaxRetries(t *testing.T) {
	attempts := 0
	expectedErr := errors.New("boom")
	retried := Retry(2, 0, nil, func(ctx context.Context) error {
		attempts++
		return expectedErr
	})

	err := retried(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryDoesNotRetryOnContextErrors(t *testing.T) {
	for _, expectedErr := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(expectedErr.Error(), func(t *testing.T) {
			attempts := 0
			retried := Retry(5, 0, nil, func(ctx context.Context) error {
				attempts++
				return expectedErr
			})

			err := retried(context.Background())
			if !errors.Is(err, expectedErr) {
				t.Fatalf("expected %v, got %v", expectedErr, err)
			}
			if attempts != 1 {
				t.Fatalf("expected 1 attempt, got %d", attempts)
			}
		})
	}
}

func TestRetryReturnsWhenContextCanceledDuringWait(t *testing.T) {
	attempts := 0
	expectedErr := errors.New("boom")
	retried := Retry(3, 200*time.Millisecond, nil, func(ctx context.Context) error {
		attempts++
		return expectedErr
	})

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	start := time.Now()
	err := retried(ctx)
	elapsed := time.Since(start)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("expected retry wait to stop early, elapsed=%s", elapsed)
	}
}

func TestRetryAllowsNegativeMaxRetries(t *testing.T) {
	attempts := 0
	expectedErr := errors.New("boom")
	retried := Retry(-10, 0, nil, func(ctx context.Context) error {
		attempts++
		return expectedErr
	})

	err := retried(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
