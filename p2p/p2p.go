package p2p

import (
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/bus"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

const transportScheme = "tcp"

func NewPeer(selfConfig *config.PeerConfig, peersConfig []*config.PeerConfig) *Peer {
	peers := make(map[string]peerConfig, len(peersConfig))
	for _, p := range peersConfig {
		pc := peerConfig{
			Address: buildURL(p).String(),
			Name:    p.Name,
		}
		peers[pc.Address] = pc
	}
	return &Peer{
		Name:    selfConfig.Name,
		Address: buildURL(selfConfig).String(),
		Peers:   peers,
	}
}

type (
	Peer struct {
		Name    string
		Address string
		// Peers stores address as a key, for fast searching.
		Peers map[string]peerConfig
		sock  mangos.Socket
	}
)

func (p *Peer) setup() error {
	sock, err := bus.NewSocket()
	if err != nil {
		return errors.Wrap(err, "failed to setup BUS protocol socket")
	}
	if err = sock.Listen(p.Address); err != nil {
		return errors.Wrap(err, "failed to open listening on socket")
	}

	sock.SetPipeEventHook(p.pipeEventHook)

	p.sock = sock
	return nil
}

func (p *Peer) Run() error {
	if err := p.setup(); err != nil {
		return err
	}

	go p.listen()

	for _, peer := range p.Peers {
		if err := p.sock.DialOptions(peer.Address, dialOptions()); err != nil {
			return errors.Wrap(err, "failed to dial socket")
		}
	}

	for {
		time.Sleep(3 * time.Second)
		if err := p.Send(); err != nil {
			return err
		}
	}
}

func (p *Peer) Send() error {
	msg := &protobuf.Message{
		PeerName: p.Name,
		Type:     protobuf.Message_REPLICATE,
		Content:  []byte("just a test bro!"),
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal protobuf message")
	}
	if err = p.sock.Send(data); err != nil {
		return errors.Wrap(err, "failed to send message")
	}
	//zap.L().Info("sent message",
	//	zap.String("from", msg.PeerName),
	//	zap.String("type", msg.Type.String()),
	//	zap.ByteString("content", msg.Content))
	return nil
}

func (p *Peer) listen() {
	for {
		raw, err := p.sock.Recv()
		if err != nil {
			zap.L().Error("failed to receive message", zap.Error(err))
			continue
		}
		var msg protobuf.Message
		if err = proto.Unmarshal(raw, &msg); err != nil {
			zap.L().Error("failed to unmarshal protobuf message", zap.Error(err))
			continue
		}
		zap.L().Info("received message",
			zap.String("from", msg.PeerName),
			zap.String("type", msg.Type.String()),
			zap.ByteString("content", msg.Content))
	}
}

func (p *Peer) peerConfigForAddress(address string) peerConfig {
	pc, found := p.Peers[address]
	if !found {
		zap.L().Error("peer was not found on peers list", zap.String("address", address))
	}
	return pc
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

func dialOptions() map[string]interface{} {
	return map[string]interface{}{
		// Setting this to true might be tempting, but it causes connection duplicate.
		// We will send and receive two messages.
		mangos.OptionDialAsynch: false,
	}
}
