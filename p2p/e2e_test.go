//go:build e2e

package p2p

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

//go:embed test_data/shakespeare.txt
var testShakespeare []byte

//go:embed test_data/homer.txt
var testHomer []byte

//go:embed test_data/open_source.png
var testImage []byte

//go:embed test_data/config.json
var testConf []byte

const (
	AragornsName = "Aragorn"
	GimlisName   = "Gimli"
	LegolasName  = "Legolas"
)

func TestMain(m *testing.M) {
	logging.Configure(&config.Logging{Level: "debug"})
	m.Run()
}

func TestHost_ReplicateSingleOperation(t *testing.T) {
	Aragorn := hostStructForName(t, AragornsName)
	Gimli := hostStructForName(t, GimlisName)
	Legolas := hostStructForName(t, LegolasName)

	ctx := context.Background()
	Aragorn.Run(ctx)
	Gimli.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(1 * time.Second)

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		require.NoError(t, Aragorn.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere/else", Mode: 0666},
			Content:  testHomer,
		}),
			"Aragorn failed to send message")
	}()

	go func() {
		defer wg.Done()
		require.NoError(t, Gimli.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere", Mode: 0666},
			Content:  testShakespeare,
		}),
			"Gimli failed to send message")
	}()

	go func() {
		defer wg.Done()
		require.NoError(t, Legolas.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/in-the-forest", Mode: 0666},
			Content:  testImage,
		}),
			"Gimli failed to send message")
	}()

	wg.Wait()
}

func TestHost_Reconnection(t *testing.T) {
	Aragorn := hostStructForName(t, AragornsName)
	Gimli := hostStructForName(t, GimlisName)
	Legolas := hostStructForName(t, LegolasName)

	ctx := context.Background()
	Gimli.Run(ctx)
	Aragorn.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(time.Second)

	wg := sync.WaitGroup{}
	send := func() {
		defer wg.Done()
		require.NoError(t, Aragorn.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere/else", Mode: 0666},
			Content:  testHomer,
		}),
			"Aragorn failed to send message")
	}

	wg.Add(1)
	send()
	wg.Wait()

	_ = Legolas.Close()
	_ = Aragorn.Close()

	for i := 0; i < 10; i++ {
		Aragorn.Run(ctx)
		Legolas.Run(ctx)

		time.Sleep(50 * time.Millisecond)

		_ = Aragorn.Close()
		_ = Legolas.Close()
	}

	Aragorn.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(time.Second)

	wg.Add(1)
	send()
	wg.Wait()
}

func TestHost_ConflictsResolving(t *testing.T) {
	Aragorn := hostStructForName(t, AragornsName)
	Gimli := hostStructForName(t, GimlisName)
	Legolas := hostStructForName(t, LegolasName)

	ctx := context.Background()
	Gimli.Run(ctx)
	Aragorn.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(1 * time.Second)

	// This is a dirty hack, but should work most of the time.
	// There's still nondeterministic behaviour sleeping here though.
	// If one of the transactions still manages to finish before the other is received anywhere
	// we might get either Gimli or Legolas get ACKs.
	sweetDreams := func() { time.Sleep(time.Second) }
	Gimli.mirror.(*mirrorMock).consultCallback = sweetDreams
	Legolas.mirror.(*mirrorMock).consultCallback = sweetDreams
	Aragorn.mirror.(*mirrorMock).consultCallback = sweetDreams

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		require.NoError(t, Aragorn.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere/same", Mode: 0666},
			Content:  testHomer,
			Clock:    0,
		}),
			"Aragorn should've succeeded to replicate; conflict resolved in his favor")
	}()

	go func() {
		defer wg.Done()
		require.Error(t, Gimli.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere/same", Mode: 0666},
			Content:  testShakespeare,
			Clock:    1,
		}),
			"Gimli should've been rejected; conflict resolved in favor of Aragorn")
	}()

	go func() {
		defer wg.Done()
		require.Error(t, Legolas.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere/same", Mode: 0666},
			Content:  testImage,
			Clock:    1,
		}),
			"Legolas should've been rejected; conflict resolved in favor of Aragorn")
	}()

	wg.Wait()
}

func hostStructForName(t *testing.T, name string) *Host {
	conf := config.MustUnmarshalConfig(testConf)
	peer, peers := conf.Peers.Pop(name)
	return &Host{
		Peer:         *peer,
		peers:        peers,
		transactions: newTransactions(),
		mirror:       &mirrorMock{},
		conflicts:    newConflictsResolver(),
		connPool: connection.NewPool(
			peer,
			peers,
			&conf.Connection,
			testTLSConf(t, name)),
	}
}

//go:embed test_data/certs
var testCerts embed.FS

func testTLSConf(t *testing.T, peer string) *tls.Config {
	t.Helper()
	mustOpen := func(fn string) []byte {
		data, err := testCerts.ReadFile("test_data/certs/" + fn)
		require.NoError(t, err)
		return data
	}
	peer = strings.ToLower(peer)

	cert, err := tls.X509KeyPair(mustOpen(peer+".crt"), mustOpen(peer+".key"))
	require.NoError(t, err)

	pool := x509.NewCertPool()
	appended := pool.AppendCertsFromPEM(mustOpen("ca.crt"))
	require.True(t, appended)

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            pool,
		ClientCAs:          pool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
	}
}

type mirrorMock struct {
	consultCallback func()
}

func (m mirrorMock) Mirror(_ *protobuf.Request) error {
	return nil
}

func (m mirrorMock) Consult(_ *protobuf.Request) *protobuf.Response {
	if m.consultCallback != nil {
		m.consultCallback()
	}
	return protobuf.ACK()
}
