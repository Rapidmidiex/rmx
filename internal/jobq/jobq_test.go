package jobq_test

import (
	"context"
	"testing"

	"github.com/rapidmidiex/rmx/internal/jobq"
	"gocloud.dev/pubsub"
)

func TestJobQ(t *testing.T) {
	ctx := context.Background()
	subject := "jobq"
	ch := make(chan *pubsub.Message, 10)
	text := []byte("never gonna give you up")

	q, _ := jobq.New(ctx, subject, 5)

	t.Run("test jobq", func(t *testing.T) {
		_ = q.ChanSubscribe(ctx, ch)

		_ = q.Publish(ctx, &pubsub.Message{
			Body: text,
		})

		msg := <-ch
		if string(msg.Body) != string(text) {
			t.Fail()
		}
	})
}
