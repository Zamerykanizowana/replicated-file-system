package tlsconf

import (
	"crypto/tls"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/p2p/cert"
)

// Default returns tls.Config with sensible defaults, prepared for
// p2p communication, to achieve mTLS the crucial parameters are:
// - RootCAs and ClientCAs have to contain the same CA crt.
// - ClientAuth has to be set to tls.RequireAndVerifyClientCert
//	 (we're doing the same thing to client as we do to the server)
// - InsecureSkipVerify has to be set to true, we don't really care
//   what host are we connecting to, as long as the cert and peer name checks out.
func Default(tlsVersion uint16) *tls.Config {
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
