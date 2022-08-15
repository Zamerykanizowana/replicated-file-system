//go:build e2e
// +build e2e

package p2p

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/tlsconf"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

//go:embed test/shakespeare.txt
var testShakespeare []byte

//go:embed test/homer.txt
var testHomer []byte

//go:embed test/config.json
var testConf []byte

func TestPeer(t *testing.T) {
	logging.Configure(&config.Logging{Level: "debug"})
	conf := config.MustUnmarshalConfig(testConf)
	var (
		aragornsConf *config.Peer
		gimlisConf   *config.Peer
	)
	for _, p := range conf.Peers {
		switch p.Name {
		case "Aragorn":
			aragornsConf = p
		case "Gimli":
			gimlisConf = p
		}
	}

	aragorn := &Peer{
		Peer: *aragornsConf,
		connPool: connection.NewPool(
			aragornsConf,
			&conf.Connection,
			[]*config.Peer{gimlisConf},
			testTLSConf(t, "Aragorn")),
	}
	gimli := &Peer{
		Peer: *gimlisConf,
		connPool: connection.NewPool(
			gimlisConf,
			&conf.Connection,
			[]*config.Peer{aragornsConf},
			testTLSConf(t, "Gimli")),
	}

	aragorn.Run()
	gimli.Run()

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

	wg := sync.WaitGroup{}
	wg.Add(4)

	go func() {
		defer wg.Done()
		msg, err := aragorn.Receive()
		require.NoError(t, err, "Aragorn failed to receive message")

		// We want to compare only the decompressed content with what was sent.
		content := msg.GetRequest().GetContent()
		msg.GetRequest().Content = nil
		aragornRequest.GetRequest().Content = nil

		testCompareMessages(t, aragornRequest, msg)
		assert.Equal(t, testHomer, content)
	}()

	go func() {
		defer wg.Done()
		msg, err := gimli.Receive()
		require.NoError(t, err, "Gimli failed to receive message")

		// We want to compare only the decompressed content with what was sent.
		content := msg.GetRequest().GetContent()
		msg.GetRequest().Content = nil
		gimliRequest.GetRequest().Content = nil

		testCompareMessages(t, gimliRequest, msg)
		assert.Equal(t, testShakespeare, content)
	}()

	go func() {
		defer wg.Done()
		require.NoError(t, aragorn.Broadcast(gimliRequest), "Aragorn failed to send message")
	}()

	go func() {
		defer wg.Done()
		require.NoError(t, gimli.Broadcast(aragornRequest), "Gimli failed to send message")
	}()

	wg.Wait()
}

//go:embed test/certs
var testCerts embed.FS

func testTLSConf(t *testing.T, peer string) *tls.Config {
	t.Helper()
	mustOpen := func(fn string) []byte {
		data, err := testCerts.ReadFile("test/certs/" + fn + ".test")
		require.NoError(t, err)
		return data
	}
	peer = strings.ToLower(peer)

	cert, err := tls.X509KeyPair(mustOpen(peer+".crt"), mustOpen(peer+".key"))
	require.NoError(t, err)

	pool := x509.NewCertPool()
	appended := pool.AppendCertsFromPEM(mustOpen("ca.crt"))
	require.True(t, appended)

	tc := tlsconf.Default(tls.VersionTLS13)
	tc.RootCAs = pool
	tc.ClientCAs = pool
	tc.Certificates = []tls.Certificate{cert}

	return tc
}

func testCompareMessages(t *testing.T, expected, actual *protobuf.Message) {
	t.Helper()
	assert.Equal(t, expected.PeerName, actual.PeerName)
	assert.Equal(t, expected.Tid, actual.Tid)
	switch actual.Type.(type) {
	case *protobuf.Message_Request:
		assert.Equal(t, expected.GetRequest().Type, actual.GetRequest().Type)
	case *protobuf.Message_Response:
		panic("implement me!")
	}
}
