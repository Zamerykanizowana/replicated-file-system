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

//go:embed test_data/gimlis_config.json
var gimlisConf []byte

//go:embed test_data/aragorns_config.json
var aragornsConf []byte

const (
	AragornsName = "Aragorn"
	GimlisName   = "Gimli"
)

func TestMain(m *testing.M) {
	logging.Configure(&config.Logging{Level: "debug"})
	m.Run()
}

func TestPeer_ReplicateSingleOperation(t *testing.T) {
	Aragorn := peerStructForName(t, AragornsName, config.MustUnmarshalConfig(aragornsConf))
	Gimli := peerStructForName(t, GimlisName, config.MustUnmarshalConfig(gimlisConf))

	ctx := context.Background()
	Aragorn.Run(ctx)
	Gimli.Run(ctx)

	time.Sleep(1 * time.Second)

	wg := sync.WaitGroup{}
	wg.Add(2)

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

	wg.Wait()
}

func TestPeer_Reconnection(t *testing.T) {
	Aragorn := peerStructForName(t, AragornsName, config.MustUnmarshalConfig(aragornsConf))
	Gimli := peerStructForName(t, GimlisName, config.MustUnmarshalConfig(gimlisConf))

	ctx := context.Background()
	Gimli.Run(ctx)
	Aragorn.Run(ctx)

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

	Aragorn.Close()

	for i := 0; i < 10; i++ {
		Aragorn.Run(ctx)

		time.Sleep(50 * time.Millisecond)

		Aragorn.Close()
	}

	Aragorn.Run(ctx)

	time.Sleep(time.Second)

	wg.Add(1)
	send()
	wg.Wait()
}

func peerStructForName(t *testing.T, name string, conf *config.Config) *Host {
	peer, peers := conf.Peers.Pop(name)
	return &Host{
		Peer:  *peer,
		peers: peers,
		transactions: Transactions{
			ts: make(map[TransactionId]*Transaction, 2),
			mu: new(sync.Mutex),
		},
		mirror: mirrorMock{},
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
		data, err := testCerts.ReadFile("test_data/certs/" + fn + ".test")
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

type mirrorMock struct{}

func (m mirrorMock) Mirror(_ *protobuf.Request) error {
	return nil
}

func (m mirrorMock) Consult(_ *protobuf.Request) *protobuf.Response {
	return protobuf.ACK()
}
