package p2p

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func NewPeer(cfg *config.PeerConfig) *Peer {
	p := &Peer{
		Address: mustBuildAddress(cfg),
		Name:    cfg.Name,
		conn:    connection.New(),
	}
	p.log = log.With().EmbedObject(p).Logger()
	return p
}

type (
	Peer struct {
		Address string
		Name    string
		log     zerolog.Logger
		conn    *connection.Connection
	}
)

func (p *Peer) MarshalZerologObject(e *zerolog.Event) {
	e.Dict("peer", zerolog.Dict().
		Str("address", p.Address).
		Str("name", p.Name).
		Stringer("status", p.conn.Status()))
}

func (p *Peer) Connect(conn *connection.Connection) {
	if conn.Status() == connection.StatusDead {
		return
	}
	if p.conn.Status() == connection.StatusAlive {
		conn.Close()
		return
	}
	p.conn = conn
	go p.Listen()
}

func (p *Peer) Listen() {
	for {
		if p.conn.Status() == connection.StatusDead {
			return
		}
		data, err := p.conn.Recv()
		if err == connection.ErrClosed {
			p.log.Err(err).Msg("connection to peer was closed")
			return
		}
		if err != nil {
			p.log.Err(err).Msg("failed to receive message")
			continue
		}
		msg, err := protobuf.ReadMessage(data)
		if err != nil {
			log.Err(err).
				Bytes("raw_request", data).
				Msg("failed to read Message")
			continue
		}

		log.Info().
			Stringer("message", msg).
			Msg("received message")
	}
}

// Send accepts data instead of protobuf.Request to avoid marshalling the request for each Peer.
func (p *Peer) Send(data []byte) error {
	return p.conn.Send(data)
}
