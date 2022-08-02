package p2p

import (
	_ "embed"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/tlsconf"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func NewPeer(
	name string,
	peersConfig []*config.Peer,
	connConfig *config.Connection,
) *Peer {
	var self config.Peer
	peers := make([]*config.Peer, 0, len(peersConfig)-1)
	for _, p := range peersConfig {
		if name == p.Name {
			self = *p
			continue
		}
		peers = append(peers, p)
	}
	if len(self.Name) == 0 {
		log.Fatal().
			Str("name", name).
			Interface("peers_config", peersConfig).
			Msg("invalid peer name provided, peer must be listed in the peers config")
	}
	return &Peer{
		Peer:     self,
		connPool: connection.NewPool(&self, connConfig, peers, tlsconf.Default(connConfig.GetTLSVersion())),
	}
}

// Peer represents a single peer we're running in the p2p network.
type Peer struct {
	config.Peer
	connPool *connection.Pool
}

// Run kicks of connection processes for the Peer.
func (p *Peer) Run() {
	log.Info().Object("peer", p).Msg("initializing p2p network connection")
	p.connPool.Run()
}

// Broadcast sends the protobuf.Message to all the other peers in the network.
func (p *Peer) Broadcast(msg *protobuf.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal protobuf message")
	}
	if err = p.connPool.Broadcast(data); err != nil {
		return errors.Wrap(err, "failed to send the message to some of the peers")
	}
	return nil
}

// Receive receives a single protobuf.Message from the network.
// It blocks until the message is received.
func (p *Peer) Receive() (*protobuf.Message, error) {
	data, err := p.connPool.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive message from peer")
	}
	return protobuf.ReadMessage(data)
}
