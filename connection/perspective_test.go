package connection

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerspectiveResolver_Resolve(t *testing.T) {
	t.Run("return no error if resolved", func(t *testing.T) {
		resolver := &perspectiveResolver{
			recv: func(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error) {
				resolvent := perspectiveResolvent{Perspective: Client, ctr: 1}
				return resolvent.Encode(), nil
			},
			send: func(ctx context.Context, op uniStreamOpener, data []byte) error {
				return nil
			},
		}

		err := resolver.Resolve(context.Background(), Server, nil, func() bool { return false })
		require.NoError(t, err)
	})

	t.Run("return no error if fallback applies", func(t *testing.T) {
		resolver := &perspectiveResolver{
			recv: func(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error) {
				resolvent := perspectiveResolvent{Perspective: Client, ctr: 2}
				return resolvent.Encode(), nil
			},
			send: func(ctx context.Context, op uniStreamOpener, data []byte) error {
				return nil
			},
		}

		err := resolver.Resolve(context.Background(), Server, nil, func() bool { return true })
		require.NoError(t, err)
	})

	t.Run("return error if fallback doesn't apply", func(t *testing.T) {
		resolver := &perspectiveResolver{
			recv: func(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error) {
				resolvent := perspectiveResolvent{Perspective: Client, ctr: 2}
				return resolvent.Encode(), nil
			},
			send: func(ctx context.Context, op uniStreamOpener, data []byte) error {
				return nil
			},
		}
		resolver.counter.Store(1)

		err := resolver.Resolve(context.Background(), Server, nil, func() bool { return false })
		require.Error(t, err)
		assert.ErrorIs(t, err, connErrNotResolved)
	})

	t.Run("mismatched counters, perspective is not resolved", func(t *testing.T) {
		resolver := &perspectiveResolver{
			recv: func(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error) {
				resolvent := perspectiveResolvent{Perspective: Client, ctr: 2}
				return resolvent.Encode(), nil
			},
			send: func(ctx context.Context, op uniStreamOpener, data []byte) error {
				return nil
			},
		}
		resolver.counter.Store(0)

		err := resolver.Resolve(context.Background(), Server, nil, func() bool { return false })
		require.Error(t, err)
		assert.ErrorIs(t, err, connErrNotResolved)
	})

	t.Run("perspectives differ, return error", func(t *testing.T) {
		resolver := &perspectiveResolver{
			recv: func(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error) {
				resolvent := perspectiveResolvent{Perspective: Client, ctr: 1}
				return resolvent.Encode(), nil
			},
			send: func(ctx context.Context, op uniStreamOpener, data []byte) error {
				return nil
			},
		}
		err := resolver.Resolve(context.Background(), Client, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "BUG")
	})

	t.Run("return receiving error", func(t *testing.T) {
		recvErr := errors.New("receive error")
		resolver := &perspectiveResolver{
			recv: func(ctx context.Context, ac uniStreamAcceptor, timeout time.Duration) ([]byte, error) {
				return nil, recvErr
			},
			send: func(ctx context.Context, op uniStreamOpener, data []byte) error {
				return nil
			},
		}
		err := resolver.Resolve(context.Background(), Client, nil, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, recvErr)
	})
}
