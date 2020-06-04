package kafka

import (
	"fmt"

	"github.com/gocql/gocql"

	"github.com/dollarshaveclub/furan/generated/lib"
)

type FakeEventBus struct {
	c chan *lib.BuildEvent
}

var _ EventBusProducer = &FakeEventBus{}
var _ EventBusConsumer = &FakeEventBus{}

func NewFakeEventBusProducer(capacity uint) *FakeEventBus {
	return &FakeEventBus{
		c: make(chan *lib.BuildEvent, capacity),
	}
}

func (fb *FakeEventBus) PublishEvent(event *lib.BuildEvent) error {
	select {
	case fb.c <- event:
		return nil
	default:
		return fmt.Errorf("channel full")
	}
}

func (fb *FakeEventBus) SubscribeToTopic(c chan<- *lib.BuildEvent, done <-chan struct{}, id gocql.UUID) error {
	if fb.c == nil {
		return fmt.Errorf("nil channel")
	}
	go func() {
		for {
			select {
			case event, ok := <-fb.c:
				if !ok {
					return
				}
				if event != nil && event.BuildId == id.String() {
					c <- event
				}
			case <-done:
				return
			}
		}
	}()
	return nil
}
