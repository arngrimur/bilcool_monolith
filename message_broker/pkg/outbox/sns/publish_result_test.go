package sns

import (
	"testing"

	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/messaging"
)

func TestToPublishResult(t *testing.T) {
	ok := uuid.New()
	failed := uuid.New()
	okId := ok.String()
	failedId := failed.String()
	notAnEventId := "not-a-uuid"
	reason := "throttled"

	result := toPublishResult(&awssns.PublishBatchOutput{
		Successful: []types.PublishBatchResultEntry{{Id: &okId}, {Id: &notAnEventId}, {}},
		Failed:     []types.BatchResultErrorEntry{{Id: &failedId, Message: &reason}, {Message: &reason}},
	})

	require.Equal(t, []uuid.UUID{ok}, result.Successful)
	require.Len(t, result.Failed, 4)
	require.Equal(t, uuid.Nil, result.Failed[0].EventId, "a successful entry without an event id cannot be marked as emitted")
	require.Contains(t, result.Failed[0].Reason, notAnEventId)
	require.Equal(t, uuid.Nil, result.Failed[1].EventId)
	require.Equal(t, messaging.PublishFailure{EventId: failed, Reason: reason}, result.Failed[2])
	require.Equal(t, messaging.PublishFailure{Reason: reason}, result.Failed[3])
}

func TestToPublishResultEmpty(t *testing.T) {
	result := toPublishResult(&awssns.PublishBatchOutput{})
	require.Empty(t, result.Successful)
	require.Empty(t, result.Failed)
}
