package jobq

import (
	"context"
	"log"
	"time"

	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/mempubsub"
)

type JobQ struct {
	q    *pubsub.Topic
	sem  chan struct{}
	log  func(v ...any)
	logF func(format string, v ...any)
}

func New(maxHandlers int) *JobQ {
	return &JobQ{
		mempubsub.NewTopic(),
		make(chan struct{}, maxHandlers),
		log.Println,
		log.Printf,
	}
}

func (j *JobQ) ChanSubscribe(ctx context.Context, out chan *pubsub.Message) {
	sub := mempubsub.NewSubscription(j.q, 1*time.Minute)
	go j.listen(ctx, sub, out)
	j.block()
}

func (j *JobQ) listen(ctx context.Context, sub *pubsub.Subscription, out chan *pubsub.Message) {
	for {
		msg, err := sub.Receive(ctx)
		if err != nil {
			j.logF("jobq receive: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case j.sem <- struct{}{}:
		}

		go func() {
			defer func() { <-j.sem }() // frees up the channel for a new receiver
			defer msg.Ack()            // message must always be acknowledged

			// handle message
			out <- msg
		}()
	}
}

func (j *JobQ) block() {
	// we're no longer receiving messages. Wait to finish handling any
	// unacknowledged messages by totally acquiring the semaphore.
	for n := 0; n < cap(j.sem); n++ {
		j.sem <- struct{}{}
	}
}

func (j *JobQ) Publish(ctx context.Context, msg *pubsub.Message) error { return j.q.Send(ctx, msg) }
