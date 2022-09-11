//go:build e2e

package test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/connection"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/p2p"
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
	logging.Configure("test", &config.Logging{Level: "debug"})
	m.Run()
}

func TestHost_ReplicateSingleOperation(t *testing.T) {
	Aragorn := hostStructForName(t, AragornsName)
	defer mustClose(t, Aragorn)
	Gimli := hostStructForName(t, GimlisName)
	defer mustClose(t, Gimli)
	Legolas := hostStructForName(t, LegolasName)
	defer mustClose(t, Legolas)

	ctx := context.Background()
	Aragorn.Run(ctx)
	Gimli.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(100 * time.Millisecond)

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
	defer mustClose(t, Aragorn)
	Gimli := hostStructForName(t, GimlisName)
	defer mustClose(t, Gimli)
	Legolas := hostStructForName(t, LegolasName)
	defer mustClose(t, Legolas)

	ctx := context.Background()
	Gimli.Run(ctx)
	Aragorn.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(100 * time.Millisecond)

	send := func() {
		require.NoError(t, Aragorn.Replicate(ctx, &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere/else", Mode: 0666},
			Content:  testHomer,
		}),
			"Aragorn failed to send message")
	}

	send()

	err := Legolas.Close()
	require.NoError(t, err)
	err = Aragorn.Close()
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 10; i++ {
		Aragorn.Run(ctx)
		Legolas.Run(ctx)

		time.Sleep(50 * time.Millisecond)

		err = Aragorn.Close()
		require.NoError(t, err)
		err = Legolas.Close()
		require.NoError(t, err)
	}

	Aragorn.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(200 * time.Millisecond)

	send()
}

func TestHost_ConflictsResolving(t *testing.T) {
	Aragorn := hostStructForName(t, AragornsName)
	defer mustClose(t, Aragorn)
	Gimli := hostStructForName(t, GimlisName)
	defer mustClose(t, Gimli)
	Legolas := hostStructForName(t, LegolasName)
	defer mustClose(t, Legolas)

	ctx := context.Background()
	Gimli.Run(ctx)
	Aragorn.Run(ctx)
	Legolas.Run(ctx)

	time.Sleep(100 * time.Millisecond)

	// This is a dirty hack, but should work most of the time.
	// There's still nondeterministic behaviour sleeping here though.
	// If one of the transactions still manages to finish before the other is received anywhere
	// we might get either Gimli or Legolas get ACKs.
	sweetDreams := func() { time.Sleep(time.Second) }
	Gimli.Mirror.(*mirrorMock).consultCallback = sweetDreams
	Legolas.Mirror.(*mirrorMock).consultCallback = sweetDreams
	Aragorn.Mirror.(*mirrorMock).consultCallback = sweetDreams

	wg := sync.WaitGroup{}
	wg.Add(3)

	send := func(host *p2p.Host, shouldSucceed bool) {
		defer wg.Done()

		req := &protobuf.Request{
			Type:     protobuf.Request_CREATE,
			Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere/same", Mode: 0666},
			Clock:    host.LoadConflictsClock(),
		}
		err := host.Replicate(ctx, req)
		if shouldSucceed {
			require.NoError(t, err,
				"%s should've succeeded to replicate; conflict resolved in his favor", host.Name)
		} else {
			require.Error(t, err,
				"%s should've been rejected; conflict resolved in favor of the other peer", host.Name)
		}
	}

	// Set the clocks.
	Aragorn.SetConflictsClock(1)
	Legolas.SetConflictsClock(0)
	Gimli.SetConflictsClock(0)

	go send(Aragorn, true)
	go send(Gimli, false)
	go send(Legolas, false)

	wg.Wait()

	assert.EqualValues(t, 1, Aragorn.LoadConflictsClock())
	assert.EqualValues(t, 1, Gimli.LoadConflictsClock())
	assert.EqualValues(t, 1, Legolas.LoadConflictsClock())

	wg.Add(2)

	go send(Gimli, false)
	go send(Legolas, true)

	wg.Wait()

	assert.True(t, Legolas.Name > Gimli.Name)
	assert.EqualValues(t, 1, Aragorn.LoadConflictsClock())
	assert.EqualValues(t, 2, Gimli.LoadConflictsClock())
	assert.EqualValues(t, 1, Legolas.LoadConflictsClock())

	wg.Add(3)

	go send(Gimli, true)
	go send(Legolas, false)
	go send(Aragorn, false)

	wg.Wait()

	assert.EqualValues(t, 2, Aragorn.LoadConflictsClock())
	assert.EqualValues(t, 2, Gimli.LoadConflictsClock())
	assert.EqualValues(t, 2, Legolas.LoadConflictsClock())
}

func TestReplicate_Errors(t *testing.T) {
	Aragorn := hostStructForName(t, AragornsName)
	defer mustClose(t, Aragorn)
	Gimli := hostStructForName(t, GimlisName)
	defer mustClose(t, Gimli)
	// Not Running Legolas at all!
	// 		Legolas := hostStructForName(t, LegolasName)
	// 		defer mustClose(t, Legolas)
	// 		Legolas.Run(ctx)

	ctx := context.Background()
	Gimli.Run(ctx)
	Aragorn.Run(ctx)

	time.Sleep(100 * time.Millisecond)

	req := &protobuf.Request{
		Type:     protobuf.Request_CREATE,
		Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere", Mode: 0666},
	}
	err := Aragorn.Replicate(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, connection.ErrPeerIsDown)
}

func hostStructForName(t *testing.T, name string) *p2p.Host {
	conf := config.MustUnmarshalConfig(testConf)
	host, peers := conf.Peers.Pop(name)
	return p2p.NewHost(
		host,
		peers,
		connection.NewPool(
			host,
			peers,
			conf.Connection,
			testTLSConf(t, name)),
		&mirrorMock{},
		5*time.Second)
}

func mustClose(t *testing.T, closer io.Closer) {
	require.NoError(t, closer.Close())
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
