package test

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/p2p"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func P2P(node *p2p.Node) error {
	node.Run()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for {
			msg, err := node.Receive()
			if err != nil {
				log.Err(err).Send()
				continue
			}
			log.Info().
				Str("from", msg.PeerName).
				Str("tid", msg.Tid).
				Bytes("content", msg.Type.(*protobuf.Message_Request).Request.Content).
				Msg("received message")
		}
	}()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			msg, err := protobuf.NewRequestMessage(
				uuid.New().String(), node.Name, protobuf.Request_CREATE, []byte("hey ho!"))
			if err != nil {
				log.Err(err).Send()
				continue
			}
			if err = node.Send(msg); err != nil {
				log.Err(err).Msg("failed to send message")
			}
		}
	}()

	wg.Wait()
	return nil
}
