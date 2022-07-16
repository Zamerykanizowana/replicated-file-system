package connection

import (
	"math/rand"
	"time"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewBackoff(conf *config.Backoff) *Backoff {
	return &Backoff{
		Backoff: conf,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Backoff is a time.Duration counter, starting at initial. After every call to
// the Next method the current timing is multiplied by factor with an added jitter.
// Backoff never exceeds max, and jitter is capped by maxJitter.
type Backoff struct {
	*config.Backoff
	// attempt is stored as float64 instead of an uint to avoid type conversion for math.Pow.
	attempt          float64
	previousDuration time.Duration
	rand             *rand.Rand
	timer            *time.Timer
}

// Next blocks on the timer with the current Backoff duration.
func (b *Backoff) Next() {
	defer func() { b.attempt++ }()

	switch b.attempt {
	case 0:
		b.timer = time.NewTimer(b.Initial)
	default:
		dur := b.next()
		b.previousDuration = dur
		b.timer.Reset(dur)
	}

	<-b.timer.C
}

func (b *Backoff) next() time.Duration {
	// Fast path when max was already reached.
	if b.previousDuration > b.Max {
		return b.Max
	}

	df := b.previousDuration.Seconds()

	switch b.MaxFactorJitter {
	case 0:
		df *= b.Factor
	default:
		jt := b.rand.Intn(int(b.MaxFactorJitter * 100))
		// Single bit shift and modulo to decide which sign we choose.
		// Reason for this is we want to avoid calling rand twice.
		// MaxFactorJitter should be applied with either sign (+ -) with somewhat equal probability.
		if jt>>1%2 == 0 {
			jt = -jt
		}
		df *= b.Factor + (b.Factor * float64(jt))
	}

	dur := time.Duration(df)
	if dur < b.Initial {
		return b.Initial
	}
	if dur > b.Max {
		return b.Max
	}
	return dur
}

// Reset restarts the current attempt counter at zero.
func (b *Backoff) Reset() {
	b.attempt = 0
	b.previousDuration = b.Initial
	if b.timer != nil {
		if stop := b.timer.Stop(); !stop {
			<-b.timer.C // Drain the channel.
		}
	}
}
