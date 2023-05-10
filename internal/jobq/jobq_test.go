package jobq_test

import (
	"context"
	"testing"

	"github.com/rapidmidiex/rmx/internal/jobq"
	"gocloud.dev/pubsub"
)

func TestJobQ(t *testing.T) {
	ctx := context.Background()
	ch := make(chan *pubsub.Message, 10)
	text := []byte("never gonna give you up")

	q := jobq.New(5)

	t.Run("test JobQ publish", func(t *testing.T) {
		q.ChanSubscribe(ctx, ch)

		_ = q.Publish(ctx, &pubsub.Message{
			Body: text,
		})

		msg := <-ch
		if string(msg.Body) != string(text) {
			t.Fail()
		}
	})
}
