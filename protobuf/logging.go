package protobuf

import "github.com/rs/zerolog"

func (m *Message) MarshalZerologObject(e *zerolog.Event) {
	e.Str("tid", m.Tid).
		Str("peer_name", m.PeerName)
	if req := m.GetRequest(); req != nil {
		e.Dict("request", zerolog.Dict().
			Stringer("type", req.Type).
			Int("content_size", len(req.Content)))
	}
	if resp := m.GetResponse(); resp != nil {
		e.Dict("response", zerolog.Dict().
			Stringer("type", resp.Type))
		if resp.Error != nil {
			e.Stringer("error", resp.Error)
		}
		if resp.ErrorMsg != nil {
			e.Str("error_msg", *resp.ErrorMsg)
		}
	}
}
