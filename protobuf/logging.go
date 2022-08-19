package protobuf

import "github.com/rs/zerolog"

func (m *Message) MarshalZerologObject(e *zerolog.Event) {
	e.Str("tid", m.Tid).
		Str("peer_name", m.PeerName)
	if req := m.GetRequest(); req != nil {
		e.Object("request", req)
	}
	if resp := m.GetResponse(); resp != nil {
		e.Object("response", resp)
	}
}

func (r *Request) MarshalZerologObject(e *zerolog.Event) {
	e.Stringer("type", r.Type).
		Int("content_size", len(r.Content))
}

func (r *Response) MarshalZerologObject(e *zerolog.Event) {
	e.Stringer("type", r.Type)
	if r.Error != nil {
		e.Stringer("error", r.Error)
	}
	if r.ErrorMsg != nil {
		e.Str("error_msg", *r.ErrorMsg)
	}
}
