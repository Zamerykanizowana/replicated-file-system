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
			receive:      make(chan connection.Message),
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
			conn.receive <- func() connection.Message {
				var tid string
				for tid = range host.transactions.ts {
				} // Loop once to get the existing transaction id.
				return newMessage(t, tid, "Gimli", &protobuf.Response{Type: protobuf.Response_ACK})
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
		err := host.processResponses(&transaction{Responses: responses})

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
		err := host.processResponses(&transaction{Responses: responses})

		require.Error(t, err)
		assert.Equal(t, ErrNotPermitted, err)
		assert.EqualValues(t, 1, host.conflicts.clock.Load())
	})
}

func TestHost_handleTransaction(t *testing.T) {
	setup := func() (host *Host, conn *mockConnection, mir *mockMirror) {
		conn = &mockConnection{receive: make(chan connection.Message)}
		mir = &mockMirror{}
		host = NewHost(
			&config.Peer{Name: "Aragorn"},
			config.Peers{
				{Name: "Legolas", Address: "https://mirkwood:9011"},
				{Name: "Gimli", Address: "https://the-lonely-mountain:9011"},
			},
			conn, mir, 10*time.Second)
		return
	}

	t.Run("golden path, everything goes smooth", func(t *testing.T) {
		host, conn, mir := setup()
		host.Run(context.Background())

		conn.receive <- func() connection.Message {
			return newMessage(t, "123", "Gimli", &protobuf.Response{Type: protobuf.Response_ACK})
		}()

		conn.receive <- func() connection.Message {
			return newMessage(t, "123", "Legolas", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"}})
		}()

		// This will make sure all active transactions are done.
		// We're also testing Close here :)
		_ = host.Close()

		assert.Len(t, host.transactions.ts, 0)
		require.Len(t, mir.calledWithRequests, 1)
		assert.Equal(t, "/somewhere", mir.calledWithRequests[0].Metadata.RelativePath)
		assert.Equal(t, protobuf.Request_CREATE, mir.calledWithRequests[0].Type)
	})

	t.Run("response is consulted and NACK", func(t *testing.T) {
		host, conn, mir := setup()
		mir.consultResponse = protobuf.NACK(protobuf.Response_ERR_ALREADY_EXISTS, errors.New("test"))
		defer func() { mir.consultResponse = nil }()
		host.Run(context.Background())

		conn.receive <- func() connection.Message {
			return newMessage(t, "123", "Legolas", &protobuf.Response{Type: protobuf.Response_ACK})
		}()

		conn.receive <- func() connection.Message {
			return newMessage(t, "123", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"}})
		}()

		_ = host.Close()

		assert.Len(t, host.transactions.ts, 0)
		require.Len(t, mir.consultedRequests, 1)
		assert.Equal(t, "/somewhere", mir.consultedRequests[0].Metadata.RelativePath)
		assert.Equal(t, protobuf.Request_CREATE, mir.consultedRequests[0].Type)
		assert.Len(t, mir.calledWithRequests, 0)
	})

	t.Run("response is consulted and ACK, but conflict is detected", func(t *testing.T) {
		host, conn, mir := setup()
		host.Run(context.Background())

		conn.receive <- func() connection.Message {
			return newMessage(t, "321", "Legolas", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
				Clock:    1})
		}()

		conn.receive <- func() connection.Message {
			return newMessage(t, "123", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_MKDIR,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
				Clock:    0})
		}()

		conn.receive <- func() connection.Message {
			return newMessage(t, "123", "Legolas", &protobuf.Response{Type: protobuf.Response_ACK})
		}()

		conn.receive <- func() connection.Message {
			return newMessage(t, "321", "Gimli", &protobuf.Response{Type: protobuf.Response_ACK})
		}()

		_ = host.Close()

		assert.Len(t, host.transactions.ts, 0)
		require.Len(t, mir.consultedRequests, 2)
		assert.Len(t, mir.calledWithRequests, 1)
		// Legolas wins conflict!
		assert.Equal(t, protobuf.Request_CREATE, mir.calledWithRequests[0].Type)
	})
}

func newMessage(t *testing.T, tid, peerName string, typ interface{}) connection.Message {
	var (
		pm  *protobuf.Message
		err error
	)
	switch v := typ.(type) {
	case *protobuf.Request:
		pm, err = protobuf.NewRequestMessage(tid, peerName, v)
		require.NoError(t, err)
	case *protobuf.Response:
		pm = protobuf.NewResponseMessage(tid, peerName, v)
	}
	data, err := proto.Marshal(pm)
	require.NoError(t, err)
	return connection.Message{Data: data}
}

type mockMirror struct {
	calledWithRequests []*protobuf.Request
	consultedRequests  []*protobuf.Request
	consultResponse    *protobuf.Response
}

func (m *mockMirror) Mirror(request *protobuf.Request) error {
	m.calledWithRequests = append(m.calledWithRequests, request)
	return nil
}

func (m *mockMirror) Consult(request *protobuf.Request) *protobuf.Response {
	m.consultedRequests = append(m.consultedRequests, request)
	if m.consultResponse == nil {
		return protobuf.ACK()
	}
	return m.consultResponse
}

type mockConnection struct {
	broadcastErr error
	receive      chan connection.Message
}

func (m mockConnection) Close() error { return nil }

func (m mockConnection) Run(_ context.Context) {}

func (m mockConnection) Broadcast(_ context.Context, _ []byte) error { return m.broadcastErr }

func (m mockConnection) ReceiveC() <-chan connection.Message { return m.receive }
