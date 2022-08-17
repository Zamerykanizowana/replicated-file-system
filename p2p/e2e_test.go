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
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/logging"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection/tlsconf"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

//go:embed test_data/shakespeare.txt
var testShakespeare []byte

//go:embed test_data/homer.txt
var testHomer []byte

//go:embed test_data/config.json
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

	Aragorn := &Peer{
		Peer:  *aragornsConf,
		peers: []*config.Peer{gimlisConf},
		transactions: Transactions{
			ts: make(map[TransactionId]*Transaction, 2),
			mu: new(sync.Mutex),
		},
		connPool: connection.NewPool(
			aragornsConf,
			[]*config.Peer{gimlisConf},
			&conf.Connection,
			testTLSConf(t, "Aragorn")),
	}
	Gimli := &Peer{
		Peer:  *gimlisConf,
		peers: []*config.Peer{aragornsConf},
		transactions: Transactions{
			ts: make(map[TransactionId]*Transaction, 2),
			mu: new(sync.Mutex),
		},
		connPool: connection.NewPool(
			gimlisConf,
			[]*config.Peer{aragornsConf},
			&conf.Connection,
			testTLSConf(t, "Gimli")),
	}

	Aragorn.Run()
	Gimli.Run()

	time.Sleep(1 * time.Second)

	wg := sync.WaitGroup{}
	wg.Add(2)

	// TODO FIX THE TEST!
	go func() {
		defer wg.Done()
		require.NoError(t, Aragorn.Replicate(protobuf.Request_CREATE, nil, testHomer),
			"Aragorn failed to send message")
	}()

	// TODO FIX THE TEST!
	go func() {
		defer wg.Done()
		require.NoError(t, Gimli.Replicate(protobuf.Request_CREATE, nil, testShakespeare),
			"Gimli failed to send message")
	}()

	wg.Wait()
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

	tc := tlsconf.Default(tls.VersionTLS13)
	tc.RootCAs = pool
	tc.ClientCAs = pool
	tc.Certificates = []tls.Certificate{cert}

	return tc
}
