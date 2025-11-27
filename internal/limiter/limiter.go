package limiter

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type Limiter struct {
	backoffAfter int
	backoffFunc  hcloud.BackoffFunc

	counterMapLock      sync.Mutex
	counterMap          map[string]int
	counterUpdatedAtMap map[string]time.Time
}

type Opts struct {
	// Number of attempts after which the backoff function starts.
	BackoffAfter int
	// Returns a sleep duration based on the number of attempts.
	BackoffFunc hcloud.BackoffFunc
}

func New(opts Opts) *Limiter {
	return &Limiter{
		backoffAfter: opts.BackoffAfter,
		backoffFunc:  opts.BackoffFunc,

		counterMapLock:      sync.Mutex{},
		counterMap:          make(map[string]int),
		counterUpdatedAtMap: make(map[string]time.Time),
	}
}

func (l *Limiter) updateCounter(id string, increase bool) {
	if increase {
		l.counterMap[id]++
	} else {
		if l.counterMap[id] > l.backoffAfter {
			l.counterMap[id] = l.backoffAfter - 1
		} else {
			l.counterMap[id] = max(0, l.counterMap[id]-1)
		}
	}

	l.counterUpdatedAtMap[id] = time.Now()
}

func (l *Limiter) backoff(id string) time.Duration {
	l.counterMapLock.Lock()
	defer l.counterMapLock.Unlock()

	if l.counterMap[id] < l.backoffAfter {
		return time.Duration(0)
	}

	// Reset count if it was not updated in the past hour.
	if time.Since(l.counterUpdatedAtMap[id]) > time.Hour {
		l.updateCounter(id, false)
	}

	// Start at the bottom of the exponential curve.
	return l.backoffFunc(max(l.counterMap[id]-l.backoffAfter, 0))
}

func (l *Limiter) update(id string, increase bool) {
	l.counterMapLock.Lock()
	defer l.counterMapLock.Unlock()

	l.updateCounter(id, increase)
}

func (l *Limiter) Operation(id string) *Operation {
	return &Operation{
		limiter: l,
		id:      id,
	}
}

type Operation struct {
	id      string
	limiter *Limiter
}

func (o *Operation) Backoff() time.Duration {
	return o.limiter.backoff(o.id)
}

func (o *Operation) Sleep(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
	}
	return nil
}

func (o *Operation) Limit(ctx context.Context, logger hclog.Logger) error {
	if duration := o.Backoff(); duration > 0 {
		logger.Warn("too many failures, limiting request rate", "operation", o.id, "duration", duration.String())

		if err := o.Sleep(ctx, duration); err != nil {
			return err
		}
	}
	return nil
}

func (o *Operation) Increase(increase bool) {
	o.limiter.update(o.id, increase)
}
