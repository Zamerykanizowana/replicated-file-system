package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func TestHost_Replicate(t *testing.T) {
	host := NewHost(
		&config.Peer{Name: "Ben"},
		config.Peers{
			{
				Name:    "Legolas",
				Address: "https://mirkwood:9011",
			},
		},
		&mockConnection{},
		&mockMirror{},
		10*time.Second)
	host.Run(context.Background())

	t.Run("return timeout when context deadline is exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := host.Replicate(ctx, &protobuf.Request{})

		assert.Error(t, err)
		assert.ErrorIs(t, context.DeadlineExceeded, err)
		assert.Empty(t, host.transactions.ts)
	})

	t.Run("return cancelled when context is exceeded", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := host.Replicate(ctx, &protobuf.Request{})

		assert.Error(t, err)
		assert.ErrorIs(t, context.Canceled, err)
		assert.Empty(t, host.transactions.ts)
	})

	t.Run("detect and resolve transaction conflict with error", func(t *testing.T) {
		// Existing transaction has higher clock.
		host.transactions.Put(&protobuf.Message{
			Tid:      "123",
			PeerName: "Legolas",
			Type: &protobuf.Message_Request{Request: &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
				Clock:    1,
			}},
		})

		err := host.Replicate(context.Background(), &protobuf.Request{
			Type:     protobuf.Request_MKDIR,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
		})
		host.transactions.Delete("123")

		assert.Error(t, err)
		assert.ErrorIs(t, ErrTransactionConflict, err)
		assert.Empty(t, host.transactions.ts)
	})
}

type mockMirror struct{}

func (m mockMirror) Mirror(request *protobuf.Request) error {
	return nil
}

func (m mockMirror) Consult(request *protobuf.Request) *protobuf.Response {
	return protobuf.ACK()
}

type mockConnection struct {
	receive chan []byte
}

func (m mockConnection) Close() error { return nil }

func (m mockConnection) Run(_ context.Context) {}

func (m mockConnection) Broadcast(_ context.Context, _ []byte) error { return nil }

func (m mockConnection) Receive() (data []byte, err error) {
	data = <-m.receive
	return
}
