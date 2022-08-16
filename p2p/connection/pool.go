package connection

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewPool(
	host *config.Peer,
	peers []*config.Peer,
	connConf *config.Connection,
	tlsConf *tls.Config,
) *Pool {
	msgs := make(chan message, connConf.MessageBufferSize)

	whitelist := make(map[peerName]struct{}, len(peers))
	cs := make(map[peerName]*Connection, len(whitelist))
	for _, peer := range peers {
		whitelist[peer.Name] = struct{}{}
		cs[peer.Name] = NewConnection(host, peer, connConf.SendRecvTimeout, msgs)
	}

	p := &Pool{
		host:              host,
		net:               connConf.Network,
		whitelist:         whitelist,
		pool:              cs,
		msgs:              msgs,
		dialBackoffConfig: &connConf.DialBackoff,
		handshakeTimeout:  connConf.HandshakeTimeout,
	}

	tlsConf.VerifyConnection = p.VerifyConnection
	tlsConf.NextProtos = []string{Protocol}
	p.tlsConfig = tlsConf

	quicConf := quicConfig(p.handshakeTimeout)
	p.quicConf = quicConf

	var err error
	p.listener, err = quic.ListenAddr(host.Address, tlsConf, quicConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create TLS listener")
	}
	p.dialFunc = quic.DialAddrContext

	return p
}

type (
	// Pool manages each Connection, it governs the net.Listener and net.Dialer setup,
	// along with TLS configuration and peer authentication.
	Pool struct {
		// host describes the peer this Pool was created for.
		host *config.Peer
		net  string
		// whitelist is used during VerifyConnection to check if the Subject.CommonName
		// matches any of the peer's names.
		whitelist map[peerName]struct{}
		tlsConfig *tls.Config
		// QUIC related configuration and listening/dialing facilities.
		quicConf *quic.Config
		listener quic.Listener
		dialFunc quicDial
		// pool is the map storing each Connection. The key is peer's name.
		pool map[peerName]*Connection
		// msgs is a buffered channel onto which each Connection.Send publishes message.
		msgs              chan message
		dialBackoffConfig *config.Backoff
		// TODO correct naming here!
		handshakeTimeout time.Duration
	}
	// message allows passing errors along with data through channels.
	message struct {
		data []byte
		err  error
	}
	// peerName is a type alias serving solely code readability.
	peerName = string
)

// Run has to be called once per Pool.
// There's no need to run it in a separate routine.
func (p *Pool) Run(ctx context.Context) {
	ctx = log.With().EmbedObject(p.host).Logger().WithContext(ctx)
	go p.listen(ctx)
	go p.dialAll(ctx)
}

// Broadcast sends the data to all connections by calling Connection.Send in a separate goroutine
// and waits for all of them to finish.
func (p *Pool) Broadcast(data []byte) error {
	var (
		mErr = &SendMultiErr{}
		wg   sync.WaitGroup
	)
	// TODO add context to the Broadcast parameters.
	ctx := context.Background()
	wg.Add(len(p.pool))
	for _, c := range p.pool {
		go func(conn *Connection) {
			if err := conn.Send(ctx, data); err != nil {
				mErr.Append(conn.peer.Name, err)
			}
			wg.Done()
		}(c)
	}
	wg.Wait()
	if !mErr.Empty() {
		return mErr
	}
	return nil
}

// Recv reads one message from the messages channel, it will block until
// a message appears on the channel.
func (p *Pool) Recv() ([]byte, error) {
	m := <-p.msgs
	return m.data, m.err
}

// listen starts accepting connections and runs Connection.Listen for all the
// connections in a separate goroutine
func (p *Pool) listen(ctx context.Context) {
	go p.acceptConnections(ctx)

	for _, c := range p.pool {
		go c.Listen(ctx)
	}
}

// acceptConnections waits for a new connection and attempts to add it in a
// separate goroutine.
func (p *Pool) acceptConnections(ctx context.Context) {
	for {
		conn, err := p.listener.Accept(ctx)
		if err != nil {
			log.Err(err).Msg("failed to accept incoming connection")
			continue
		}
		go func() {
			ctx, cancel := contextWithOptionalTimeout(ctx, p.handshakeTimeout)
			defer cancel()
			if err = p.add(ctx, Server, conn); err != nil {
				log.Debug().Err(err).Send()
			}
		}()
	}
}

// dialAll simply runs dial for each peer's Connection in a separate goroutine.
func (p *Pool) dialAll(ctx context.Context) {
	for _, conn := range p.pool {
		go p.dial(ctx, conn)
	}
}

// dial performs status check before attempting to dial and adds connection to the Pool.
// This function will run forever responding to connection being closed, in this case
// it will attempt to dial again.
// If the dial fails for whatever reason a Backoff mechanism is applied
// until successful or until the Connection changes state to StatusAlive.
func (p *Pool) dial(ctx context.Context, c *Connection) {
	backoff := NewBackoff(p.dialBackoffConfig)
	var err error
	for {
		if err = p.dialOnce(c); err != nil {
			backoff.Next()
			log.Ctx(ctx).Debug().EmbedObject(backoff).Err(err).Msg("failed to dial connection")
			continue
		}
		backoff.Reset()
	}
}

// dialOnce performs a single dial attempt.
// It blocks when the connection was already established and
// waits for it to be closed to make its attempt.
func (p *Pool) dialOnce(c *Connection) error {
	if c.status == StatusAlive {
		c.WaitForClosed()
	}
	ctx, cancel := contextWithOptionalTimeout(context.Background(), p.handshakeTimeout)
	defer cancel()
	conn, err := p.dialFunc(ctx, c.peer.Address, p.tlsConfig, p.quicConf)
	if err != nil {
		return err
	}
	return p.add(ctx, Client, conn)
}

// add extracts peer name from x509.Certificate's Subject.CommonName.
// It searches for the peer in the Pool and calls Connection.Establish.
func (p *Pool) add(ctx context.Context, perspective Perspective, quicConn quic.Connection) error {
	// VerifyConnection makes sure we don't end up with nil pointers or missing map entries here.
	peer := quicConn.ConnectionState().TLS.
		PeerCertificates[0].
		Subject.
		CommonName
	conn := p.pool[peer]
	if err := conn.Establish(ctx, perspective, quicConn); err != nil {
		closeConn(quicConn, err)
		return errors.Wrap(err, "failed to establish connection")
	}
	return nil
}

// VerifyConnection checks peer certificates, it expects a single cert
// containing in the subject CN of the whitelisted peer.
func (p *Pool) VerifyConnection(state tls.ConnectionState) error {
	if len(state.PeerCertificates) != 1 {
		return errors.Errorf("expected exactly one peer certificate, got %d",
			len(state.PeerCertificates))
	}
	var peer string
	for _, crt := range state.PeerCertificates {
		peer = crt.Subject.CommonName
		if _, ok := p.whitelist[peer]; !ok {
			return errors.Errorf(
				"SN: %s is not authorized to participate in the peer network", peer)
		}
	}
	return nil
}
