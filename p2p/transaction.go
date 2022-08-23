package p2p

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type (
	// Transactions as a cache for all the Transaction structs.
	Transactions struct {
		ts map[TransactionID]*Transaction
		mu *sync.Mutex
	}
	TransactionID = string
	// Transaction describes a single replication request and all the responses gathered for it.
	// Each Transaction is identified by it's Tid --> TransactionID.
	Transaction struct {
		Tid TransactionID
		// InitiatorName is the name of the peer which initiated the Transaction with his protobuf.Request.
		InitiatorName string
		// There may be only one Request per Transaction.
		Request   *protobuf.Request
		Responses []*protobuf.Response
		// NotifyChan is used to convey all protobuf.Message associated with this transcation.
		NotifyChan chan *protobuf.Message
	}
)

// newTransactions is a convenience constructor for Transactions.
func newTransactions() *Transactions {
	return &Transactions{
		ts: make(map[TransactionID]*Transaction, 100),
		mu: new(sync.Mutex),
	}
}

// Put inserts the protobuf.Message, be it protobuf.Request or protobuf.Response into Transaction.
// It will create the Transaction If not yet present or update the existing one.
// It is safe to use concurrently.
func (t *Transactions) Put(message *protobuf.Message) (trans *Transaction, created bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, alreadyCreated := t.ts[message.Tid]; !alreadyCreated {
		created = true
		t.ts[message.Tid] = &Transaction{Tid: message.Tid}
		t.ts[message.Tid].NotifyChan = make(chan *protobuf.Message)
	}

	trans = t.ts[message.Tid]

	switch v := message.Type.(type) {
	case *protobuf.Message_Request:
		if trans.Request != nil {
			log.Error().Interface("msg", message).Msg("Request has already been set in transaction")
		}
		trans.Request = v.Request
		trans.InitiatorName = message.PeerName
	case *protobuf.Message_Response:
		trans.Responses = append(trans.Responses, v.Response)
	}

	return
}

// Delete removes the Transaction with the give TransactionID from Transactions.
func (t *Transactions) Delete(tid TransactionID) {
	t.mu.Lock()
	delete(t.ts, tid)
	t.mu.Unlock()
}
