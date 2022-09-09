package p2p

import (
	"context"
	_ "embed"
	"io"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type Mirror interface {
	Mirror(request *protobuf.Request) error
	Consult(request *protobuf.Request) *protobuf.Response
}

type Connection interface {
	io.Closer
	Run(ctx context.Context)
	Broadcast(ctx context.Context, data []byte) error
	Receive() (data []byte, err error)
}

func NewHost(
	host *config.Peer,
	peers []*config.Peer,
	conn Connection,
	mirror Mirror,
) *Host {
	return &Host{
		Peer:         *host,
		Conn:         conn,
		transactions: newTransactions(),
		peers:        peers,
		Mirror:       mirror,
		conflicts:    newConflictsResolver(),
	}
}

// Host represents a single peer we're running in the p2p network.
type Host struct {
	config.Peer
	Conn         Connection
	Mirror       Mirror
	transactions *Transactions
	peers        []*config.Peer
	conflicts    *conflictsResolver
}

// Run kicks of connection processes for the Host.
func (h *Host) Run(ctx context.Context) {
	log.Info().Object("host", h).Msg("initializing p2p network connection")
	h.Conn.Run(ctx)
	go h.listen(ctx)
}

// Close closes the underlying connection.Pool.
func (h *Host) Close() error {
	return h.Conn.Close()
}

var (
	ErrTransactionConflict = errors.New("transaction conflict detected and resolved in favor of other peer")
	ErrNotPermitted        = errors.New("operation was not permitted")
)

// Replicate TODO document me.
func (h *Host) Replicate(ctx context.Context, request *protobuf.Request) error {
	tid := uuid.New().String()
	message, err := protobuf.NewRequestMessage(tid, h.Name, request)
	if err != nil {
		return err
	}

	trans, _ := h.transactions.Put(message)
	defer h.transactions.Delete(message.Tid)

	detected, greenLight := h.conflicts.DetectAndResolveConflict(h.transactions, message)
	if detected && !greenLight {
		h.conflicts.IncrementClock()
		return ErrTransactionConflict
	}

	sentMessagesCount := len(h.peers)
	if err = h.broadcast(ctx, message); err != nil {
		if sendErr, ok := err.(*connection.MultiErr); ok && len(sendErr.Errors()) == len(h.peers) {
			return err
		} else {
			sentMessagesCount = len(h.peers) - len(sendErr.Errors())
		}
	}

	for i := 0; i < sentMessagesCount; i++ {
		msg := <-trans.NotifyChan
		log.Info().
			Object("host", h).
			Object("msg", msg).
			Msg("message received")
	}

	return h.processResponses(trans)
}

// SetConflictsClock sets the conflicts clock atomically.
func (h *Host) SetConflictsClock(v uint64) {
	h.conflicts.clock.Store(v)
}

// LoadConflictsClock loads the conflicts clock atomically.
func (h *Host) LoadConflictsClock() uint64 {
	return h.conflicts.clock.Load()
}

func (h *Host) processResponses(trans *Transaction) error {
	errorsCtr := map[protobuf.Response_Error]uint{
		protobuf.Response_ERR_UNKNOWN:              0,
		protobuf.Response_ERR_ALREADY_EXISTS:       0,
		protobuf.Response_ERR_DOES_NOT_EXIST:       0,
		protobuf.Response_ERR_TRANSACTION_CONFLICT: 0,
	}
	var rejected bool
	for _, resp := range trans.Responses {
		if resp.Type == protobuf.Response_NACK {
			errorsCtr[*resp.Error]++
			rejected = true
		}
	}
	logDict := zerolog.Dict()
	var otherErrors uint
	for respErr, count := range errorsCtr {
		logDict.Uint(respErr.String(), count)
		if respErr == protobuf.Response_ERR_TRANSACTION_CONFLICT {
			continue
		}
		otherErrors += count
	}
	// If the only error we encountered was protobuf.Response_ERR_TRANSACTION_CONFLICT.
	if errorsCtr[protobuf.Response_ERR_TRANSACTION_CONFLICT] > 0 && otherErrors == 0 {
		h.conflicts.IncrementClock()
	}
	if rejected {
		log.Debug().
			Object("host", h).
			Dict("errors_count", logDict).
			Msg("transaction was rejected")
		return ErrNotPermitted
	}
	return nil
}

// broadcast sends the protobuf.Message to all the other peers in the network.
func (h *Host) broadcast(ctx context.Context, msg *protobuf.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal protobuf message")
	}
	if err = h.Conn.Broadcast(ctx, data); err != nil {
		return err
	}
	return nil
}

// receive a single protobuf.Message from the network.
// It blocks until the message is received.
func (h *Host) receive() (*protobuf.Message, error) {
	data, err := h.Conn.Receive()
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive message from peer")
	}
	return protobuf.ReadMessage(data)
}

func (h *Host) handleTransaction(ctx context.Context, transaction *Transaction) error {
	var ourResponse protobuf.Response_Type
	for range h.peers {
		msg := <-transaction.NotifyChan

		log.Info().
			Object("host", h).
			Object("msg", msg).
			Msg("message received")

		if request := msg.GetRequest(); request != nil {
			response := h.Mirror.Consult(request)
			if response.Type == protobuf.Response_ACK {
				detected, greenLight := h.conflicts.DetectAndResolveConflict(h.transactions, msg)
				if detected && !greenLight {
					response = protobuf.NACK(
						protobuf.Response_ERR_TRANSACTION_CONFLICT,
						ErrTransactionConflict)
				}
			}
			ourResponse = response.Type
			message := protobuf.NewResponseMessage(msg.Tid, h.Name, response)
			log.Info().
				Object("host", h).
				Object("msg", message).
				Msg("consulted response result")
			if err := h.broadcast(ctx, message); err != nil {
				log.Err(err).
					Object("host", h).
					Object("msg", message).
					Msg("error occurred while broadcasting a response")
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
		return ErrNotPermitted
	}

	return h.Mirror.Mirror(transaction.Request)
}

func (h *Host) listen(ctx context.Context) {
	for {
		msg, err := h.receive()
		if err != nil {
			log.Err(err).
				Object("host", h).
				Msg("Error while collecting a message")
			continue
		}
		trans, created := h.transactions.Put(msg)
		if created {
			go func() {
				if err = h.handleTransaction(ctx, trans); err != nil {
					log.Err(err).Send()
				}
				h.transactions.Delete(trans.Tid)
			}()
		}
		trans.NotifyChan <- msg
	}
}
