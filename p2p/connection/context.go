package connection

import (
	"context"
	"time"
)

func contextWithOptionalTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return parent, func() {} // noop cancel
	}
	return context.WithTimeout(parent, timeout)
}
