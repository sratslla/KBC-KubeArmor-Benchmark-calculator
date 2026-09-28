// SPDX-License-Identifier: Apache-2.0
// Copyright 2024-2026 Authors of KBC / KubeArmor

package waitutil

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Condition is a polling predicate. Return true when done.
type Condition func(ctx context.Context) (done bool, err error)

// Until polls cond until it returns true, ctx is cancelled, or timeout elapses.
func Until(ctx context.Context, interval, timeout time.Duration, cond Condition) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return wait.PollUntilContextCancel(ctx, interval, true, wait.ConditionWithContextFunc(cond))
}

// StableUntil requires cond to succeed stableTimes consecutive times.
func StableUntil(ctx context.Context, interval, timeout time.Duration, stableTimes int, cond Condition) error {
	if stableTimes < 1 {
		stableTimes = 1
	}
	streak := 0
	return Until(ctx, interval, timeout, func(ctx context.Context) (bool, error) {
		ok, err := cond(ctx)
		if err != nil {
			streak = 0
			return false, err
		}
		if !ok {
			streak = 0
			return false, nil
		}
		streak++
		return streak >= stableTimes, nil
	})
}

// Soak sleeps for d unless ctx is cancelled earlier.
func Soak(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("soak interrupted: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
