package jobq_test

import (
	"context"
	"testing"

	"github.com/hyphengolang/prelude/testing/is"
	"github.com/rapidmidiex/rmx/pkg/jobq"
	"gocloud.dev/pubsub"
)

func TestJobQ(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	subject := "jobq"
	ch := make(chan *pubsub.Message, 10)
	text := []byte("never gonna give you up")

	t.Run("test jobq", func(t *testing.T) {
		_, err := jobq.New(ctx, subject)
		is.NoErr(err)

		err = jobq.ChanSubscribe(ctx, subject, ch, 5)
		is.NoErr(err)

		err = jobq.AsyncSubscribe(ctx, subject, func(m *pubsub.Message) {
			is.Equal(string(m.Body), text)
		}, 5)
		is.NoErr(err)

		err = jobq.Publish(ctx, subject, &pubsub.Message{
			Body: text,
		})
		is.NoErr(err)

		msg := <-ch
		is.Equal(msg.Body, text)
	})
}
