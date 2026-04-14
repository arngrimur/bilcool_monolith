package inbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	broker "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func TestWorker_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockEventConsumer(ctrl)

	consumer.EXPECT().RetrieveMessages(gomock.Any()).Return(nil, nil).AnyTimes()

	w := NewWorker(consumer, 2)
	w.Start(context.Background())
	w.Stop()
}

func TestWorker_StopBeforeStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockEventConsumer(ctrl)

	w := NewWorker(consumer, 1)
	require.NotPanics(t, func() { w.Stop() })
}

func TestWorker_ProcessesRetrievedMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockEventConsumer(ctrl)

	batch := []broker.Message{{ReceiptHandle: "r1"}}
	processed := make(chan struct{})

	consumer.EXPECT().RetrieveMessages(gomock.Any()).Return(batch, nil).MinTimes(1)
	consumer.EXPECT().ProcessMessages(gomock.Any(), batch).DoAndReturn(func(_ context.Context, _ []broker.Message) {
		select {
		case processed <- struct{}{}:
		default:
		}
	}).AnyTimes()

	w := NewWorker(consumer, 1)
	w.Start(context.Background())
	defer w.Stop()

	select {
	case <-processed:
	case <-time.After(2 * time.Second):
		t.Fatal("ProcessMessages was not called within timeout")
	}
}

func TestWorker_SkipsEmptyBatches(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockEventConsumer(ctrl)

	called := make(chan struct{}, 1)
	consumer.EXPECT().RetrieveMessages(gomock.Any()).DoAndReturn(func(_ context.Context) ([]broker.Message, error) {
		select {
		case called <- struct{}{}:
		default:
		}
		return nil, nil
	}).AnyTimes()
	consumer.EXPECT().ProcessMessages(gomock.Any(), gomock.Any()).Times(0)

	w := NewWorker(consumer, 1)
	w.Start(context.Background())
	defer w.Stop()

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("RetrieveMessages was not called")
	}

}

func TestWorker_ContinuesAfterRetrieveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockEventConsumer(ctrl)

	batch := []broker.Message{{ReceiptHandle: "r2"}}
	processed := make(chan struct{})

	gomock.InOrder(
		consumer.EXPECT().RetrieveMessages(gomock.Any()).Return(nil, errors.New("transient error")),
		consumer.EXPECT().RetrieveMessages(gomock.Any()).Return(batch, nil).MinTimes(1),
	)
	consumer.EXPECT().ProcessMessages(gomock.Any(), batch).DoAndReturn(func(_ context.Context, _ []broker.Message) {
		select {
		case processed <- struct{}{}:
		default:
		}
	}).AnyTimes()

	w := NewWorker(consumer, 1)
	w.Start(context.Background())
	defer w.Stop()

	select {
	case <-processed:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not recover after error")
	}
}

func TestWorker_MultipleWorkersProcessBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	consumer := NewMockEventConsumer(ctrl)

	batch := []broker.Message{{ReceiptHandle: "r3"}}
	processed := make(chan struct{}, 1)

	consumer.EXPECT().RetrieveMessages(gomock.Any()).Return(batch, nil).MinTimes(1)
	consumer.EXPECT().ProcessMessages(gomock.Any(), batch).DoAndReturn(func(_ context.Context, _ []broker.Message) {
		select {
		case processed <- struct{}{}:
		default:
		}
	}).AnyTimes()

	w := NewWorker(consumer, 3)
	w.Start(context.Background())
	defer w.Stop()

	select {
	case <-processed:
	case <-time.After(2 * time.Second):
		t.Fatal("ProcessMessages was not called with multiple workers")
	}
}
