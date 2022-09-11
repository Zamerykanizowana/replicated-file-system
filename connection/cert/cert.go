package cert

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"os"

	"github.com/pkg/errors"
)

// Certificate parses peer's certificate and key into tls.Certificate.
func Certificate(certPath, keyPath string) (*tls.Certificate, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read: %s cert file", certPath)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read: %s key file", keyPath)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse peer's cert and key pair")
	}
	return &cert, err
}

// CAPool returns x509.CertPool with a single parsed CA certificate.
func CAPool(path string) (*x509.CertPool, error) {
	caPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read: %s CA file", path)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("failed to parse CA certificate")
	}
	return pool, nil
}
