package connection

import (
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
)

func NewPool(network, address, name string, pcs []*config.Peer) *Pool {
	msgs := make(chan message, 10000)
	cs := make(map[string]*Connection, len(whitelist))
	for _, p := range pcs {
		cs[p.Name] = NewConnection(p.Name, p.Address, msgs)
	}
	return &Pool{
		net:  network,
		addr: address,
		name: name,
		cs:   cs,
		// TODO make it configurable.
		msgs:        msgs,
		dialTimeout: 15 * time.Second,
	}
}

type (
	Pool struct {
		net         string
		addr        string
		name        string
		cs          map[string]*Connection
		msgs        chan message
		dialTimeout time.Duration
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
	)
	wg.Add(len(p.cs))
	for _, c := range p.cs {
		go func(conn *Connection) {
			if err := conn.Send(data); err != nil {
				if mErr == nil {
					mErr = make(SendMultiErr)
				}
				mErr[conn.peerName] = err
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
	listener, err := net.Listen(p.net, p.addr)
	if err != nil {
		return errors.Wrap(err, "failed to start listening for new connections")
	}
	go p.acceptConnections(listener)

	for _, c := range p.cs {
		go c.Listen()
	}
	return nil
}

func (p *Pool) acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Err(err).Msg("failed to accept incoming connection")
			continue
		}
		go p.add(conn)
	}
}

func (p *Pool) dialAll() {
	for _, conn := range p.cs {
		// We don't want to dial peers which are already connected.
		// If conn is nil we've introduced a serious bug and should be punished.
		// Connection struct is intended to be created once and only the underlying
		// net.Conn might change in any way (including nillable).
		if conn.status == StatusAlive {
			continue
		}
		go p.dial(conn)
	}
}

func (p *Pool) dial(c *Connection) {
	if c.status == StatusAlive {
		return
	}
	d := net.Dialer{Timeout: p.dialTimeout}
	conn, err := d.Dial(p.net, c.addr)
	if err != nil {
		c.log.Err(err).Msg("failed to dial connection")
		return
	}
	p.add(conn)
}

func (p *Pool) add(conn net.Conn) {
	peerName, err := handshake(conn, p.name)
	if err != nil {
		closeConn(conn)
		log.Debug().Err(err).Msg("connection handshake failed")
		return
	}
	if err = p.cs[peerName].Establish(conn); err != nil {
		log.Debug().Err(err).Msg("closing connection")
		closeConn(conn)
	}
}
