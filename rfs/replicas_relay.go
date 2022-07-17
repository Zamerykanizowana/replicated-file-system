package rfs

import (
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type ReplicasRelay interface {
	Run()
	Send(msg *protobuf.Message) error
	Receive() (*protobuf.Message, error)
}
