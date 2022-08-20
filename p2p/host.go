package p2p

import (
	"context"
	_ "embed"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection/tlsconf"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type Mirror interface {
	Mirror(request *protobuf.Request) error
	Consult(request *protobuf.Request) *protobuf.Response
}

func NewHost(
	name string,
	peersConfig config.Peers,
	connConfig *config.Connection,
	mirror Mirror,
) *Host {
	host, peers := peersConfig.Pop(name)
	if len(host.Name) == 0 {
		log.Fatal().
			Str("name", name).
			Interface("peers_config", peersConfig).
			Msg("invalid peer name provided, peer must be listed in the peers config")
	}
	return &Host{
		Peer:         *host,
		connPool:     connection.NewPool(host, peers, connConfig, tlsconf.Default(connConfig.GetTLSVersion())),
		transactions: newTransactions(host.Name),
		peers:        peers,
		mirror:       mirror,
	}
}

// Host represents a single peer we're running in the p2p network.
type Host struct {
	config.Peer
	connPool     *connection.Pool
	transactions *transactions
	peers        []*config.Peer
	mirror       Mirror
}

// Run kicks of connection processes for the Host.
func (h *Host) Run(ctx context.Context) {
	log.Info().Object("host", h).Msg("initializing p2p network connection")
	h.connPool.Run(ctx)
	go h.listen(ctx)
}

// Close closes the underlying connection.Pool.
func (h *Host) Close() error {
	h.connPool.Close()
	return nil
}

func (h *Host) Replicate(ctx context.Context, request *protobuf.Request) error {
	tid := uuid.New().String()
	message, err := protobuf.NewRequestMessage(tid, h.Name, request)
	if err != nil {
		return err
	}

	trans, _ := h.transactions.Put(message)

	sentMessagesCount := len(h.peers)
	if err = h.broadcast(ctx, message); err != nil {
		if sendErr, ok := err.(*connection.SendMultiErr); ok && len(sendErr.Errors()) == len(h.peers) {
			h.transactions.Delete(message.Tid)
			return err
		} else {
			sentMessagesCount = len(h.peers) - len(sendErr.Errors())
		}
	}

	for i := 0; i < sentMessagesCount; i++ {
		msg := <-trans.NotifyChan
		log.Info().Object("msg", msg).Msg("message received")
	}

	for _, resp := range trans.Responses {
		if resp.Type != protobuf.Response_ACK {
			return errors.New("operation was not permitted")
		}
	}
	return nil
}

// broadcast sends the protobuf.Message to all the other peers in the network.
func (h *Host) broadcast(ctx context.Context, msg *protobuf.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal protobuf message")
	}
	if err = h.connPool.Broadcast(ctx, data); err != nil {
		return err
	}
	return nil
}

// receive a single protobuf.Message from the network.
// It blocks until the message is received.
func (h *Host) receive() (*protobuf.Message, error) {
	data, err := h.connPool.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive message from peer")
	}
	return protobuf.ReadMessage(data)
}

func (h *Host) handleTransaction(ctx context.Context, transaction *transaction) error {
	var ourResponse protobuf.Response_Type
	for range h.peers {
		msg := <-transaction.NotifyChan

		log.Info().Object("msg", msg).Msg("message received")

		if request := msg.GetRequest(); request != nil {
			response := h.mirror.Consult(request)
			ourResponse = response.Type
			message := protobuf.NewResponseMessage(msg.Tid, h.Name, response)
			log.Info().Object("message", message).Msg("consulted response result")
			if err := h.broadcast(ctx, message); err != nil {
				log.Err(err).Object("message", message).Msg("error occurred while broadcasting a response")
			}
		}
	}

	permitted := ourResponse == protobuf.Response_ACK
	for _, resp := range transaction.Responses {
		if resp.Type == protobuf.Response_NACK {
			permitted = false
			break
		}
	}
	if !permitted {
		return errors.New("operation was not permitted")
	}

	return h.mirror.Mirror(transaction.Request)
}

func (h *Host) listen(ctx context.Context) {
	for {
		msg, err := h.receive()
		if err != nil {
			log.Err(err).Msg("Error while collecting a message")
			continue
		}
		trans, created := h.transactions.Put(msg)
		if created {
			go func() {
				if err = h.handleTransaction(ctx, trans); err != nil {
					log.Err(err)
				}
				h.transactions.Delete(trans.Tid)
			}()
		}
		trans.NotifyChan <- msg
	}
}
