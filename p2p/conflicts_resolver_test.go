package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Zamerykanizowana/replicated-file-system/protobuf"
)

func TestConflictsResolver_DetectAndResolveConflict(t *testing.T) {
	for name, test := range map[string]struct {
		currentTransactions []*protobuf.Message
		message             *protobuf.Message
		conflicts           bool
	}{
		"no conflicting paths, no conflict detected": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "321", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			})},
			message: mustNewRequestMessage(t, "333", "Aragorn", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/else"},
			}),
			conflicts: false,
		},
		"skip message host's own message conflict, don't detect conflict": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			})},
			message: mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			}),
			conflicts: false,
		},
		"if conflict is for the same peer on two of his transactions, don't detect conflict": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			})},
			message: mustNewRequestMessage(t, "321", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
				Clock:    1,
			}),
			conflicts: false,
		},
		"even if types differ, detect conflict": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			})},
			message: mustNewRequestMessage(t, "321", "Aragorn", &protobuf.Request{
				Type:     protobuf.Request_MKDIR,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			}),
			conflicts: true,
		},
		"conflict resolved in favor of new message based on clock": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			})},
			message: mustNewRequestMessage(t, "321", "Aragorn", &protobuf.Request{
				Type:     protobuf.Request_MKDIR,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
				Clock:    1,
			}),
			conflicts: false,
		},
		"conflict resolved in favor of ongoing transaction based on clock": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
				Clock:    1,
			})},
			message: mustNewRequestMessage(t, "321", "Aragorn", &protobuf.Request{
				Type:     protobuf.Request_MKDIR,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			}),
			conflicts: true,
		},
		"conflict resolved in favor of new message based on peer name": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Aragorn", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			})},
			message: mustNewRequestMessage(t, "321", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_MKDIR,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			}),
			conflicts: false,
		},
		"conflict resolved in favor of ongoing transaction based on peer name": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			})},
			message: mustNewRequestMessage(t, "321", "Aragorn", &protobuf.Request{
				Type:     protobuf.Request_MKDIR,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/somewhere"},
			}),
			conflicts: true,
		},
		"conflict is not detected if NewRelativePath exists and is different": {
			currentTransactions: []*protobuf.Message{mustNewRequestMessage(t, "333", "Gimli", &protobuf.Request{
				Type: protobuf.Request_SYMLINK,
				Metadata: &protobuf.Request_Metadata{
					RelativePath:    "/somewhere",
					NewRelativePath: "/else",
				},
			})},
			message: mustNewRequestMessage(t, "321", "Aragorn", &protobuf.Request{
				Type: protobuf.Request_SYMLINK,
				Metadata: &protobuf.Request_Metadata{
					RelativePath:    "/somewhere",
					NewRelativePath: "/out-there",
				},
			}),
			conflicts: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resolver := newConflictsResolver()

			trans := newTransactions()
			// Extra transaction, just to see if our looping is correct
			trans.Put(mustNewRequestMessage(t, "1234", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/places"},
			}))
			// Insert our transactions in the middle.
			for _, msg := range test.currentTransactions {
				trans.Put(msg)
			}
			// Extra transaction, just to see if our looping is correct
			trans.Put(mustNewRequestMessage(t, "1235", "Gimli", &protobuf.Request{
				Type:     protobuf.Request_CREATE,
				Metadata: &protobuf.Request_Metadata{RelativePath: "/places-else"},
			}))

			err := resolver.DetectAndResolveConflict(trans, test.message)
			if test.conflicts {
				require.Error(t, err)
				assert.ErrorIs(t, ErrTransactionConflict, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func mustNewRequestMessage(
	t *testing.T,
	tid, peerName string,
	request *protobuf.Request,
) *protobuf.Message {
	t.Helper()
	message, err := protobuf.NewRequestMessage(tid, peerName, request)
	require.NoError(t, err)
	return message
}
