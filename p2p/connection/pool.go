package connection

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewPool(
	self *config.Peer,
	conf *config.Connection,
	pcs []*config.Peer,
) *Pool {
	msgs := make(chan message, conf.MessageBufferSize)

	whitelist := make(map[peerName]struct{}, len(pcs))
	cs := make(map[peerName]*Connection, len(whitelist))
	for _, p := range pcs {
		whitelist[p.Name] = struct{}{}
		cs[p.Name] = NewConnection(p, conf.SendRecvTimeout, msgs)
	}

	p := &Pool{
		self:              self,
		net:               conf.Network,
		whitelist:         whitelist,
		pool:              cs,
		msgs:              msgs,
		dialBackoffConfig: conf.DialBackoff,
		handshakeTimeout:  conf.HandshakeTimeout,
	}

	tlsConf := tlsConfig(conf.GetTLSVersion())
	tlsConf.VerifyConnection = p.VerifyConnection

	p.dialer = &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    tlsConf}
	if p.handshakeTimeout != 0 {
		p.dialer.NetDialer.Timeout = p.handshakeTimeout
	}

	var err error
	p.listener, err = tls.Listen(p.net, self.Address, tlsConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create TLS listener")
	}

	return p
}

type (
	// Pool manages each Connection, it governs the net.Listener and net.Dialer setup,
	// along with TLS configuration and peer authentication.
	Pool struct {
		// self describes the peer this Pool was created for.
		self *config.Peer
		net  string
		// whitelist is used during VerifyConnection to check if the Subject.CommonName
		// matches any of the peer's names.
		whitelist map[peerName]struct{}
		dialer    *tls.Dialer
		listener  net.Listener
		// pool is the map storing each Connection. The key is peer's name.
		pool map[peerName]*Connection
		// msgs is a buffered channel onto which each Connection.Send publishes message.
		msgs              chan message
		dialBackoffConfig *config.Backoff
		handshakeTimeout  time.Duration
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
func (p *Pool) Run() {
	go p.listen()
	go p.dialAll()
}

// Send sends the data to all connections by calling Connection.Send in a separate goroutine
// and waits for all of them to finish.
func (p *Pool) Send(data []byte) error {
	var (
		mErr = &SendMultiErr{}
		wg   sync.WaitGroup
	)
	wg.Add(len(p.pool))
	for _, c := range p.pool {
		go func(conn *Connection) {
			if err := conn.Send(data); err != nil {
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
func (p *Pool) listen() {
	go p.acceptConnections()

	for _, c := range p.pool {
		go c.Listen()
	}
}

// acceptConnections waits for a new connection and attempts to add it in a
// separate goroutine.
func (p *Pool) acceptConnections() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			log.Err(err).Msg("failed to accept incoming connection")
			continue
		}
		go func() {
			if err = p.add(conn, tlsServer); err != nil {
				log.Debug().Err(err).Send()
			}
		}()
	}
}

// dialAll simply runs dial for each peer's Connection in a separate goroutine.
func (p *Pool) dialAll() {
	for _, conn := range p.pool {
		go p.dial(conn)
	}
}

// dial performs status check before attempting to dial and adds connection to the Pool.
// This function will run forever responding to connection being closed, in this case
// it will attempt to dial again.
// If the dial fails for whatever reason a Backoff mechanism is applied
// until successful or until the Connection changes state to StatusAlive.
func (p *Pool) dial(c *Connection) {
	backoff := NewBackoff(p.dialBackoffConfig)
	for {
		if c.status == StatusAlive {
			backoff.Reset()
			c.WaitForClosed()
		}
		netConn, err := p.dialer.Dial(p.net, c.peer.Address)
		if err != nil {
			backoff.Next()
			c.log.Debug().EmbedObject(backoff).Err(err).Msg("failed to dial connection")
			continue
		}
		if err = p.add(netConn, tlsClient); err != nil {
			backoff.Next()
			c.log.Debug().EmbedObject(backoff).Err(err).Msg("failed to add connection")
		}
	}
}

// tlsRole describes a logical role the peer's connection assumes.
// It can be either server or client depending on the method
// the connection was established through:
// - Dial: 		tlsClient
// - Accept:    tlsServer
type tlsRole uint8

const (
	tlsServer tlsRole = iota
	tlsClient
)

// add extracts peer name from x509.Certificate's Subject.CommonName.
// It searches for the peer in the Pool and calls Connection.Establish.
// Depending on which tlsRole is passed a handshake might be called on tls.Conn for tlsServer.
func (p *Pool) add(netConn net.Conn, role tlsRole) error {
	tlsConn, isTLS := netConn.(*tls.Conn)
	if !isTLS {
		closeConn(netConn)
		return errors.New("non TLS connection")
	}

	// tls client is initiating handshake during dial, we have to do it
	// manually here as a server.
	if role == tlsServer {
		if err := p.handshake(tlsConn); err != nil {
			return errors.Wrap(err, "tls handshake failed")
		}
	}

	state := tlsConn.ConnectionState()
	// At this point if this cert does not exist,
	// we have done something very wrong code wise.
	if len(state.PeerCertificates) != 1 {
		return errors.Errorf("expected excatly one peer certificate, got: %d",
			len(state.PeerCertificates))
	}

	peerName := state.PeerCertificates[0].Subject.CommonName
	conn := p.pool[peerName]
	if err := conn.Establish(netConn); err != nil {
		closeConn(netConn)
		return errors.Wrap(err, "failed to establish connection")
	}
	return nil
}

// handshake should only be run by tlsServer, since tlsClient does it when dialing.
func (p *Pool) handshake(conn *tls.Conn) error {
	if p.handshakeTimeout == 0 {
		return conn.Handshake()
	}
	ctx, cancel := context.WithTimeout(context.Background(), p.handshakeTimeout)
	defer cancel()
	return conn.HandshakeContext(ctx)
}

// VerifyConnection checks peer certificates, it expects a single cert
// containing in the subject CN of the whitelisted peer.
func (p *Pool) VerifyConnection(state tls.ConnectionState) error {
	if len(state.PeerCertificates) != 1 {
		return errors.Errorf("expected exactly one peer certificate, got %d",
			len(state.PeerCertificates))
	}
	var pn string
	for _, crt := range state.PeerCertificates {
		pn = crt.Subject.CommonName
		if _, ok := p.whitelist[pn]; !ok {
			return errors.Errorf(
				"SN: %s is not authorized to participate in the peer network", pn)
		}
	}
	if p.pool[pn].status == StatusAlive {
		return errors.Errorf("connection between %s was already established", pn)
	}
	return nil
}
