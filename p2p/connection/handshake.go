package connection

import (
	"net"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func handshake(conn net.Conn, self string) (peerName string, err error) {
	h := &protobuf.Handshake{PeerName: self}

	// Send handshake.
	data, err := proto.Marshal(h)
	if err != nil {
		return "", errors.Wrap(err, "failed to proto marshal handshake")
	}
	if err = send(conn, data); err != nil {
		return "", errors.Wrap(err, "failed to send handshake")
	}

	// Receive handshake.
	data, err = recv(conn)
	if err != nil {
		return "", errors.Wrap(err, "failed to read handshake header")
	}
	if err = proto.Unmarshal(data, h); err != nil {
		return "", errors.Errorf("invalid handshake received: %v", data)
	}

	// Auth validation.
	if _, whitelisted := whitelist[h.PeerName]; !whitelisted {
		return h.PeerName, errors.Errorf("remote peer: %s is not whitelisted", h.PeerName)
	}
	return h.PeerName, nil
}
