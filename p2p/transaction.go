package p2p

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type (
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
