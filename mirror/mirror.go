package mirror

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func NewMirror(conf *config.Paths) *Mirror {
	return &Mirror{conf: conf}
}

type Mirror struct {
	conf *config.Paths
}

func (m *Mirror) Mirror(request *protobuf.Request) error {
	switch request.Type {
	case protobuf.Request_CREATE:
		_, err := os.OpenFile(
			m.path(request.Metadata.RelativePath),
			os.O_CREATE,
			os.FileMode(request.Metadata.Mode))
		return err
	case protobuf.Request_LINK:
	case protobuf.Request_MKDIR:
	case protobuf.Request_RENAME:
	case protobuf.Request_RMDIR, protobuf.Request_UNLINK:
		return os.Remove(m.path(request.Metadata.RelativePath))
	case protobuf.Request_SETATTR:
		return os.Chmod(
			m.path(request.Metadata.RelativePath),
			os.FileMode(request.Metadata.Mode))
	case protobuf.Request_SYMLINK:
	case protobuf.Request_WRITE:
	default:
		log.Panic().Msg("BUG: unknown protobuf.Request_Type")
	}
	return nil
}

func (m *Mirror) Consult(request *protobuf.Request) *protobuf.Response {
	switch request.Type {
	case protobuf.Request_CREATE:
		if _, err := os.Stat(m.path(request.Metadata.RelativePath)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return protobuf.ACK()
			}
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		return protobuf.NACK(protobuf.Response_ERR_ALREADY_EXISTS, errors.New("file already exists"))
	case protobuf.Request_LINK:
	case protobuf.Request_MKDIR:
	case protobuf.Request_RENAME:
	case protobuf.Request_RMDIR:
		if pathInfo, err := os.Stat(m.path(request.Metadata.RelativePath)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return protobuf.NACK(protobuf.Response_ERR_DOES_NOT_EXIST, err)
			}

			if !pathInfo.IsDir() {
				return protobuf.NACK(protobuf.Response_ERR_NOT_A_DIRECTORY, nil)
			}
			return protobuf.ACK()
		}
	case protobuf.Request_SETATTR:
		if _, err := os.Stat(m.path(request.Metadata.RelativePath)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return protobuf.NACK(protobuf.Response_ERR_DOES_NOT_EXIST, err)
			}
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		return protobuf.ACK()
	case protobuf.Request_SYMLINK:
	case protobuf.Request_UNLINK:
		if _, err := os.Stat(m.path(request.Metadata.RelativePath)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return protobuf.ACK()
			}
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		return protobuf.ACK()
	case protobuf.Request_WRITE:
	default:
		log.Panic().Msg("BUG: unknown protobuf.Request_Type")
	}
	return nil
}

func (m *Mirror) path(relPath string) string {
	return filepath.Join(m.conf.MirrorDir, relPath)
}
