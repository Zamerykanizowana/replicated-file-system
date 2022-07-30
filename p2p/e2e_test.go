//go:build e2e
// +build e2e

package p2p

import (
	_ "embed"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

//go:embed test/shakespeare.txt
var testShakespeare []byte

//go:embed test/homer.txt
var testHomer []byte

func TestPeer(t *testing.T) {
	conf := config.Default()
	limitedPeers := make([]*config.Peer, 0, 2)
	for _, p := range conf.Peers {
		if p.Name == "Aragorn" || p.Name == "Gimli" {
			limitedPeers = append(limitedPeers, p)
		}
	}
	conf.Peers = limitedPeers

	aragorn := NewPeer("Aragorn", conf.Peers, &conf.Connection)
	gimli := NewPeer("Gimli", conf.Peers, &conf.Connection)

	aragorn.Run()
	gimli.Run()

	wg := sync.WaitGroup{}
	wg.Add(4)

	gimliRequest, err := protobuf.NewRequestMessage(
		uuid.New().String(),
		"Aragorn",
		protobuf.Request_CREATE,
		testShakespeare)
	require.NoError(t, err)

	aragornRequest, err := protobuf.NewRequestMessage(
		uuid.New().String(),
		"Gimli",
		protobuf.Request_CREATE,
		testHomer)
	require.NoError(t, err)

	go func() {
		defer wg.Done()
		msg, err := aragorn.Receive()
		assert.NoError(t, err, "Aragorn failed to receive message")

		// We want to compare only the decompressed content with what was sent.
		content := msg.GetRequest().GetContent()
		msg.GetRequest().Content = nil
		gimliRequest.GetRequest().Content = nil

		assert.Equal(t, gimliRequest, msg)
		assert.Equal(t, testShakespeare, content)
	}()

	go func() {
		defer wg.Done()
		msg, err := gimli.Receive()
		assert.NoError(t, err, "Gimli failed to receive message")

		// We want to compare only the decompressed content with what was sent.
		content := msg.GetRequest().GetContent()
		msg.GetRequest().Content = nil
		aragornRequest.GetRequest().Content = nil

		assert.Equal(t, aragornRequest, msg)
		assert.Equal(t, testHomer, content)
	}()

	go func() {
		defer wg.Done()
		require.NoError(t, aragorn.Send(gimliRequest), "Aragorn failed to send message")
	}()

	go func() {
		defer wg.Done()
		require.NoError(t, gimli.Send(aragornRequest), "Gimli failed to send message")
	}()

	wg.Wait()
}
