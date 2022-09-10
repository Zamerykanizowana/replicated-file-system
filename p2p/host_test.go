package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func TestHost_Replicate(t *testing.T) {
	host := NewHost(
		&config.Peer{Name: "Aragorn"},
		config.Peers{{Name: "Legolas", Address: "https://mirkwood:9011"}},
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

	t.Run("wait for responses only from peers which had received the broadcast", func(t *testing.T) {
		mErr := &connection.MultiErr{}
		legolasErr := errors.New("some error")
		mErr.Append("Legolas", legolasErr)
		conn := &mockConnection{
			receive:      make(chan []byte),
			broadcastErr: mErr,
		}
		host = NewHost(
			&config.Peer{Name: "Aragorn"},
			config.Peers{
				{Name: "Legolas", Address: "https://mirkwood:9011"},
				{Name: "Gimli", Address: "https://the-lonely-mountain:9011"},
			},
			conn,
			&mockMirror{},
			10*time.Second)
		host.Run(context.Background())

		go func() {
			time.Sleep(10 * time.Millisecond)
			// We're waiting only for Gimlis response now.
			conn.receive <- func() []byte {
				var tid string
				for tid = range host.transactions.ts {
				} // Loop once to get the existing transaction id.
				message := protobuf.NewResponseMessage(tid, "Gimli",
					&protobuf.Response{Type: protobuf.Response_ACK})
				data, err := proto.Marshal(message)
				require.NoError(t, err)
				return data
			}()
		}()

		err := host.Replicate(context.Background(), &protobuf.Request{
			Type:     protobuf.Request_MKDIR,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
		})

		require.Error(t, err)
		assert.IsType(t, &connection.MultiErr{}, err)
		sendErr := err.(*connection.MultiErr)
		require.Len(t, sendErr.Errors(), 1)
		assert.Equal(t, legolasErr, sendErr.Errors()["Legolas"])
	})
}

func TestHost_processResponses(t *testing.T) {
	host := &Host{conflicts: newConflictsResolver()}

	t.Run("aggregate response errors", func(t *testing.T) {
		responses := []*protobuf.Response{
			{Error: protobuf.Response_ERR_INVALID_MODE.Enum()},
			{Error: protobuf.Response_ERR_UNKNOWN.Enum()},
			{Error: protobuf.Response_ERR_TRANSACTION_CONFLICT.Enum()},
			{Error: protobuf.Response_ERR_INVALID_MODE.Enum()},
			{Error: protobuf.Response_ERR_INVALID_MODE.Enum()},
			{Error: protobuf.Response_ERR_UNKNOWN.Enum()},
		}
		for _, resp := range responses {
			resp.Type = protobuf.Response_NACK
		}
		err := host.processResponses(&Transaction{Responses: responses})

		require.Error(t, err)
		assert.Equal(t, ErrNotPermitted, err)
		assert.EqualValues(t, 0, host.conflicts.clock.Load())
	})

	t.Run("increment clock if the only error encountered was conflict", func(t *testing.T) {
		responses := []*protobuf.Response{
			{Error: protobuf.Response_ERR_TRANSACTION_CONFLICT.Enum()},
			{Error: protobuf.Response_ERR_TRANSACTION_CONFLICT.Enum()},
		}
		for _, resp := range responses {
			resp.Type = protobuf.Response_NACK
		}
		err := host.processResponses(&Transaction{Responses: responses})

		require.Error(t, err)
		assert.Equal(t, ErrNotPermitted, err)
		assert.EqualValues(t, 1, host.conflicts.clock.Load())
	})
}

type mockMirror struct{}

func (m mockMirror) Mirror(request *protobuf.Request) error { return nil }

func (m mockMirror) Consult(request *protobuf.Request) *protobuf.Response { return protobuf.ACK() }

type mockConnection struct {
	broadcastErr error
	receive      chan []byte
}

func (m mockConnection) Close() error { return nil }

func (m mockConnection) Run(_ context.Context) {}

func (m mockConnection) Broadcast(_ context.Context, _ []byte) error { return m.broadcastErr }

func (m mockConnection) Receive() (data []byte, err error) {
	data = <-m.receive
	return
}
