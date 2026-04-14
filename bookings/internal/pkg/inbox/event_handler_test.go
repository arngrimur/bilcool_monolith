package inbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func makeUserCreatedMessage(userRef uuid.UUID, msgID string) brokerpostgres.Message {
	payload, _ := json.Marshal(authdomain.UserResponse{UserRef: userRef})
	return brokerpostgres.Message{
		ReceiptHandle: "handle-" + msgID,
		MessageBody: brokerpostgres.MessageBody{
			MessageId: msgID,
			Message: brokerpostgres.Event{
				Type:    authdomain.EventUserCreated,
				Payload: payload,
			},
		},
	}
}

func setupHandler(ctrl *gomock.Controller) (*EventHandler, *MocksqsClient, *domain.MockBookingsRepository) {
	sqs := NewMocksqsClient(ctrl)
	repo := domain.NewMockBookingsRepository(ctrl)
	sqs.EXPECT().VisibilityTimeout().Return(30).AnyTimes()
	return NewEventHandler(sqs, repo), sqs, repo
}

func TestProcessMessages_UserCreated_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	userRef := uuid.New()
	msg := makeUserCreatedMessage(userRef, "msg-1")

	repoMock.EXPECT().AddUser(gomock.Any(), userRef, "msg-1").Return(nil)
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{msg}).Return(1, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_RepoError_MessageNotDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	userRef := uuid.New()
	msg := makeUserCreatedMessage(userRef, "msg-2")

	repoMock.EXPECT().AddUser(gomock.Any(), userRef, "msg-2").Return(errors.New("db error"))
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{}).Return(0, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_InvalidPayload_MessageNotDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, _ := setupHandler(ctrl)

	msg := brokerpostgres.Message{
		ReceiptHandle: "handle-bad",
		MessageBody: brokerpostgres.MessageBody{
			MessageId: "msg-bad",
			Message: brokerpostgres.Event{
				Type:    authdomain.EventUserCreated,
				Payload: []byte("not-valid-json"),
			},
		},
	}

	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{}).Return(0, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_UnknownEventType_Skipped(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, _ := setupHandler(ctrl)

	msg := brokerpostgres.Message{
		ReceiptHandle: "handle-unknown",
		MessageBody: brokerpostgres.MessageBody{
			MessageId: "msg-unknown",
			Message: brokerpostgres.Event{
				Type:    "some.unknown.event",
				Payload: []byte(`{}`),
			},
		},
	}

	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{}).Return(0, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_DeleteError_NoReturnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	userRef := uuid.New()
	msg := makeUserCreatedMessage(userRef, "msg-3")

	repoMock.EXPECT().AddUser(gomock.Any(), userRef, "msg-3").Return(nil)
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{msg}).Return(0, errors.New("sqs error"))

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_EmptyMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, _ := setupHandler(ctrl)

	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{}).Return(0, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{})
	})
}

func TestProcessMessages_MultipleMessages_PartialFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	userRef1 := uuid.New()
	userRef2 := uuid.New()
	msg1 := makeUserCreatedMessage(userRef1, "msg-4")
	msg2 := makeUserCreatedMessage(userRef2, "msg-5")

	repoMock.EXPECT().AddUser(gomock.Any(), userRef1, "msg-4").Return(nil)
	repoMock.EXPECT().AddUser(gomock.Any(), userRef2, "msg-5").Return(errors.New("db error"))
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{msg1}).Return(1, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg1, msg2})
	})
}
