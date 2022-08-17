package mirror

import (
	"os"
	"path"

	"github.com/pkg/errors"

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
	}
	return nil
}

func (m *Mirror) Consult(request *protobuf.Request) (accept bool) {
	switch request.Type {
	case protobuf.Request_CREATE:
		if _, err := os.Stat(m.path(request.Metadata.RelativePath)); errors.Is(err, os.ErrNotExist) {
			return true
		}
		return false
	case protobuf.Request_LINK:
	case protobuf.Request_MKDIR:
	case protobuf.Request_RENAME:
	case protobuf.Request_RMDIR:
	case protobuf.Request_SETATTR:
	case protobuf.Request_SYMLINK:
	case protobuf.Request_UNLINK:
	case protobuf.Request_WRITE:
	}
	return
}

func (m *Mirror) path(relPath string) string {
	return path.Join(m.Conf.MirrorDir, relPath)
}
