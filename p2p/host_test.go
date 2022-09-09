package p2p

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func TestHost_Replicate(t *testing.T) {
	host := NewHost(&config.Peer{Name: "Ben"}, config.Peers{}, &mockConnection{}, &mockMirror{})
	ctx := context.Background()
	err := host.Replicate(ctx, &protobuf.Request{})
	assert.NoError(t, err)
}

type mockMirror struct{}

func (m mockMirror) Mirror(request *protobuf.Request) error {
	//TODO implement me
	panic("implement me")
}

func (m mockMirror) Consult(request *protobuf.Request) *protobuf.Response {
	//TODO implement me
	panic("implement me")
}

type mockConnection struct{}

func (m mockConnection) Close() error {
	//TODO implement me
	panic("implement me")
}

func (m mockConnection) Run(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (m mockConnection) Broadcast(ctx context.Context, data []byte) error {
	//TODO implement me
	panic("implement me")
}

func (m mockConnection) Receive() (data []byte, err error) {
	//TODO implement me
	panic("implement me")
}
