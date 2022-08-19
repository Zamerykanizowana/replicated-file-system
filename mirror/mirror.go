package mirror

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type Mirror struct {
	Conf *config.Paths
}

func (m *Mirror) Mirror(request *protobuf.Request) error {
	switch request.Type {
	case protobuf.Request_CREATE:
		_, err := os.Create(m.path(request.Metadata.RelativePath))
		return err
	case protobuf.Request_LINK:
	case protobuf.Request_MKDIR:
	case protobuf.Request_RENAME:
	case protobuf.Request_RMDIR:
	case protobuf.Request_SETATTR:
	case protobuf.Request_SYMLINK:
	case protobuf.Request_UNLINK:
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
			var errno protobuf.Response_Error
			switch {
			case errors.Is(err, os.ErrNotExist):
				errno = protobuf.Response_ERR_ALREADY_EXISTS
			default:
				errno = protobuf.Response_ERR_UNKNOWN
			}
			return protobuf.NACK(errno, err)
		}
		return protobuf.ACK()
	case protobuf.Request_LINK:
	case protobuf.Request_MKDIR:
	case protobuf.Request_RENAME:
	case protobuf.Request_RMDIR:
	case protobuf.Request_SETATTR:
	case protobuf.Request_SYMLINK:
	case protobuf.Request_UNLINK:
	case protobuf.Request_WRITE:
	default:
		log.Panic().Msg("BUG: unknown protobuf.Request_Type")
	}
	return nil
}

func (m *Mirror) path(relPath string) string {
	return path.Join(m.Conf.MirrorDir, relPath)
}
