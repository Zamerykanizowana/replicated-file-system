package mirror

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func TestMirror_CopyFileRange(t *testing.T) {
	m := Mirror{dir: t.TempDir()}
	src, err := os.OpenFile(m.path("source"), os.O_CREATE|os.O_WRONLY, os.FileMode(0666))
	require.NoError(t, err)
	dst, err := os.OpenFile(m.path("destination"), os.O_CREATE|os.O_RDONLY, os.FileMode(0666))
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
		_, err := os.OpenFile(m.path(name), os.O_CREATE, os.FileMode(perm))
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

func TestMirror_Rename(t *testing.T) {
	m := Mirror{dir: t.TempDir()}
	f, err := os.OpenFile(m.path("old"), os.O_CREATE|os.O_WRONLY, os.FileMode(0666))
	require.NoError(t, err)
	content := []byte("hey what a test it is!")
	_, err = f.Write(content)
	require.NoError(t, err)

	err = m.Mirror(&protobuf.Request{
		Type: protobuf.Request_RENAME,
		Metadata: &protobuf.Request_Metadata{
			RelativePath:    "old",
			NewRelativePath: "new",
		},
	})
	require.NoError(t, err)

	f, err = os.Open(m.path("new"))
	require.NoError(t, err)
	data, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestConsult_Rename(t *testing.T) {
	m := Mirror{dir: t.TempDir()}
	require.NoError(t, os.Mkdir(m.path("old-directory"), 0))
	_, err := os.OpenFile(m.path("old"), os.O_CREATE, 0)
	require.NoError(t, err)
	_, err = os.OpenFile(m.path("new"), os.O_CREATE, 0)
	require.NoError(t, err)

	for name, test := range map[string]struct {
		Metadata      *protobuf.Request_Metadata
		ExpectedError protobuf.Response_Error
	}{
		"old doesn't exist": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath:    "not-exists",
				NewRelativePath: "new",
			},
			ExpectedError: protobuf.Response_ERR_DOES_NOT_EXIST,
		},
		"new doesn't exist": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath:    "old",
				NewRelativePath: "not-exists",
			},
		},
		"types don't match": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath:    "old-directory",
				NewRelativePath: "new",
			},
			ExpectedError: protobuf.Response_ERR_INVALID_MODE,
		},
		"types match": {
			Metadata: &protobuf.Request_Metadata{
				RelativePath:    "old",
				NewRelativePath: "new",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			resp := m.Consult(&protobuf.Request{
				Type:     protobuf.Request_RENAME,
				Metadata: test.Metadata,
			})

			if test.ExpectedError > 0 {
				require.Equal(t, protobuf.Response_NACK, resp.Type)
				assert.Equal(t, test.ExpectedError, *resp.Error)
			} else {
				require.Equal(t, protobuf.Response_ACK, resp.Type)
			}
		})
	}
}
