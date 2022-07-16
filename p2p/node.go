package p2p

import (
	_ "embed"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/p2p/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func NewNode(
	self *config.Peer,
	peersConfig []*config.Peer,
	connConfig *config.Connection,
) *Node {
	return &Node{
		Name:     self.Name,
		Address:  self.Address,
		connPool: connection.NewPool(self, connConfig, peersConfig),
	}
}

type Node struct {
	Name     string
	Address  string
	connPool *connection.Pool
}

func (n *Node) Run() {
	n.connPool.Run()
}

func (n *Node) Send(msg *protobuf.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal protobuf message")
	}
	if err = n.connPool.Send(data); err != nil {
		return errors.Wrap(err, "failed to send the message to some of the peers")
	}
	return nil
}

func (n *Node) Receive() (*protobuf.Message, error) {
	data, err := n.connPool.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive message from peer")
	}
	return protobuf.ReadMessage(data)
}
