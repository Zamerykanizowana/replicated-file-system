package p2p

import (
	"github.com/rs/zerolog/log"
	"go.nanomsg.org/mangos/v3"
)

func (p *Peer) pipeEventHook(event mangos.PipeEvent, pipe mangos.Pipe) {
	if pipe.Listener() != nil {
		return
	}
	var msg string
	switch event {
	case mangos.PipeEventAttaching:
		msg = "attaching peer"
	case mangos.PipeEventAttached:
		msg = "peer attached"
	case mangos.PipeEventDetached:
		msg = "peer detached"
	}
	log.Debug().Interface("peer", p.peerConfigForAddress(pipe.Address())).Msg(msg)
}
