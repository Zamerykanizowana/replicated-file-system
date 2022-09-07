package mirror

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func TestMirror_CopyFileRange(t *testing.T) {
	m := Mirror{dir: t.TempDir()}
	src, err := os.OpenFile(filepath.Join(m.dir, "source"), os.O_CREATE|os.O_WRONLY, os.FileMode(0666))
	require.NoError(t, err)
	dst, err := os.OpenFile(filepath.Join(m.dir, "destination"), os.O_CREATE|os.O_RDONLY, os.FileMode(0666))
	require.NoError(t, err)

	var writeOffset, readOffset, writeLen int64 = 3, 6, 5

	content := []byte("this is just a test!")
	n, err := src.Write(content)
	require.NoError(t, err)
	require.Equal(t, n, len(content))

	err = m.Mirror(&protobuf.Request{
		Type: protobuf.Request_COPY_FILE_RANGE,
		Metadata: &protobuf.Request_Metadata{
			RelativePath:    "source",
			NewRelativePath: "destination",
			WriteOffset:     writeOffset,
			ReadOffset:      readOffset,
			WriteLen:        writeLen,
		},
	})
	require.NoError(t, err)

	result := make([]byte, writeLen)
	_, err = dst.ReadAt(result, writeOffset)
	require.NoError(t, err)
	expected := content[int(readOffset) : readOffset+writeLen]
	assert.Equal(t, expected, result)
}

func TestConsult_CopyFileRange(t *testing.T) {
	m := Mirror{dir: t.TempDir()}
	for name, perm := range map[string]uint32{
		"read-only":  0444,
		"read-write": 0666,
	} {
		_, err := os.OpenFile(filepath.Join(m.dir, name), os.O_CREATE, os.FileMode(perm))
		require.NoError(t, err)
	}

	for name, test := range map[string]struct {
		Metadata      *protobuf.Request_Metadata
		ExpectedError protobuf.Response_Error
	}{
		"ack": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath:    "read-only",
				NewRelativePath: "read-write",
			},
		},
		"not existing file": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath:    "read-write",
				NewRelativePath: "not-existing",
			},
			ExpectedError: protobuf.Response_ERR_DOES_NOT_EXIST,
		},
		"not a file": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath: "read-write",
				// This will be interpreted as the tmp dir created for the test.
				NewRelativePath: "",
			},
			ExpectedError: protobuf.Response_ERR_NOT_A_FILE,
		},
		"invalid mode for destination": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath:    "read-only",
				NewRelativePath: "read-only",
			},
			ExpectedError: protobuf.Response_ERR_INVALID_MODE,
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := &protobuf.Request{
				Type:     protobuf.Request_COPY_FILE_RANGE,
				Metadata: test.Metadata,
			}

			resp := m.Consult(req)

			if test.ExpectedError > 0 {
				require.Equal(t, protobuf.Response_NACK, resp.Type)
				assert.Equal(t, test.ExpectedError, *resp.Error)
			} else {
				require.Equal(t, protobuf.Response_ACK, resp.Type)
			}
		})
	}
}
