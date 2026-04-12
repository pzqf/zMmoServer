package util

import (
	"time"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

type RetryFunc func() error

type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Retryable     func(error) bool
}

var DefaultRetryConfig = RetryConfig{
	MaxRetries:    3,
	InitialDelay:  1 * time.Second,
	MaxDelay:      10 * time.Second,
	BackoffFactor: 2.0,
	Retryable: func(err error) bool {
		return true
	},
}

func Retry(fn RetryFunc, config RetryConfig) error {
	var lastErr error
	currentDelay := config.InitialDelay

	for i := 0; i < config.MaxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		if !config.Retryable(err) {
			return err
		}

		lastErr = err
		zLog.Warn("Operation failed, retrying...",
			zap.Error(err),
			zap.Int("attempt", i+1),
			zap.Duration("delay", currentDelay))

		time.Sleep(currentDelay)

		currentDelay = time.Duration(float64(currentDelay) * config.BackoffFactor)
		if currentDelay > config.MaxDelay {
			currentDelay = config.MaxDelay
		}
	}

	return lastErr
}

func RetryWithDefault(fn RetryFunc) error {
	return Retry(fn, DefaultRetryConfig)
}

func SimpleRetry(maxAttempts int, delay time.Duration, fn func() error) error {
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxAttempts-1 {
				time.Sleep(delay)
				delay *= 2
			}
		}
	}
	return lastErr
}
