package connection

import (
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/rs/zerolog/log"
)

func quicConfig(handshakeTimeout time.Duration) *quic.Config {
	return &quic.Config{
		Versions:                       []quic.VersionNumber{quic.Version2},
		ConnectionIDLength:             12,
		HandshakeIdleTimeout:           handshakeTimeout,
		MaxIdleTimeout:                 5 * time.Minute,
		InitialStreamReceiveWindow:     0,
		MaxStreamReceiveWindow:         0,
		InitialConnectionReceiveWindow: 0,
		MaxConnectionReceiveWindow:     0,
		AllowConnectionWindowIncrease:  nil,
		// Doesn't allow bidirectional streams.
		MaxIncomingStreams:      -1,
		MaxIncomingUniStreams:   100,
		StatelessResetKey:       nil,
		KeepAlivePeriod:         15 * time.Second,
		DisablePathMTUDiscovery: false,
		// We're communicating between peers only, which are built with a single version.
		DisableVersionNegotiationPackets: true,
		// We don't want these, you can read more on why here: https://www.rfc-editor.org/rfc/rfc9221.html
		EnableDatagrams: false,
	}
}

func closeConn(conn quic.Connection, err error) {
	if conn == nil {
		return
	}
	// TODO provide error codes.
	if err = conn.CloseWithError(ConnErrGeneric, err.Error()); err != nil {
		log.Err(err).
			Stringer("remote_addr", conn.RemoteAddr()).
			Msg("failed to closed connection")
	}
}
