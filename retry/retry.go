package retry

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/slok/goresilience"
	runnerutils "github.com/slok/goresilience/internal/util/runner"
)

// Config is the configuration used for the retry Runner.
type Config struct {
	// WaitBase is the base unit duration to wait on the retries.
	WaitBase time.Duration
	// Backoff enables exponential backoff on the retry (also disables jitter).
	DisableBackoff bool
	// Times is the number of times that will be retried in case of error
	// before returning the error itself.
	Times int
}

func (c *Config) defaults() {
	if c.WaitBase <= 0 {
		c.WaitBase = 20 * time.Millisecond
	}

	// TODO: Allow -1 for forever retries?
	if c.Times <= 0 {
		c.Times = 3
	}
}

// New returns a new retry execution, the execution will be retried the number
// of times specificed on the config (+1, the original execution that is not a retry).
func New(cfg Config, r goresilience.Runner) goresilience.Runner {
	cfg.defaults()

	r = runnerutils.Sanitize(r)

	// Use the algorithms for jitter and backoff.
	// https://aws.amazon.com/es/blogs/architecture/exponential-backoff-and-jitter/
	return goresilience.RunnerFunc(func(ctx context.Context, f goresilience.Func) error {
		var err error

		// Start the attemps. (it's 1 + the number of retries.)
		for i := 0; i <= cfg.Times; i++ {
			err = r.Run(ctx, f)
			if err == nil {
				return nil
			}

			// We need to sleep before making a retry.
			waitDuration := cfg.WaitBase

			// Apply Backoff.
			// The backoff is calculated exponentially based on a base time
			// and the attemp of the retry.
			if !cfg.DisableBackoff {
				exp := math.Exp2(float64(i + 1))
				waitDuration = time.Duration(float64(cfg.WaitBase) * exp)
				// Round to millisecs.
				waitDuration = waitDuration.Round(time.Millisecond)

				// Apply "full jitter".
				random := rand.New(rand.NewSource(time.Now().UnixNano()))
				waitDuration = time.Duration(float64(waitDuration) * random.Float64())
			}

			time.Sleep(waitDuration)
		}

		return err
	})
}
