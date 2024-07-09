package utils

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type ExponentialBackoffOpts struct {
	Base       time.Duration
	Multiplier float64
	Cap        time.Duration
	Jitter     float64
}

// ExponentialBackoffWithOpts returns a BackoffFunc which implements an exponential
// backoff, capped to a provided maximum, with an additional jitter ratio.
//
// It uses the formula:
// - backoff = min(cap, base * (multiplier ^ retries))
// - backoff = backoff + (rand(0, backoff) * jitter).
func ExponentialBackoffWithOpts(opts ExponentialBackoffOpts) hcloud.BackoffFunc {
	baseSeconds := opts.Base.Seconds()
	capSeconds := opts.Cap.Seconds()

	return func(retries int) time.Duration {
		backoff := baseSeconds * math.Pow(opts.Multiplier, float64(retries)) // Exponential backoff
		backoff = math.Min(capSeconds, backoff)                              // Cap backoff
		backoff += rand.Float64() * backoff * opts.Jitter                    // #nosec G404 Add jitter

		return time.Duration(backoff * float64(time.Second))
	}
}
