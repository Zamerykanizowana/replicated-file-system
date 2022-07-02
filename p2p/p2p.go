package p2p

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/bus"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

const transportScheme = "tcp"

func NewPeer(selfConfig *config.PeerConfig, peersConfig []*config.PeerConfig) *Peer {
	peers := make([]peerConfig, 0, len(peersConfig))
	for _, p := range peersConfig {
		peers = append(peers, peerConfig{
			Address: buildURL(p).String(),
			Name:    p.Name,
		})
	}
	return &Peer{
		Name:    selfConfig.Name,
		Address: buildURL(selfConfig).String(),
		Peers:   peers,
	}
}

type Peer struct {
	Name    string
	Address string
	Peers   []peerConfig
	sock    mangos.Socket
}

func (p *Peer) setup() error {
	sock, err := bus.NewSocket()
	if err != nil {
		return errors.Wrap(err, "failed to setup BUS protocol socket")
	}
	if err = sock.Listen(p.Address); err != nil {
		return errors.Wrap(err, "failed to open listening on socket")
	}

	sock.SetPipeEventHook(pipeEventHook)

	p.sock = sock
	return nil
}

func (p *Peer) Run() error {
	if err := p.setup(); err != nil {
		return err
	}

	for _, peer := range p.Peers {
		if err := p.sock.Dial(peer.Address); err != nil {
			zap.L().
				With(zap.Object("peer", peer)).
				Warn("failed to dial socket, peer might not be available right now", zap.Error(err))
		}
	}

	if err := p.sock.Send([]byte(p.Name)); err != nil {
		return err
	}
	zap.L().Info(fmt.Sprintf("%s: SENT '%s' ONTO BUS\n", p.Name, p.Name))

	for range p.Peers {
		msg, err := p.sock.Recv()
		if err != nil {
			return err
		}
		zap.L().Info(fmt.Sprintf("%s: RECEIVED \"%s\" FROM BUS\n", p.Name, string(msg)))
	}
	return nil
}

type peerConfig struct {
	Address string
	Name    string
}

func (p peerConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("address", p.Address)
	enc.AddString("name", p.Name)
	return nil
}

func buildURL(cfg *config.PeerConfig) *url.URL {
	return &url.URL{
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Scheme: transportScheme,
	}
}
