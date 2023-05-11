package jobq

import (
	"context"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
	"log"
)

type JobQ struct {
	subject string
	q       *pubsub.Topic
	sem     chan struct{}
	log     func(v ...any)
	logF    func(format string, v ...any)
}

func New(ctx context.Context, subject string, maxHandlers int) (*JobQ, error) {
	topic, err := pubsub.OpenTopic(ctx, "mem://"+subject)
	if err != nil {
		return nil, err
	}

	return &JobQ{
		subject,
		topic,
		make(chan struct{}, maxHandlers),
		log.Println,
		log.Printf,
	}, nil
}

func (j *JobQ) ChanSubscribe(ctx context.Context, out chan *pubsub.Message) error {
	sub, err := pubsub.OpenSubscription(ctx, j.subject)
	if err != nil {
		return err
	}
	go j.listen(ctx, sub, out)

	return nil
}

func (j *JobQ) listen(ctx context.Context, sub *pubsub.Subscription, out chan *pubsub.Message) {
recvLoop:
	for {
		msg, err := sub.Receive(ctx)
		if err != nil {
			j.logF("jobq receive: %v", err)
		}

		select {
		case <-ctx.Done():
			break recvLoop
		case j.sem <- struct{}{}:
		}

		go func() {
			defer func() { <-j.sem }() // frees up the channel for a new receiver
			defer msg.Ack()            // message must always be acknowledged

			// handle message
			out <- msg
		}()
	}

	j.block()
}

func (j *JobQ) block() {
	// we're no longer receiving messages. Wait to finish handling any
	// unacknowledged messages by totally acquiring the semaphore.
	for n := 0; n < cap(j.sem); n++ {
		j.sem <- struct{}{}
	}
}

func (j *JobQ) Publish(ctx context.Context, msg *pubsub.Message) error { return j.q.Send(ctx, msg) }
