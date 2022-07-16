package connection

import (
	"crypto/tls"
	_ "embed"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/p2p/cert"
)

// tlsConfig returns
func tlsConfig(tlsVersion uint16) *tls.Config {
	certificate, err := cert.Certificate()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	caPool, err := cert.CAPool()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	return &tls.Config{
		Certificates: []tls.Certificate{*certificate},
		RootCAs:      caPool,
		ClientCAs:    caPool,
		// This turns the whole thing to mTLS.
		ClientAuth: tls.RequireAndVerifyClientCert,
		// We don't want to verify the host, rather we're interested in peer name.
		InsecureSkipVerify: true,
		MinVersion:         tlsVersion,
		MaxVersion:         tlsVersion,
		// This might be useful for Wireshark debugging/analysis.
		KeyLogWriter: nil,
	}
}
