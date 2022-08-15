package connection

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/lucas-clemente/quic-go"
)

type quicDial = func(
	ctx context.Context,
	addr string,
	tlsConf *tls.Config,
	config *quic.Config,
) (quic.Connection, error)

func quicConfig(handshakeTimeout time.Duration) *quic.Config {
	conf := &quic.Config{
		Versions:                       []quic.VersionNumber{quic.Version2},
		ConnectionIDLength:             12,
		HandshakeIdleTimeout:           handshakeTimeout,
		MaxIdleTimeout:                 5 * time.Minute,
		InitialStreamReceiveWindow:     (1 << 10) * 512,       // 512 Kb
		MaxStreamReceiveWindow:         (1 << 20) * 6,         // 6 Mb
		InitialConnectionReceiveWindow: (1 << 10) * 512 * 1.5, // 768 Kb
		MaxConnectionReceiveWindow:     (1 << 20) * 15,        // 15 Mb
		MaxIncomingStreams:             -1,                    // Doesn't allow bidirectional streams.
		MaxIncomingUniStreams:          100,
		StatelessResetKey:              nil,
		KeepAlivePeriod:                15 * time.Second,
		DisablePathMTUDiscovery:        false,
		// We're communicating between peers only, which are built with a single version.
		DisableVersionNegotiationPackets: true,
		// We don't want these, you can read more on why here: https://www.rfc-editor.org/rfc/rfc9221.html
		EnableDatagrams: false,
	}
	return conf
}
