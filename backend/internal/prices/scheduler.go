package prices

import (
	"context"
	"time"
)

type Scheduler struct {
	interval time.Duration
	fn       func(context.Context) error
}

func NewScheduler(interval time.Duration, fn func(context.Context) error) *Scheduler {
	return &Scheduler{interval: interval, fn: fn}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	if err := s.fn(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.fn(ctx); err != nil {
				return err
			}
		}
	}
}
