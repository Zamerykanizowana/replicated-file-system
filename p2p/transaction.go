package p2p

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type transactionInitiator uint8

const (
	hostInitiator transactionInitiator = iota
	peerInitiator
)

type (
	transactions struct {
		hostName string
		ts       map[transactionId]*transaction
		mu       *sync.Mutex
	}
	transactionId = string
	transaction   struct {
		Tid        string
		Initiator  transactionInitiator
		Request    *protobuf.Request
		Responses  []*protobuf.Response
		NotifyChan chan *protobuf.Message
	}
)

func newTransactions(hostName string) *transactions {
	return &transactions{
		hostName: hostName,
		ts:       make(map[transactionId]*transaction, 100),
		mu:       new(sync.Mutex),
	}
}

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
	case *protobuf.Message_Response:
		trans.Responses = append(trans.Responses, v.Response)
	}

	return
}

func (t *transactions) Delete(tid transactionId) {
	t.mu.Lock()
	delete(t.ts, tid)
	t.mu.Unlock()
}
