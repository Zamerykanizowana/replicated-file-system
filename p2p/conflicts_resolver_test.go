package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func TestConflictsResolver_DetectAndResolveConflict(t *testing.T) {
	for name, testCase := range map[string]struct {
		currentTransactions []*protobuf.Message
		message             *protobuf.Message
		isDetected          bool
		isGreenLight        bool
	}{
		"no conflicting paths, no conflict detected": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "123", "/places", "Gimli", 0),
				newRequestMessage(t, protobuf.Request_CREATE, "321", "/stuff", "Gimli", 0),
			},
			message:    newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Aragorn", 0),
			isDetected: false,
		},
		"skip message host's own message conflict, don't detect conflict": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "123", "/places", "Gimli", 0),
				newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Gimli", 0),
			},
			message:    newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Gimli", 0),
			isDetected: false,
		},
		"if conflict is for the same peer on two of his transactions, don't detect conflict": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "321", "/somewhere", "Gimli", 0),
			},
			message:    newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Gimli", 0),
			isDetected: false,
		},
		"even if types differ, detect conflict": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "123", "/places", "Gimli", 0),
				newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Gimli", 0),
			},
			message:    newRequestMessage(t, protobuf.Request_MKDIR, "321", "/somewhere", "Aragorn", 0),
			isDetected: true,
		},
		"conflict resolved in favor of new message based on clock": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "123", "/places", "Gimli", 0),
				newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Gimli", 1),
			},
			message:      newRequestMessage(t, protobuf.Request_MKDIR, "321", "/somewhere", "Aragorn", 0),
			isDetected:   true,
			isGreenLight: true,
		},
		"conflict resolved in favor of ongoing transaction based on clock": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "123", "/places", "Gimli", 0),
				newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Gimli", 0),
			},
			message:      newRequestMessage(t, protobuf.Request_MKDIR, "321", "/somewhere", "Aragorn", 1),
			isDetected:   true,
			isGreenLight: false,
		},
		"conflict resolved in favor of new message based on peer name": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "123", "/places", "Gimli", 0),
				newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Aragorn", 0),
			},
			message:      newRequestMessage(t, protobuf.Request_MKDIR, "321", "/somewhere", "Gimli", 0),
			isDetected:   true,
			isGreenLight: true,
		},
		"conflict resolved in favor of ongoing transaction based on peer name": {
			currentTransactions: []*protobuf.Message{
				newRequestMessage(t, protobuf.Request_CREATE, "123", "/places", "Gimli", 0),
				newRequestMessage(t, protobuf.Request_CREATE, "333", "/somewhere", "Gimli", 0),
			},
			message:      newRequestMessage(t, protobuf.Request_MKDIR, "321", "/somewhere", "Aragorn", 0),
			isDetected:   true,
			isGreenLight: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolver := newConflictsResolver()

			trans := newTransactions()
			for _, msg := range testCase.currentTransactions {
				trans.Put(msg)
			}

			detected, greenLight := resolver.DetectAndResolveConflict(trans, testCase.message)
			assert.Equal(t, testCase.isDetected, detected)
			assert.Equal(t, testCase.isGreenLight, greenLight)
		})
	}
}

func newRequestMessage(
	t *testing.T,
	typ protobuf.Request_Type,
	tid, path, peerName string,
	clock uint64,
) *protobuf.Message {
	t.Helper()
	message, err := protobuf.NewRequestMessage(tid, peerName, &protobuf.Request{
		Type:     typ,
		Metadata: &protobuf.Request_Metadata{RelativePath: path},
		Clock:    clock,
	})
	require.NoError(t, err)
	return message
}
