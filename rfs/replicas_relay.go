package rfs

import (
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type ReplicasRelay interface {
	Run()
	Broadcast(msg *protobuf.Message) error
	Receive() (*protobuf.Message, error)
}
