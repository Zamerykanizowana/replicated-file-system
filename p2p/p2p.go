package p2p

import (
	"fmt"
	"net/url"
	"time"

	"go.nanomsg.org/mangos/v3/protocol/bus"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
	"go.uber.org/zap"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

const transportScheme = "tcp"

type peer struct {
	Address string
	Name    string
}

func Run(selfConfig *config.PeerConfig, peersConfig []*config.PeerConfig) error {
	sock, err := bus.NewSocket()
	if err != nil {
		zap.L().With(zap.Error(err)).Fatal("failed to setup BUS protocol socket")
	}
	selfURL := buildURL(selfConfig)
	if err = sock.Listen(selfURL.String()); err != nil {
		zap.L().With(zap.Error(err)).Fatal("failed to start listening on the BUS")
	}

	peers := make([]peer, 0, len(peersConfig))
	for _, p := range peersConfig {
		peerURL := buildURL(p)
		peers = append(peers, peer{
			Address: peerURL.String(),
			Name:    p.Name,
		})
	}

	time.Sleep(time.Second)

	for _, p := range peers {
		if err = sock.Dial(p.Address); err != nil {
			return err
		}
	}
	time.Sleep(time.Second)

	zap.L().Info(fmt.Sprintf("%s: SENDING '%s' ONTO BUS\n", selfConfig.Name, selfConfig.Name))
	if err = sock.Send([]byte(selfConfig.Name)); err != nil {
		return err
	}
	for range peersConfig {
		msg, err := sock.Recv()
		if err != nil {
			return err
		}
		zap.L().Info(fmt.Sprintf("%s: RECEIVED \"%s\" FROM BUS\n", selfConfig.Name, string(msg)))
	}
	return nil
}

func buildURL(cfg *config.PeerConfig) url.URL {
	return url.URL{
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Scheme: transportScheme,
	}
}
