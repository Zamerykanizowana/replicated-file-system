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

	whitelist := make(map[string]struct{}, len(pcs))
	cs := make(map[string]*Connection, len(whitelist))
	for _, p := range pcs {
		whitelist[p.Name] = struct{}{}
		cs[p.Name] = NewConnection(p, conf.SendRecvTimeout, msgs)
	}

	pool := &Pool{
		self:              self,
		net:               conf.Network,
		whitelist:         whitelist,
		cs:                cs,
		msgs:              msgs,
		dialBackoffConfig: conf.DialBackoff,
		handshakeTimeout:  conf.HandshakeTimeout,
	}

	tlsConf := tlsConfig(conf.GetTLSVersion())
	tlsConf.VerifyConnection = pool.VerifyConnection

	pool.dialer = &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config:    tlsConf}
	if pool.handshakeTimeout != 0 {
		pool.dialer.NetDialer.Timeout = pool.handshakeTimeout
	}

	var err error
	pool.listener, err = tls.Listen(pool.net, self.Address, tlsConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create TLS listener")
	}

	return pool
}

type (
	// Pool manages each Connection, it governs the net.Listener and net.Dialer setup,
	// along with TLS configuration and peer authentication.
	Pool struct {
		// self describes the peer this Pool was created for.
		self              *config.Peer
		net               string
		whitelist         map[string]struct{}
		dialer            *tls.Dialer
		listener          net.Listener
		cs                map[string]*Connection
		msgs              chan message
		dialBackoffConfig *config.Backoff
		handshakeTimeout  time.Duration
	}
	// message allows passing errors along with data through channels.
	message struct {
		data []byte
		err  error
	}
)

// Run has to be called once per Pool.
// There's no need to run it in a separate routine.
func (p *Pool) Run() {
	go p.listen()
	go p.dialAll()
}

func (p *Pool) Send(data []byte) error {
	var (
		mErr SendMultiErr
		wg   sync.WaitGroup
		mu   sync.Mutex
	)
	wg.Add(len(p.cs))
	for _, c := range p.cs {
		go func(conn *Connection) {
			if err := conn.Send(data); err != nil {
				mu.Lock()
				if mErr == nil {
					mErr = make(SendMultiErr)
				}
				mErr[conn.peer.Name] = err
				mu.Unlock()
			}
			wg.Done()
		}(c)
	}
	wg.Wait()
	if len(mErr) > 0 {
		return mErr
	}
	return nil
}

func (p *Pool) Recv() ([]byte, error) {
	m := <-p.msgs
	return m.data, m.err
}

func (p *Pool) listen() {
	go p.acceptConnections()

	for _, c := range p.cs {
		go c.Listen()
	}
}

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
	for _, conn := range p.cs {
		go p.dial(conn)
	}
}

// dial performs status check before attempting to dial and adds connection to the Pool.
// This function will run forever responding to connection being closed, in this case
// it will attempt to dial again.
// If the dial fails for whatever reason a Backoff mechanism is applied,
// until successful or the Connection changes state to StatusAlive.
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

type tlsRole uint8

const (
	tlsServer tlsRole = iota
	tlsClient
)

// add extracts peer name from x509.Certificate's Subject.CommonName.
// It searches for the peer in cs and calls Connection.Establish.
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
	conn := p.cs[peerName]
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
	if p.cs[pn].status == StatusAlive {
		return errors.Errorf("connection between %s was already established", pn)
	}
	return nil
}
