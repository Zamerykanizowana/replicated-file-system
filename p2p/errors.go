package p2p

import "strings"

type SendMultiErr map[string]error

func (s SendMultiErr) Error() string {
	b := strings.Builder{}
	b.WriteString("sending errors: ")
	for peerAddr, err := range s {
		b.WriteString(peerAddr)
		b.WriteString(" > ")
		b.WriteString(err.Error())
		b.WriteString(", ")
	}
	return b.String()[:b.Len()-2]
}

func (s SendMultiErr) Append(addr string, err error) {
	s[addr] = err
}
