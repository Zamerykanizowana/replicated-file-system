package rfs

import (
	"context"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/Zamerykanizowana/replicated-file-system/connection"
	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func replicate(ctx context.Context, req *protobuf.Request, rep Replicator, mir Mirror) syscall.Errno {
	if resp := mir.Consult(req); resp.GetType() == protobuf.Response_NACK {
		return errnoFromProtobufError(*resp.Error)
	}

	if err := rep.Replicate(ctx, req); err != nil {
		log.Err(err).
			Object("request", req).
			Msgf("failed to replicate %s request", req.Type)
		switch errors.Cause(err) {
		case context.Canceled:
			return syscall.ECANCELED
		case context.DeadlineExceeded:
			return syscall.ETIMEDOUT
		case connection.ErrPeerIsDown:
			return syscall.EHOSTDOWN
		default:
			return PermissionDenied
		}
	}
	return NoError
}

func errnoFromProtobufError(e protobuf.Response_Error) syscall.Errno {
	switch e {
	case protobuf.Response_ERR_UNKNOWN:
		return PermissionDenied
	case protobuf.Response_ERR_ALREADY_EXISTS:
		return syscall.EEXIST
	case protobuf.Response_ERR_DOES_NOT_EXIST:
		return syscall.ENOENT
	case protobuf.Response_ERR_TRANSACTION_CONFLICT:
		return PermissionDenied
	case protobuf.Response_ERR_NOT_A_DIRECTORY:
		return syscall.ENOTDIR
	case protobuf.Response_ERR_NOT_A_FILE:
		return syscall.EBADF
	case protobuf.Response_ERR_INVALID_MODE:
		return syscall.EINVAL
	default:
		log.Error().Stringer("error", e).Msg("BUG: Unsupported error")
		return PermissionDenied
	}
}
