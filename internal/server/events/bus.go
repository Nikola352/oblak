package events

import (
	"log"
	"sync"
)

const BufferSize = 1024
const BufferSizePerSubscription = 100

type Bus[Event any] struct {
	ch          chan Event
	subscribers []chan Event
	mu          sync.RWMutex
}

func NewBus[Event any]() *Bus[Event] {
	return &Bus[Event]{
		ch:          make(chan Event, BufferSize),
		subscribers: []chan Event{},
	}
}

func (b *Bus[Event]) Subscribe() <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, BufferSizePerSubscription)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

func (b *Bus[Event]) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, sub := range b.subscribers {
		select {
		case sub <- event:
		default:
			log.Printf("dropping event %Event", event)
		}
	}
}
