package p2p

import (
	"go.nanomsg.org/mangos/v3"
	"go.uber.org/zap"
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
	zap.L().Debug(msg, zap.Object("peer", p.peerConfigForAddress(pipe.Address())))
}
