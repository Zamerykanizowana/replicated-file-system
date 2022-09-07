package mirror

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"

	"github.com/Zamerykanizowana/replicated-file-system/config"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func NewMirror(conf *config.Paths) *Mirror {
	return &Mirror{dir: conf.MirrorDir}
}

type Mirror struct {
	dir string
}

func (m Mirror) Mirror(request *protobuf.Request) error {
	switch request.Type {
	case protobuf.Request_CREATE:
		_, err := os.OpenFile(
			m.path(request.Metadata.RelativePath),
			os.O_CREATE,
			os.FileMode(request.Metadata.Mode))
		return err
	case protobuf.Request_SYMLINK, protobuf.Request_LINK:
		return os.Symlink(m.path(request.Metadata.RelativePath), m.path(request.Metadata.NewRelativePath))
	case protobuf.Request_MKDIR:
		return os.Mkdir(
			m.path(request.Metadata.RelativePath),
			os.FileMode(request.Metadata.Mode))
	case protobuf.Request_RENAME:
		return os.Rename(
			m.path(request.Metadata.RelativePath),
			m.path(request.Metadata.NewRelativePath))
	case protobuf.Request_RMDIR, protobuf.Request_UNLINK:
		return os.Remove(m.path(request.Metadata.RelativePath))
	case protobuf.Request_SETATTR:
		return os.Chmod(
			m.path(request.Metadata.RelativePath),
			os.FileMode(request.Metadata.Mode))
	case protobuf.Request_WRITE:
		f, err := os.OpenFile(
			m.path(request.Metadata.RelativePath),
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0)
		if err != nil {
			return err
		}
		_, err = f.WriteAt(request.Content, request.Metadata.WriteOffset)
		return err
	case protobuf.Request_COPY_FILE_RANGE:
		src, err := syscall.Open(
			m.path(request.Metadata.RelativePath),
			syscall.O_RDONLY, 0)
		if err != nil {
			return err
		}
		dst, err := syscall.Open(
			m.path(request.Metadata.NewRelativePath),
			syscall.O_WRONLY, 0)
		if err != nil {
			return err
		}
		rOff, wOff := request.Metadata.ReadOffset, request.Metadata.WriteOffset
		written, err := unix.CopyFileRange(
			src, &rOff,
			dst, &wOff,
			int(request.Metadata.WriteLen), 0)
		if written != int(request.Metadata.WriteLen) {
			return errors.Errorf("should've written %d but only wrote %d to a file",
				request.Metadata.WriteLen, written)
		}
		return nil
	default:
		log.Panic().Msg("BUG: unknown protobuf.Request_Type")
	}
	return nil
}

func (m Mirror) Consult(request *protobuf.Request) *protobuf.Response {
	switch request.Type {
	case protobuf.Request_LINK, protobuf.Request_SYMLINK:
		oldExists, err := m.exists(request.Metadata.RelativePath)
		if err != nil {
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		newExists, err := m.exists(request.Metadata.NewRelativePath)
		if err != nil {
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		if !oldExists {
			return protobuf.NACK(protobuf.Response_ERR_DOES_NOT_EXIST,
				errors.Errorf("symlink source: %s does not exist",
					request.Metadata.RelativePath))
		}
		if newExists {
			return protobuf.NACK(protobuf.Response_ERR_ALREADY_EXISTS,
				errors.Errorf("symlink destination: %s already exist",
					request.Metadata.NewRelativePath))
		}
		return protobuf.ACK()
	case protobuf.Request_CREATE, protobuf.Request_MKDIR:
		exists, err := m.exists(request.Metadata.RelativePath)
		if err != nil {
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		if exists {
			return protobuf.NACK(protobuf.Response_ERR_ALREADY_EXISTS, errors.New("dir/file already exists"))
		}
		return protobuf.ACK()
	case protobuf.Request_RENAME:
		oldInfo, err := os.Stat(m.path(request.Metadata.RelativePath))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return protobuf.NACK(protobuf.Response_ERR_DOES_NOT_EXIST, err)
			}
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		newInfo, err := os.Stat(m.path(request.Metadata.NewRelativePath))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return protobuf.ACK()
			}
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		if oldInfo.Mode().Type() != newInfo.Mode().Type() {
			return protobuf.NACK(protobuf.Response_ERR_INVALID_MODE,
				errors.New("file types of old and new filename don't match"))
		}
		return protobuf.ACK()
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
	case protobuf.Request_UNLINK:
		if _, err := os.Stat(m.path(request.Metadata.RelativePath)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return protobuf.ACK()
			}
			return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
		}
		return protobuf.ACK()
	case protobuf.Request_WRITE:
		// TODO verify it!
		return protobuf.ACK()
	case protobuf.Request_COPY_FILE_RANGE:
		for _, path := range []string{
			request.Metadata.RelativePath,
			request.Metadata.NewRelativePath,
		} {
			info, err := os.Stat(m.path(path))
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return protobuf.NACK(protobuf.Response_ERR_DOES_NOT_EXIST,
						errors.Errorf("%s does not exist", path))
				}
				return protobuf.NACK(protobuf.Response_ERR_UNKNOWN, err)
			}
			if !info.Mode().IsRegular() {
				return protobuf.NACK(protobuf.Response_ERR_NOT_A_FILE,
					errors.Errorf("%s is not a regular file", path))
			}
		}
		if err := syscall.Access(m.path(request.Metadata.NewRelativePath), syscall.O_RDWR); err != nil {
			return protobuf.NACK(protobuf.Response_ERR_INVALID_MODE,
				errors.Wrap(err, "expected to have read-write access to the destination file"))
		}
		return protobuf.ACK()
	default:
		log.Panic().Msg("BUG: unknown protobuf.Request_Type")
	}
	return nil
}

func (m Mirror) exists(relativePath string) (bool, error) {
	if _, err := os.Stat(m.path(relativePath)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

func (m Mirror) path(relPath string) string {
	return filepath.Join(m.dir, relPath)
}
