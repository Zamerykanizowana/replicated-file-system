package config

import (
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//go:embed config.json
var defaultConfig []byte

func Default() *Config {
	return mustUnmarshalConfig(defaultConfig)
}

func Read(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal().Err(err).
			Str("filepath", path).
			Msg("failed to open config file")
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		log.Fatal().Err(err).
			Str("filepath", path).
			Msg("failed to read config file contents")
	}
	return mustUnmarshalConfig(raw)
}

type (
	Config struct {
		Connection Connection `json:"connection"`
		Peers      []*Peer    `json:"peers"`
		Filesystem Filesystem `json:"filesystem"`
		Logging    Logging    `json:"logging"`
	}

	Peer struct {
		// Name must be a unique identifier across all peers.
		Name string `json:"name"`
		// Address should be in form of a host:port, without the network scheme.
		Address string `json:"address"`
	}

	Filesystem struct {
		FuseDir      string        `json:"fuse_dir"`
		MirrorDir    string        `json:"mirror_dir"`
		EntryTimeout time.Duration `json:"entry_timeout"`
		AttrTimeout  time.Duration `json:"attr_timeout"`
	}

	Connection struct {
		DialBackoff *Backoff `json:"dial_backoff"`
		// TLSVersion describes both max and mind TLS version in the tls.Config.
		TLSVersion string `json:"tls_version"`
		// MessageBufferSize is the buffer of the global message channel onto which
		// goroutines listening on peer Connection push received messages.
		MessageBufferSize uint `json:"message_buffer_size"`
		// SendRecvTimeout sets the timeout for Recv and Send operations.
		SendRecvTimeout time.Duration `json:"send_recv_timeout"`
		// HandshakeTimeout sets the timeout for handshakes.
		HandshakeTimeout time.Duration `json:"handshake_timeout"`
		// Network is the transport scheme string, e.g. 'tcp'.
		Network string `json:"network"`
		// Compression defines content compression for gzip, for more details go to protobuf/gzip.go.
		Compression string `json:"compression"`
	}

	Backoff struct {
		// Factor is the multiplying factor for each increment step.
		Factor float64 `json:"factor"`
		// MaxFactorJitter is the maximum factor jitter expressed in %.
		// A value of 0.2 means we'll modify factor by 20%.
		// Setting it to 0 will effectively turn jitter off.
		MaxFactorJitter float64 `json:"max_factor_jitter"`
		// Initial sets the initial value of the Backoff, which is not subject to jitter.
		Initial time.Duration `json:"initial"`
		// Max sets the maximum value after which reaching Backoff will no longer
		// be increased.
		Max time.Duration `json:"max"`
	}

	Logging struct {
		Level string `json:"level"`
	}
)

func (c Config) MarshalZerologObject(e *zerolog.Event) {
	e.Object("connection", c.Connection).
		Object("paths", c.Filesystem).
		Interface("peers", c.Peers).
		Object("logging", c.Logging)
}

func (f Filesystem) MarshalZerologObject(e *zerolog.Event) {
	e.Str("fuse_dir", f.FuseDir).
		Str("mirror_dir", f.MirrorDir).
		Stringer("attr_timeout", f.AttrTimeout).
		Stringer("entry_timeout", f.EntryTimeout)
}

func (p Peer) MarshalZerologObject(e *zerolog.Event) {
	e.Str("name", p.Name).
		Str("address", p.Address)
}

func (c Connection) MarshalZerologObject(e *zerolog.Event) {
	e.Object("backoff", c.DialBackoff).
		Str("tls_version", c.TLSVersion).
		Uint("message_buffer_size", c.MessageBufferSize).
		Stringer("send_recv_timeout", c.SendRecvTimeout).
		Stringer("handshake_timeout", c.HandshakeTimeout).
		Str("network", c.Network).
		Str("compression", c.Compression)
}

func (b Backoff) MarshalZerologObject(e *zerolog.Event) {
	e.Float64("factor", b.Factor).
		Float64("max_factor_jitter", b.MaxFactorJitter).
		Stringer("initial", b.Initial).
		Stringer("max", b.Max)
}

func (l Logging) MarshalZerologObject(e *zerolog.Event) {
	e.Str("level", l.Level)
}

var tlsVersions = map[string]uint16{
	"1.0": tls.VersionTLS10,
	"1.1": tls.VersionTLS11,
	"1.2": tls.VersionTLS12,
	"1.3": tls.VersionTLS13,
}

func (c Connection) GetTLSVersion() uint16 {
	return tlsVersions[c.TLSVersion]
}

func mustUnmarshalConfig(raw []byte) *Config {
	var (
		conf Config
		err  error
	)
	if err = json.Unmarshal(raw, &conf); err != nil {
		log.Fatal().Err(err).Msg("failed to read config.json")
	}
	for _, path := range []*string{&conf.Filesystem.FuseDir, &conf.Filesystem.MirrorDir} {
		*path, err = expandHome(*path)
		if err != nil {
			log.Fatal().Err(err).
				Str("path", *path).
				Msg("failed to expand home")
		}
	}
	if err = conf.Validate(); err != nil {
		log.Fatal().Err(err).Msg("validation failed for config")
	}

	return &conf
}

func expandHome(path string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", errors.Wrap(err, "failed to expand home")
	}

	if path == "~" {
		return "", errors.New("Mirroring home directory directly is not allowed!")
	} else if strings.HasPrefix(path, "~/") {
		return filepath.Join(usr.HomeDir, path[2:]), nil
	} else {
		return path, nil
	}
}
