package config

import (
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

func (c Config) Validate() (err error) {
	if len(c.Peers) == 0 {
		err = multierr.Append(err, errors.New("provide at least one peer config"))
	}
	for _, p := range c.Peers {
		if pErr := p.Validate(); pErr != nil {
			err = multierr.Append(err, errors.Wrapf(pErr, "peer validation failed for: %s", p))
		}
	}
	if cErr := c.Connection.Validate(); cErr != nil {
		err = multierr.Append(err, errors.Wrap(cErr, "connection config failed validation"))
	}
	return
}

func (p Peer) Validate() (err error) {
	if len(p.Address) == 0 {
		err = multierr.Append(err, errors.New("address must not be empty"))
	}
	if len(p.Name) == 0 {
		err = multierr.Append(err, errors.New("name must not be empty"))
	}
	return
}

func (c Connection) Validate() (err error) {
	if _, valid := tlsVersions[c.TLSVersion]; !valid {
		versionsSlice := make([]string, 0, len(tlsVersions))
		for v := range tlsVersions {
			versionsSlice = append(versionsSlice, v)
		}
		err = multierr.Append(err, errors.Errorf(
			"invalid TLS version provided: %s, valid TLS versions: %v",
			c.TLSVersion, versionsSlice))
	}
	if c.MessageBufferSize == 0 {
		err = multierr.Append(err, errors.New("message buffer size must be greater than zero,"+
			" bear in mind small values might influence performance and cause timeouts"))
	}
	if c.SendRecvTimeout < 0 {
		err = multierr.Append(err, errors.New("send and recv timeout must not be negative"))
	}
	if c.HandshakeTimeout < 0 {
		err = multierr.Append(err, errors.New("handshake timeout must not be negative"))
	}
	if len(c.Network) == 0 {
		err = multierr.Append(err, errors.New("network must not be empty"))
	}
	if dErr := c.DialBackoff.Validate(); dErr != nil {
		err = multierr.Append(err, errors.Wrap(dErr, "dial_backoff config failed validation"))
	}
	if len(c.Compression) == 0 {
		err = multierr.Append(err, errors.New("compression must not be empty"))
	}
	return nil
}

func (b Backoff) Validate() error {
	if b.Factor >= 1 {
		return errors.New("backoff factor must be greater or equal than one")
	}
	if b.MaxFactorJitter < 0 {
		return errors.New("max factor jitter must be greater than zero")
	}
	if b.Initial <= 0 {
		return errors.New("initial backoff must be greater or equal than zero")
	}
	if b.Max <= 0 {
		return errors.New("max backoff must be greater or equal than zero")
	}
	if b.Initial > b.Max {
		return errors.New("initial backoff must be smaller than or equal to max backoff")
	}
	return nil
}
