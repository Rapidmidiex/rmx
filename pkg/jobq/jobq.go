package jobq

import (
	"context"
	"log"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

type JobQ struct {
	sub  string
	q    *pubsub.Topic
	sem  chan struct{}
	log  func(v ...any)
	logF func(format string, v ...any)
}

type msgHandler func(*pubsub.Message)

type subscriptionType int

const (
	asyncSubscription = subscriptionType(iota)
	chanSubscription
)

type subscription struct {
	conn *pubsub.Subscription
	mcb  msgHandler
	mch  chan *pubsub.Message
	typ  subscriptionType
}

func New(ctx context.Context, subject string, maxHandlers int) (*JobQ, error) {
	url := "mem://" + subject
	topic, err := pubsub.OpenTopic(ctx, url)
	if err != nil {
		return nil, err
	}

	return &JobQ{
		url,
		topic,
		make(chan struct{}, maxHandlers),
		log.Println,
		log.Printf,
	}, nil
}

func (j *JobQ) AsyncSubscribe(ctx context.Context, cb msgHandler) error {
	sub, err := pubsub.OpenSubscription(ctx, j.sub)
	if err != nil {
		return err
	}
	go j.listen(ctx, &subscription{sub, cb, nil, chanSubscription})

	return nil

}

func (j *JobQ) ChanSubscribe(ctx context.Context, out chan *pubsub.Message) error {
	sub, err := pubsub.OpenSubscription(ctx, j.sub)
	if err != nil {
		return err
	}
	go j.listen(ctx, &subscription{sub, nil, out, chanSubscription})

	return nil
}

func (j *JobQ) listen(ctx context.Context, sub *subscription) {
recvLoop:
	for {
		msg, err := sub.conn.Receive(ctx)
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
			switch sub.typ {
			case asyncSubscription:
				sub.mcb(msg)
			case chanSubscription:
				sub.mch <- msg
			}
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
