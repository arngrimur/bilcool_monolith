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

func makeUserCreatedMessage(userRef uuid.UUID, eventID uuid.UUID) brokerpostgres.Message {
	payload, _ := json.Marshal(authdomain.UserResponse{UserRef: userRef})
	return brokerpostgres.Message{
		ReceiptHandle: "handle-" + eventID.String(),
		MessageBody: brokerpostgres.MessageBody{
			MessageId: "sqs-" + eventID.String(),
			Message: brokerpostgres.Event{
				EventId: eventID,
				Type:    authdomain.EventUserCreated,
				Payload: payload,
			},
		},
	}
}

func makeUserDeletedMessage(userRef uuid.UUID, eventID uuid.UUID) brokerpostgres.Message {
	payload, _ := json.Marshal(authdomain.UserResponse{UserRef: userRef})
	return brokerpostgres.Message{
		ReceiptHandle: "handle-" + eventID.String(),
		MessageBody: brokerpostgres.MessageBody{
			MessageId: "sqs-" + eventID.String(),
			Message: brokerpostgres.Event{
				EventId: eventID,
				Type:    authdomain.EventUserDeleted,
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
	eventID := uuid.New()
	msg := makeUserCreatedMessage(userRef, eventID)

	repoMock.EXPECT().AddUser(gomock.Any(), msg).Return(nil)
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{msg}).Return(1, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_UserDeleted_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	userRef := uuid.New()
	eventID := uuid.New()
	msg := makeUserDeletedMessage(userRef, eventID)

	repoMock.EXPECT().DeleteUser(gomock.Any(), msg).Return(nil)
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{msg}).Return(1, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_UserDeleted_RepoError_MessageNotDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	userRef := uuid.New()
	eventID := uuid.New()
	msg := makeUserDeletedMessage(userRef, eventID)

	repoMock.EXPECT().DeleteUser(gomock.Any(), msg).Return(errors.New("db error"))
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{}).Return(0, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_RepoError_MessageNotDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	userRef := uuid.New()
	eventID := uuid.New()
	msg := makeUserCreatedMessage(userRef, eventID)

	repoMock.EXPECT().AddUser(gomock.Any(), msg).Return(errors.New("db error"))
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{}).Return(0, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg})
	})
}

func TestProcessMessages_InvalidPayload_MessageNotDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	handler, sqsMock, repoMock := setupHandler(ctrl)

	msg := brokerpostgres.Message{
		ReceiptHandle: "handle-bad",
		MessageBody: brokerpostgres.MessageBody{
			MessageId: "msg-bad",
			Message: brokerpostgres.Event{
				EventId: uuid.New(),
				Type:    authdomain.EventUserCreated,
				Payload: []byte("not-valid-json"),
			},
		},
	}

	repoMock.EXPECT().AddUser(gomock.Any(), msg).Return(errors.New("invalid json"))
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
				EventId: uuid.New(),
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
	eventID := uuid.New()
	msg := makeUserCreatedMessage(userRef, eventID)

	repoMock.EXPECT().AddUser(gomock.Any(), msg).Return(nil)
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
	eventID1 := uuid.New()
	eventID2 := uuid.New()
	msg1 := makeUserCreatedMessage(userRef1, eventID1)
	msg2 := makeUserCreatedMessage(userRef2, eventID2)

	repoMock.EXPECT().AddUser(gomock.Any(), msg1).Return(nil)
	repoMock.EXPECT().AddUser(gomock.Any(), msg2).Return(errors.New("db error"))
	sqsMock.EXPECT().DeleteMessages(gomock.Any(), []brokerpostgres.Message{msg1}).Return(1, nil)

	require.NotPanics(t, func() {
		handler.ProcessMessages(context.Background(), []brokerpostgres.Message{msg1, msg2})
	})
}
