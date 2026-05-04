package gmcore_messenger

import (
	"sync"
	"time"
)

type MessageHandler func(message interface{}) error

type Bus interface {
	Dispatch(message interface{}) error
	DispatchAsync(message interface{})
	Register(handler MessageHandler, messageType string)
}

type bus struct {
	handlers map[string][]MessageHandler
	mu       sync.RWMutex
}

func NewBus() *bus {
	return &bus{handlers: make(map[string][]MessageHandler)}
}

func (b *bus) Dispatch(message interface{}) error {
	b.mu.RLock()
	handlers := b.handlers["*"]
	if hs, ok := b.handlers[getType(message)]; ok {
		handlers = append(handlers, hs...)
	}
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(message); err != nil {
			return err
		}
	}
	return nil
}

func (b *bus) DispatchAsync(message interface{}) {
	go func() {
		b.Dispatch(message)
	}()
}

func (b *bus) Register(handler MessageHandler, messageType string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[messageType] = append(b.handlers[messageType], handler)
}

func getType(message interface{}) string {
	return "default"
}

type Transport interface {
	Send(messages []interface{}) error
	Receive() (interface{}, error)
	Ack(message interface{}) error
	Reject(message interface{}) error
}

type InMemoryTransport struct {
	messages []interface{}
	mu       sync.RWMutex
}

func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{messages: make([]interface{}, 0)}
}

func (t *InMemoryTransport) Send(messages []interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages = append(t.messages, messages...)
	return nil
}

func (t *InMemoryTransport) Receive() (interface{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.messages) == 0 {
		return nil, nil
	}
	msg := t.messages[0]
	t.messages = t.messages[1:]
	return msg, nil
}

func (t *InMemoryTransport) Ack(message interface{}) error   { return nil }
func (t *InMemoryTransport) Reject(message interface{}) error { return nil }

type Worker struct {
	transport Transport
	bus       Bus
	stop      chan struct{}
}

func NewWorker(transport Transport, bus Bus) *Worker {
	return &Worker{transport: transport, bus: bus, stop: make(chan struct{})}
}

func (w *Worker) Start() {
	go func() {
		for {
			select {
			case <-w.stop:
				return
			default:
				msg, err := w.transport.Receive()
				if err != nil {
					time.Sleep(1 * time.Second)
					continue
				}
				if msg != nil {
					w.bus.Dispatch(msg)
					w.transport.Ack(msg)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stop)
}
