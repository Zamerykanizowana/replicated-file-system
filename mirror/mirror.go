package mirror

import (
	"os"

	"github.com/pkg/errors"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

type Mirror struct{}

func (m *Mirror) Mirror(request *protobuf.Request) error {
	switch request.Type {
	case protobuf.Request_CREATE:
		_, err := os.Create(request.Metadata.RelativePath)
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
		if _, err := os.Stat(request.Metadata.RelativePath); errors.Is(err, os.ErrNotExist) {
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
