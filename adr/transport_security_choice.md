TLS is de facto the standard of choice, it's widely utilized and fairly secure,
at least the 1.3 version. It also adds very minimal overhead to our connection,
the handshake in 1.3 performed with Diffie-Hellman brings not increased
security but also improves performance (one less round trip).

The standard TLS connection is established by client, which verifies the server's
certificate, thankfully TLS also allows client certificate verification, which is
essential for p2p network like ours. The certificates for server and client are all
the same, there's no difference between them, which again comes in handy with p2p.

TLS is based around certificates, which are signed by a CA (Certificate Authority).
The authority can be anyone and anything, it is up to us to decide who's going to be
the trusted authority, this is especially useful for our case, since we can create
such authority and use it to sign certificates for each of the peers. This is done by
generating a certificate (which can be thought of as a public key) and private key for
the CA, these will be used to sign the peers' certificate request and generate the
certificate for each peer.

In result each peer has both it's certificate and the CA certificate. It uses the CA
certificate to verify another peer's certificate.

Additional custom step is performed during the TLS handshake, after the certificates
are parsed and verified each peers checks certificate's subject CN (common name), which
should contain the peer's name. If the peer name is not recognized the handshake will fail,
If it succeeds, the peer's name extracted from subject CN is used to uniquely identify the
connection - who are we speaking with.