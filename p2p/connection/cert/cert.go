package cert

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"

	"github.com/pkg/errors"
)

//go:embed peer.crt
var certPEM []byte

//go:embed peer.key
var keyPEM []byte

//go:embed ca.crt
var caPEM []byte

// Certificate parses peer's certificate and key into tls.Certificate.
func Certificate() (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse peer's cert and key pair")
	}
	return &cert, err
}

// CAPool returns x509.CertPool with a single parsed CA certificate.
func CAPool() (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}
	return pool, nil
}
