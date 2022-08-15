package protobuf

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func NewRequestMessage(tid, peerName string, typ Request_Type, content []byte) (*Message, error) {
	req, err := newRequest(typ, content)
	if err != nil {
		return nil, err
	}
	return newMessage(tid, peerName, req), nil
}

func NewResponseMessage(tid, peerName string, typ Response_Type, respErr *Response_Error) *Message {
	return newMessage(tid, peerName, newResponse(typ, respErr))
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

func newRequest(typ Request_Type, content []byte) (*Request, error) {
	if content != nil {
		var err error
		content, err = compress(content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compress Request.Content")
		}
	}
	return &Request{
		Type:    typ,
		Content: content,
	}, nil
}

func newResponse(typ Response_Type, respErr *Response_Error) *Response {
	return &Response{
		Type:  typ,
		Error: respErr,
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
