package p2p

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type (
	// transactions as a cache for all the transaction structs.
	transactions struct {
		ts map[transactionID]*transaction
		mu *sync.Mutex
	}
	transactionID = string
	// transaction describes a single replication request and all the responses gathered for it.
	// Each transaction is identified by it's Tid --> transactionID.
	transaction struct {
		Tid transactionID
		// InitiatorName is the name of the peer which initiated the transaction with his protobuf.Request.
		InitiatorName string
		// There may be only one Request per transaction.
		Request   *protobuf.Request
		Responses []*protobuf.Response
		// NotifyChan is used to convey all protobuf.Message associated with this transcation.
		NotifyChan chan *protobuf.Message
	}
)

// newTransactions is a convenience constructor for transactions.
func newTransactions() *transactions {
	return &transactions{
		ts: make(map[transactionID]*transaction, 100),
		mu: new(sync.Mutex),
	}
}

// Put inserts the protobuf.Message, be it protobuf.Request or protobuf.Response into transaction.
// It will create the transaction If not yet present or update the existing one.
// It is safe to use concurrently.
func (t *transactions) Put(message *protobuf.Message) (trans *transaction, created bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, alreadyCreated := t.ts[message.Tid]; !alreadyCreated {
		created = true
		t.ts[message.Tid] = &transaction{Tid: message.Tid}
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

// Delete removes the transaction with the give transactionID from transactions.
func (t *transactions) Delete(tid transactionID) {
	t.mu.Lock()
	delete(t.ts, tid)
	t.mu.Unlock()
}
