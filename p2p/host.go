package p2p

import (
	"context"
	_ "embed"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type (
	Mirror interface {
		Mirror(request *protobuf.Request) error
		Consult(request *protobuf.Request) *protobuf.Response
	}
	Connection interface {
		io.Closer
		Run(ctx context.Context)
		Broadcast(ctx context.Context, data []byte) error
		ReceiveC() <-chan connection.Message
	}
)

const defaultReplicationTimeout = 5 * time.Minute

// NewHost creates new Host with provided arguments while also initialising
// both transactions and conflictsResolver and setting default replication timeout
// if none was provided.
func NewHost(
	host *config.Peer,
	peers config.Peers,
	conn Connection,
	mirror Mirror,
	replicationTimeout time.Duration,
) *Host {
	if replicationTimeout == 0 {
		replicationTimeout = defaultReplicationTimeout
	}
	return &Host{
		Peer:               *host,
		Conn:               conn,
		transactions:       newTransactions(),
		peers:              peers,
		Mirror:             mirror,
		conflicts:          newConflictsResolver(),
		replicationTimeout: replicationTimeout,
	}
}

type (
	// Host represents a single peer we're running in the p2p network.
	Host struct {
		config.Peer
		Conn   Connection
		Mirror Mirror
		// transactions caches existing replication operations.
		transactions *transactions
		peers        config.Peers
		// conflicts helps with detecting and resolving potential conflicts,
		// like performing operations on the same path.
		conflicts *conflictsResolver
		// replicationTimeout describes how long are we willing to wait for a
		// replication transaction to be resolved before dropping it for good.
		replicationTimeout time.Duration
		// cancel main context associated with the listen loop.
		cancel context.CancelFunc
	}
)

// Run kicks of connection processes for the Host and begins listening for incoming messages.
func (h *Host) Run(ctx context.Context) {
	ctx, h.cancel = context.WithCancel(ctx)
	log.Info().Object("host", h).Msg("initializing p2p network connection")
	h.Conn.Run(ctx)
	go h.listen(ctx)
}

// Close first closes the underlying Connection and then
// awaits for all the active transactions to end.
func (h *Host) Close() error {
	log.Info().Msg("Waiting for all active transactions to be closed")
	for {
		if len(h.transactions.ts) == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	// FIXME this might result in a new transaction leaking in before we close the connection for good.
	h.cancel()
	closeErr := h.Conn.Close()
	return closeErr
}

// ErrNotPermitted is the main error of concern as it describes all the scenarios
// when replication request is rejected due to either:
// - lack of consensus among peers
// - conflicting replication requests
var ErrNotPermitted = errors.New("operation was not permitted")

// Replicate broadcasts protobuf.Request and waits for all peers to respond with
// protobuf.Response. It the gathers these responses and processes them do decide
// if the replication was successful (if the peers agreed to do so). In case the
// permission for replication was not granted it returns ErrNotPermitted.
//
// Replicate first tries to detect if there are any conflicts with the ongoing
// transactions before broadcasting the request.
//
// If it failed to send the message to all the peers, it returns an error immediately,
// if it failed to send it to only a subset of the peers it will await only for responses
// of these peers.
func (h *Host) Replicate(ctx context.Context, request *protobuf.Request) error {
	tid := uuid.New().String()
	message, err := protobuf.NewRequestMessage(tid, h.Name, request)
	if err != nil {
		return err
	}

	trans, _ := h.transactions.Put(message)
	defer h.transactions.Delete(message.Tid)

	if err = h.conflicts.DetectAndResolveConflict(h.transactions, message); err != nil {
		h.conflicts.IncrementClock()
		return err
	}

	sentMessagesCount := len(h.peers)
	if err = h.broadcast(ctx, message); err != nil {
		sendErr, ok := err.(*connection.MultiErr)
		if !ok || (ok && len(sendErr.Errors()) == len(h.peers)) {
			return err
		} else if ok {
			sentMessagesCount = len(h.peers) - len(sendErr.Errors())
		}
	}

	ctx, cancel := context.WithTimeout(ctx, h.replicationTimeout)
	defer cancel()
	var msg *protobuf.Message
	for i := 0; i < sentMessagesCount; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg = <-trans.NotifyChan:
			// Message was already set in transaction by listen(),
			// no need to do anything more here than log it.
			log.Info().
				Object("host", h).
				Object("msg", msg).
				Msg("message received")
			continue
		}
	}
	// We gathered all responses, but we don't care what came,
	// since we know one or more peers did not get the broadcast.
	if err != nil {
		return err
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

// listen blocks until new *protobuf.Message is received and finds or creates
// the right transcation for it.
func (h *Host) listen(ctx context.Context) {
	var (
		msg *protobuf.Message
		err error
	)
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-h.Conn.ReceiveC():
			if m.Err != nil {
				log.Err(err).Object("host", h).Msg("Error while collecting a message")
				continue
			}
			msg, err = protobuf.ReadMessage(m.Data)
			if err != nil {
				log.Err(err).Object("host", h).Msg("Failed to unmarshal protobuf message")
				continue
			}
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

// processResponses goes through all protobuf.Response for a single transaction.
// If any protobuf.Response is protobuf.Response_NACK, it will return ErrNotPermitted.
// If any nack responses are detected it counts their occurrences and if the only
// protobuf.Response_Error was protobuf.Response_ERR_TRANSACTION_CONFLICT, then
// it will increment the conflictsResolver.clock.
// The reasoning here is that we only want to treat the transaction as rejected during
// conflicts resolution if it was the only error, otherwise, even If the conflict was
// resolved in our favor, we would still fail, so who cares.
func (h *Host) processResponses(trans *transaction) error {
	errorsCtr := make(map[protobuf.Response_Error]uint)
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
	if _, ok := errorsCtr[protobuf.Response_ERR_TRANSACTION_CONFLICT]; ok && otherErrors == 0 {
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

// handleTransaction is a process spawned by listen, when either protobuf.Response or
// protobuf.Request comes which is not part of any transaction yet.
// It behaves similar to Replicate, but is the process run on the requested peer's side.
// In addition to gathering all protobuf.Response from other peers it also performs
// a broadcast of it's own response when new protobuf.Request comes associated with this
// transaction. The response is consulted using Mirror.Consult method which returns
// a ready-to-be-server protobuf.Response.
// Similar to broadcast, when all responses and a single request are gathered it proceeds to
// check if any protobuf.Response_NACK was served (including its own response) and returns
// ErrNotPermitted if so.
// If the replication was successful it calls Mirror.Mirror with the transaction's request.
func (h *Host) handleTransaction(ctx context.Context, transaction *transaction) error {
	ctx, cancel := context.WithTimeout(ctx, h.replicationTimeout)
	defer cancel()
	var (
		ourResponse protobuf.Response_Type
		msg         *protobuf.Message
	)
	for range h.peers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg = <-transaction.NotifyChan:
		}

		log.Info().
			Object("host", h).
			Object("msg", msg).
			Msg("message received")

		if request := msg.GetRequest(); request != nil {
			response := h.Mirror.Consult(request)
			if response.Type == protobuf.Response_ACK {
				if err := h.conflicts.DetectAndResolveConflict(h.transactions, msg); err != nil {
					response = protobuf.NACK(protobuf.Response_ERR_TRANSACTION_CONFLICT, err)
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
