package p2p

import (
	_ "embed"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

const transportScheme = "tcp"

func NewNode(selfConfig *config.PeerConfig, peersConfig []*config.PeerConfig) *Node {
	peers := make(map[string]*Peer, len(peersConfig))
	for _, p := range peersConfig {
		pc := NewPeer(p)
		peers[pc.Name] = pc
		connection.Whitelist(p.Name)
	}
	return &Node{
		Name:    selfConfig.Name,
		Address: mustBuildAddress(selfConfig),
		Peers:   peers,
	}
}

type Node struct {
	Name    string
	Address string
	// Peers stores name as a key, for fast searching.
	Peers map[string]*Peer
}

//go:embed test_context.txt
var testContent []byte

func (n *Node) Run() error {
	listener, err := net.Listen(transportScheme, n.Address)
	if err != nil {
		return err
	}
	go n.ListenForConnections(listener)

	n.Dial()

	for {
		time.Sleep(5 * time.Second)
		msg, err := protobuf.NewRequestMessage(uuid.New().String(), n.Name, protobuf.Request_CREATE, testContent)
		if err != nil {
			log.Err(err).Msg("can't create new message")
			continue
		}
		_ = n.Send(msg)
	}
}

func (n *Node) ListenForConnections(listener net.Listener) {
	for {
		nc, err := listener.Accept()
		if err != nil {
			log.Err(err).Msg("failed to accept incoming connection")
			continue
		}

		conn := connection.New()
		peerName, err := conn.Establish(nc, n.Name)
		if err != nil {
			log.Err(err).
				Stringer("remote_addr", nc.RemoteAddr()).
				Msg("failed to establish connection with peer")
			continue
		}
		n.Peers[peerName].Connect(conn)
	}
}

func (n *Node) Dial() {
	for _, p := range n.Peers {
		if p.conn.Status() == connection.StatusAlive {
			continue
		}
		nc, err := net.Dial(transportScheme, p.Address)
		if err != nil {
			p.log.Err(err).Msg("failed to dial connection")
			continue
		}

		conn := connection.New()
		if _, err = conn.Establish(nc, n.Name); err != nil {
			log.Err(err).
				Stringer("remote_addr", nc.RemoteAddr()).
				Msg("failed to connect to peer")
			return
		}
		p.Connect(conn)
	}
}

func (n *Node) Send(req *protobuf.Message) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	mErr := make(SendMultiErr)
	for _, p := range n.Peers {
		if p.conn.Status() != connection.StatusAlive {
			p.log.Warn().Msg("peer is not connected, skipping sending message")
			continue
		}

		if err = p.Send(data); err != nil {
			mErr.Append(p.Address, err)
		}
	}
	if len(mErr) > 0 {
		return mErr
	}
	return nil
}

func mustBuildAddress(cfg *config.PeerConfig) string {
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	tcpAddr, err := net.ResolveTCPAddr(transportScheme, address)
	if err != nil {
		log.Panic().
			Err(err).
			Interface("peer_config", cfg).
			Msg("failed to resolve TCP address")
	}
	return tcpAddr.String()
}
