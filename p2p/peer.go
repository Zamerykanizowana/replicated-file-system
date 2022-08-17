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
	"github.com/Zamerykanizowana/replicated-file-system/mirror"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection/tlsconf"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func NewPeer(
	name string,
	peersConfig []*config.Peer,
	connConfig *config.Connection,
	mirror *mirror.Mirror,
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
		mirror       *mirror.Mirror
	}
	Transactions struct {
		ts map[TransactionId]*Transaction
		mu *sync.Mutex
	}
	TransactionId = string
	Transaction   struct {
		Request    *protobuf.Request
		Responses  []*protobuf.Response
		NotifyChan chan *protobuf.Message
	}
)

func (t *Transactions) Has(tid string) (has bool) {
	_, has = t.ts[tid]
	return
}

func (t *Transactions) Get(tid string) *Transaction {
	return t.ts[tid]
}

func (t *Transactions) Put(message *protobuf.Message) (transaction *Transaction, created bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, alreadyCreated := t.ts[message.Tid]; !alreadyCreated {
		created = true
		t.ts[message.Tid] = &Transaction{}
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
func (p *Peer) Run() {
	log.Info().Object("peer", p).Msg("initializing p2p network connection")
	p.connPool.Run(context.Background())
	go p.listen()
}

func (p *Peer) Replicate(
	requestType protobuf.Request_Type,
	metadata *protobuf.Request_Metadata,
	content []byte,
) error {
	transactionId := uuid.New().String()
	request, err := protobuf.NewRequestMessage(transactionId, p.Name, requestType, metadata, content)
	if err != nil {
		return err
	}

	transaction, _ := p.transactions.Put(request)

	if err = p.broadcast(request); err != nil {
		p.transactions.Delete(request.Tid)
		return err
	}

	for range p.peers {
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
func (p *Peer) broadcast(msg *protobuf.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal protobuf message")
	}
	if err = p.connPool.Broadcast(context.Background(), data); err != nil {
		return errors.Wrap(err, "failed to send the message to some of the peers")
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

func (p *Peer) handleTransaction(transaction *Transaction) error {
	var ourResponse protobuf.Response_Type
	for range p.peers {
		msg := <-transaction.NotifyChan

		log.Info().Object("msg", msg).Msg("message received")

		if request := msg.GetRequest(); request != nil {
			switch p.mirror.Consult(request) {
			case true:
				ourResponse = protobuf.Response_ACK
			case false:
				ourResponse = protobuf.Response_NACK
			}
			response := protobuf.NewResponseMessage(msg.Tid, p.Name, ourResponse, nil)
			if err := p.broadcast(response); err != nil {
				log.Err(err).Interface("response", response).Msg("error occurred while broadcasting a response")
			}
		}
	}

	for _, resp := range transaction.Responses {
		if resp.Type != protobuf.Response_ACK {
			return errors.New("operation was not permitted")
		}
	}
	return p.mirror.Mirror(transaction.Request)
}

func (p *Peer) listen() {
	for {
		msg, err := p.receive()
		if err != nil {
			log.Err(err).Msg("Error while collecting a message")
			continue
		}
		transaction, created := p.transactions.Put(msg)
		if created {
			go func() {
				// TODO log it.
				_ = p.handleTransaction(transaction)
			}()
		}
		transaction.NotifyChan <- msg
	}
}
