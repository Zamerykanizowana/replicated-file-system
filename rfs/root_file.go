package rfs

import (
	"context"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/rs/zerolog/log"
)

func NewRfsFile(fd int) fs.FileHandle {
	log.Info().Int("fd: ", fd).Msg("From NewRfsFile: ")
	return &rfsFile{fs.NewLoopbackFile(fd)}
}

type rfsFile struct {
	fs.FileHandle
}

func (f *rfsFile) Read(ctx context.Context, buf []byte, off int64) (res fuse.ReadResult, errno syscall.Errno) {
	log.Info().Msg("Not logging - as Adam asked ;)")
	return res, 0
}
