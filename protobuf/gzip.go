package protobuf

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var compressionLevel = gzip.DefaultCompression

func SetCompression(cmp string) {
	switch cmp {
	case "NoCompression":
		compressionLevel = gzip.NoCompression
	case "BestSpeed":
		compressionLevel = gzip.BestSpeed
	case "BestCompression":
		compressionLevel = gzip.BestCompression
	case "DefaultCompression":
		compressionLevel = gzip.DefaultCompression
	case "HuffmanOnly":
		compressionLevel = gzip.HuffmanOnly
	default:
		log.Fatal().
			Str("compression_level", cmp).
			Msg("invalid compression level")
	}
}

func compress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(data)*80/100))
	w, err := gzip.NewWriterLevel(buf, compressionLevel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create zlib.Writer")
	}
	if _, err = w.Write(data); err != nil {
		return nil, errors.Wrap(err, "failed to write gzipped content to buffer")
	}
	if err = w.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close gzip.Writer")
	}
	return buf.Bytes(), err
}

func decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip.Reader for provided data")
	}
	data, err = io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read data through gzip.Reader")
	}
	if err = r.Close(); err != nil {
		return nil, errors.Wrap(err, "failed to close gzip.Reader, can't finalize the decompression")
	}
	return data, nil
}
