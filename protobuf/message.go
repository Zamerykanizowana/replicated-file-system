package protobuf

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func NewRequestMessage(
	tid, peerName string,
	request *Request,
) (*Message, error) {
	if request.Content != nil {
		var err error
		request.Content, err = compress(request.Content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compress Request.Content")
		}
	}
	return newMessage(tid, peerName, request), nil
}

func ACK() *Response {
	return &Response{Type: Response_ACK}
}

func NACK(respErr Response_Error, err error) *Response {
	errMsg := err.Error()
	return &Response{Type: Response_NACK, Error: &respErr, ErrorMsg: &errMsg}
}

func NewResponseMessage(tid, peerName string, response *Response) *Message {
	return newMessage(tid, peerName, response)
}

func newMessage(tid, peerName string, pm proto.Message) *Message {
	var imt isMessage_Type
	switch v := pm.(type) {
	case *Request:
		imt = &Message_Request{Request: v}
	case *Response:
		imt = &Message_Response{Response: v}
	default:
		log.Panic().Msg("invalid Message type provided, expected *Request or *Response")
	}
	return &Message{
		Tid:      tid,
		PeerName: peerName,
		Type:     imt,
	}
}

func ReadMessage(raw []byte) (*Message, error) {
	var msg Message
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal Message")
	}
	if req, ok := msg.Type.(*Message_Request); ok && req.Request.Content != nil {
		var err error
		req.Request.Content, err = decompress(req.Request.Content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decompress Request.Content")
		}
	}
	return &msg, nil
}
