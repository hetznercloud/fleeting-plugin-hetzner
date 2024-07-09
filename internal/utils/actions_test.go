package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExponentialBackoff(t *testing.T) {
	backoffFunc := ExponentialBackoffWithOpts(ExponentialBackoffOpts{
		Base:       time.Second,
		Multiplier: 2.0,
		Cap:        5 * time.Second,
		Jitter:     0.0, // Turning off jitter for testing
	})

	count := 25
	sum := 0.0
	result := make([]string, 0, count)
	for i := 0; i < count; i++ {
		backoff := backoffFunc(i)
		sum += backoff.Seconds()
		result = append(result, backoff.String())
	}

	require.Equal(t, []string{"1s", "2s", "4s", "5s", "5s"}, result[:5])
	require.Equal(t, 117.0, sum)
}
