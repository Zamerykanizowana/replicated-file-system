package connection

// Protocol could be virtually anything, it is required by TLS ALPN step which is
// part of the QUIC protocol.
const Protocol = "p2p-quic"
