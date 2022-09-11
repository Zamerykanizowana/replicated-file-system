package tlsconf

import (
	"crypto/tls"

	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/connection/cert"
)

// Default returns tls.Config with sensible defaults, prepared for
// p2p communication, to achieve mTLS the crucial parameters are:
//   - RootCAs and ClientCAs have to contain the same CA crt.
//   - ClientAuth has to be set to tls.RequireAndVerifyClientCert
//     (we're doing the same thing to client as we do to the server)
//   - InsecureSkipVerify has to be set to true, we don't really care
//     what host are we connecting to, as long as the cert and peer name checks out.
func Default(conf config.TLS) *tls.Config {
	certificate, err := cert.Certificate(conf.CertPath, conf.KeyPath)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	caPool, err := cert.CAPool(conf.CAPath)
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
		MinVersion:         conf.GetTLSVersion(),
		MaxVersion:         conf.GetTLSVersion(),
		// This might be useful for Wireshark debugging/analysis.
		KeyLogWriter: nil,
	}
}
