package p2p

import (
	"context"
	_ "embed"
	"sync"

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

func NewPeer(
	name string,
	peersConfig []*config.Peer,
	connConfig *config.Connection,
	mirror Mirror,
) *Peer {
	var self config.Peer
	peers := make([]*config.Peer, 0, len(peersConfig)-1)
	for _, p := range peersConfig {
		if name == p.Name {
			self = *p
			continue
		}
		peers = append(peers, p)
	}
	if len(self.Name) == 0 {
		log.Fatal().
			Str("name", name).
			Interface("peers_config", peersConfig).
			Msg("invalid peer name provided, peer must be listed in the peers config")
	}
	return &Peer{
		Peer:     self,
		connPool: connection.NewPool(&self, peers, connConfig, tlsconf.Default(connConfig.GetTLSVersion())),
		transactions: Transactions{
			ts: make(map[TransactionId]*Transaction, len(peersConfig)),
			mu: new(sync.Mutex),
		},
		peers:  peers,
		mirror: mirror,
	}
}

// Peer represents a single peer we're running in the p2p network.
type (
	Peer struct {
		config.Peer
		connPool     *connection.Pool
		transactions Transactions
		peers        []*config.Peer
		mirror       Mirror
	}
	Transactions struct {
		ts map[TransactionId]*Transaction
		mu *sync.Mutex
	}
	TransactionId = string
	Transaction   struct {
		Tid        string
		Request    *protobuf.Request
		Responses  []*protobuf.Response
		NotifyChan chan *protobuf.Message
	}
)

func (t *Transactions) Put(message *protobuf.Message) (transaction *Transaction, created bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, alreadyCreated := t.ts[message.Tid]; !alreadyCreated {
		created = true
		t.ts[message.Tid] = &Transaction{Tid: message.Tid}
		t.ts[message.Tid].NotifyChan = make(chan *protobuf.Message)
	}

	transaction = t.ts[message.Tid]

	switch v := message.Type.(type) {
	case *protobuf.Message_Request:
		if transaction.Request != nil {
			log.Error().Interface("msg", message).Msg("Request has already been set in transaction")
		}
		transaction.Request = v.Request
	case *protobuf.Message_Response:
		transaction.Responses = append(transaction.Responses, v.Response)
	}

	return
}

func (t *Transactions) Delete(tid TransactionId) {
	t.mu.Lock()
	delete(t.ts, tid)
	t.mu.Unlock()
}

// Run kicks of connection processes for the Peer.
func (p *Peer) Run(ctx context.Context) {
	log.Info().Object("peer", p).Msg("initializing p2p network connection")
	p.connPool.Run(ctx)
	go p.listen(ctx)
}

// Close closes the underlying connection.Pool.
func (p *Peer) Close() {
	p.connPool.Close()
}

func (p *Peer) Replicate(ctx context.Context, request *protobuf.Request) error {
	transactionId := uuid.New().String()
	message, err := protobuf.NewRequestMessage(transactionId, p.Name, request)
	if err != nil {
		return err
	}

	transaction, _ := p.transactions.Put(message)

	sentMessagesCount := len(p.peers)
	if err = p.broadcast(ctx, message); err != nil {
		if sendErr, ok := err.(*connection.SendMultiErr); ok && len(sendErr.Errors()) == len(p.peers) {
			p.transactions.Delete(message.Tid)
			return err
		} else {
			sentMessagesCount = len(p.peers) - len(sendErr.Errors())
		}
	}

	for i := 0; i < sentMessagesCount; i++ {
		msg := <-transaction.NotifyChan
		log.Info().Object("msg", msg).Msg("message received")
	}

	for _, resp := range transaction.Responses {
		if resp.Type != protobuf.Response_ACK {
			return errors.New("operation was not permitted")
		}
	}
	return nil
}

// broadcast sends the protobuf.Message to all the other peers in the network.
func (p *Peer) broadcast(ctx context.Context, msg *protobuf.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal protobuf message")
	}
	if err = p.connPool.Broadcast(ctx, data); err != nil {
		return err
	}
	return nil
}

// receive a single protobuf.Message from the network.
// It blocks until the message is received.
func (p *Peer) receive() (*protobuf.Message, error) {
	data, err := p.connPool.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive message from peer")
	}
	return protobuf.ReadMessage(data)
}

func (p *Peer) handleTransaction(ctx context.Context, transaction *Transaction) error {
	var ourResponse protobuf.Response_Type
	for range p.peers {
		msg := <-transaction.NotifyChan

		log.Info().Object("msg", msg).Msg("message received")

		if request := msg.GetRequest(); request != nil {
			response := p.mirror.Consult(request)
			ourResponse = response.Type
			message := protobuf.NewResponseMessage(msg.Tid, p.Name, response)
			log.Info().Object("message", message).Msg("consulted response result")
			if err := p.broadcast(ctx, message); err != nil {
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

	return p.mirror.Mirror(transaction.Request)
}

func (p *Peer) listen(ctx context.Context) {
	for {
		msg, err := p.receive()
		if err != nil {
			log.Err(err).Msg("Error while collecting a message")
			continue
		}
		transaction, created := p.transactions.Put(msg)
		if created {
			go func() {
				if err = p.handleTransaction(ctx, transaction); err != nil {
					log.Err(err)
				}
				p.transactions.Delete(transaction.Tid)
			}()
		}
		transaction.NotifyChan <- msg
	}
}
