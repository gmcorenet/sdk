package gmcore_messenger

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/gmcorenet/sdk/gmcore-config"
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
	msgType := getType(message)
	handlers := b.handlers["*"]
	if hs, ok := b.handlers[msgType]; ok {
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
	if message == nil {
		return "nil"
	}
	t := reflect.TypeOf(message)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}

type Transport interface {
	Send(messages []interface{}) error
	Receive() (interface{}, error)
	Ack(message interface{}) error
	Reject(message interface{}) error
}

type InMemoryTransport struct {
	messages []interface{}
	inFlight []interface{}
	mu       sync.RWMutex
}

var ErrMessageNotInFlight = errors.New("message not in flight")

func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{
		messages: make([]interface{}, 0),
		inFlight: make([]interface{}, 0),
	}
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
	t.inFlight = append(t.inFlight, msg)
	return msg, nil
}

func (t *InMemoryTransport) Ack(message interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	idx := indexOfMessage(t.inFlight, message)
	if idx < 0 {
		return ErrMessageNotInFlight
	}
	t.inFlight = append(t.inFlight[:idx], t.inFlight[idx+1:]...)
	return nil
}

func (t *InMemoryTransport) Reject(message interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	idx := indexOfMessage(t.inFlight, message)
	if idx < 0 {
		return ErrMessageNotInFlight
	}
	t.inFlight = append(t.inFlight[:idx], t.inFlight[idx+1:]...)
	t.messages = append([]interface{}{message}, t.messages...)
	return nil
}

func indexOfMessage(messages []interface{}, target interface{}) int {
	for i, message := range messages {
		if reflect.DeepEqual(message, target) {
			return i
		}
	}
	return -1
}

type Worker struct {
	transport Transport
	bus       Bus
	stop      chan struct{}
	stopOnce  sync.Once
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
	w.stopOnce.Do(func() {
		close(w.stop)
	})
}

type Config struct {
	WorkerCount int                    `yaml:"worker_count" json:"worker_count"`
	RetryPolicy RetryPolicy            `yaml:"retry_policy" json:"retry_policy"`
	Transport   string                 `yaml:"transport" json:"transport"`
	Params      map[string]interface{} `yaml:"params" json:"params"`
}

type RetryPolicy struct {
	MaxRetries   int     `yaml:"max_retries" json:"max_retries"`
	InitialDelay int     `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay     int     `yaml:"max_delay" json:"max_delay"`
	Multiplier   float64 `yaml:"multiplier" json:"multiplier"`
}

func LoadConfig(appPath string) (*Config, error) {
	l := gmcore_config.NewLoader[Config](appPath)
	for _, name := range []string{"messenger.yaml", "messenger.yml"} {
		if cfg, err := l.LoadDefault(name); cfg != nil || err != nil {
			return cfg, err
		}
	}
	return nil, nil
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 1000,
		MaxDelay:     60000,
		Multiplier:   2.0,
	}
}

func (p RetryPolicy) NextDelay(attempt int) time.Duration {
	delay := float64(p.InitialDelay) * pow(p.Multiplier, float64(attempt))
	if delay > float64(p.MaxDelay) {
		delay = float64(p.MaxDelay)
	}
	return time.Duration(delay) * time.Millisecond
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
