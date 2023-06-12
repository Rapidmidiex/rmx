package jobq

import (
	"context"
	"fmt"
	"log"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

type subscriptionType int

const (
	asyncSubscription = subscriptionType(iota)
	chanSubscription

	urlPrefix = "mem://"
)

type subscription struct {
	conn *pubsub.Subscription
	sem  chan struct{}
	mcb  func(*pubsub.Message)
	mch  chan *pubsub.Message
	typ  subscriptionType
	log  func(v ...any)
	logF func(format string, v ...any)
}

func New(ctx context.Context, subj string) (*pubsub.Topic, error) {
	topic, err := pubsub.OpenTopic(ctx, urlPrefix+subj)
	if err != nil {
		return nil, err
	}

	return topic, nil
}

func AsyncSubscribe(ctx context.Context, subj string, cb func(*pubsub.Message), maxHandlers int) error {
	sub, err := pubsub.OpenSubscription(ctx, urlPrefix+subj)
	if err != nil {
		return fmt.Errorf("jobq topic[%s] AsyncSubscribe: %v", urlPrefix+subj, err)
	}

	go listen(
		ctx,
		&subscription{
			sub,
			make(chan struct{}, maxHandlers),
			cb,
			nil,
			chanSubscription,
			log.Println,
			log.Printf,
		},
	)

	return nil
}

func ChanSubscribe(ctx context.Context, subj string, out chan *pubsub.Message, maxHandlers int) error {
	sub, err := pubsub.OpenSubscription(ctx, urlPrefix+subj)
	if err != nil {
		return fmt.Errorf("jobq topic[%s] ChanSubscribe: %v", urlPrefix+subj, err)
	}

	go listen(
		ctx,
		&subscription{
			sub,
			make(chan struct{}, maxHandlers),
			nil,
			out,
			chanSubscription,
			log.Println,
			log.Printf,
		},
	)
	return nil
}

func listen(ctx context.Context, sub *subscription) {
recvLoop:
	for {
		msg, err := sub.conn.Receive(ctx)
		if err != nil {
			sub.logF("jobq receive: %v", err)
		}

		select {
		case <-ctx.Done():
			break recvLoop
		case sub.sem <- struct{}{}:

			sub.handleMsg(msg)
		}
	}

	sub.block()
}

func (s *subscription) handleMsg(m *pubsub.Message) {
	defer func() { <-s.sem }() // frees up the channel for a new receiver
	defer m.Ack()              // message must always be acknowledged

	switch s.typ {
	case asyncSubscription:
		s.mcb(m)
	case chanSubscription:
		s.mch <- m
	}
}

func (s *subscription) block() {
	// we're no longer receiving messages. Wait to finish handling any
	// unacknowledged messages by totally acquiring the semaphore.
	for n := 0; n < cap(s.sem); n++ {
		s.sem <- struct{}{}
	}
}

func Publish(ctx context.Context, subj string, msg *pubsub.Message) error {
	topic, err := pubsub.OpenTopic(ctx, urlPrefix+subj)
	if err != nil {
		return fmt.Errorf("jobq topic[%s] Publish: %v", urlPrefix+subj, err)
	}

	return topic.Send(ctx, msg)
}