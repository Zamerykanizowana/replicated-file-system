package connection

import (
	"crypto/tls"
	"net"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewPool(
	address, name string,
	conf *config.Connection,
	pcs []*config.Peer,
) *Pool {
	msgs := make(chan message, conf.MessageBufferSize)

	whitelist := make(map[string]struct{}, len(pcs))
	cs := make(map[string]*Connection, len(whitelist))
	for _, p := range pcs {
		whitelist[p.Name] = struct{}{}
		cs[p.Name] = NewConnection(
			p.Name, p.Address,
			conf.SendRecvTimeout,
			msgs)
	}

	pool := &Pool{
		net:       conf.Network,
		whitelist: whitelist,
		name:      name,
		cs:        cs,
		msgs:      msgs,
	}

	tlsConf := tlsConfig(conf.GetTLSVersion())
	tlsConf.VerifyConnection = pool.VerifyConnection

	pool.dialer = &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: conf.DialTimeout},
		Config:    tlsConf}

	var err error
	pool.listener, err = tls.Listen(pool.net, address, tlsConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create TLS listener")
	}

	return pool
}

type (
	Pool struct {
		net           string
		whitelist     map[string]struct{}
		dialer        *tls.Dialer
		listener      net.Listener
		name          string
		cs            map[string]*Connection
		msgs          chan message
		backoffConfig *config.Backoff
	}
	message struct {
		data []byte
		err  error
	}
)

func (p *Pool) Run() (err error) {
	if err = p.listen(); err != nil {
		return
	}
	go p.dialAll()
	return
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
				mErr[conn.peerName] = err
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

func (p *Pool) listen() error {
	go p.acceptConnections()

	for _, c := range p.cs {
		go c.Listen()
	}
	return nil
}

func (p *Pool) acceptConnections() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			log.Err(err).Msg("failed to accept incoming connection")
			continue
		}
		go p.add(conn)
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
	backoff := NewBackoff(p.backoffConfig)
	for {
		if c.status == StatusAlive {
			backoff.Reset()
			c.WaitForClosed()
		}
		netConn, err := p.dialer.Dial(p.net, c.addr)
		if err != nil {
			c.log.Err(err).Msg("failed to dial connection")
			backoff.Next()
			continue
		}
		if added := p.add(netConn); !added {
			backoff.Next()
		}
	}
}

// add extracts peer name from x509.Certificate's Subject.SerialNumber.
// It searches for the peer in cs and calls Connection.Establish.
func (p *Pool) add(netConn net.Conn) (added bool) {
	tlsConn, isTLS := netConn.(*tls.Conn)
	if !isTLS {
		log.Error().
			Stringer("remote", netConn.RemoteAddr()).
			Msg("closing non TLS connection")
		closeConn(netConn)
		return
	}

	// At this point if this cert does not exist,
	// we have done something very wrong code wise, thus no nil checks.
	peerName := tlsConn.ConnectionState().PeerCertificates[0].Subject.SerialNumber
	conn := p.cs[peerName]
	if err := conn.Establish(netConn); err != nil {
		log.Debug().Err(err).Msg("closing connection")
		closeConn(netConn)
		return
	}
	return true
}

// VerifyConnection checks peer certificates, it expects a single cert
// containing in the subject CN of the whitelisted peer.
func (p *Pool) VerifyConnection(state tls.ConnectionState) error {
	if len(state.PeerCertificates) != 1 {
		return errors.Errorf("expected exactly one certificate, got %d", len(state.PeerCertificates))
	}
	var pn string
	for _, crt := range state.PeerCertificates {
		pn = crt.Subject.SerialNumber
		if _, ok := p.whitelist[pn]; !ok {
			return errors.Errorf(
				"CN: %s is not authorized to participate in the peer network", pn)
		}
	}
	if p.cs[pn].status == StatusAlive {
		return errors.Errorf("connection between %s was already established", pn)
	}
	return nil
}
