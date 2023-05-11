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
		t.Log("hello")
		q, err := jobq.New(ctx, subject, 5)
		is.NoErr(err)

		err = q.ChanSubscribe(ctx, ch)
		is.NoErr(err)

		err = q.Publish(ctx, &pubsub.Message{
			Body: text,
		})
		is.NoErr(err)

		msg := <-ch
		if string(msg.Body) != string(text) {
			t.Fail()
		}
	})
}
