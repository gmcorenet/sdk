package gmcoreevents

import (
	"context"
	"sync"
)

type Listener func(context.Context, interface{}) error

type Unsubscribe func()

type listenerEntry struct {
	id       int
	listener Listener
}

type Bus struct {
	mu         sync.RWMutex
	listeners  map[string][]listenerEntry
	nextID     int
	onUnsubscribe []Unsubscribe
}

func NewBus() *Bus {
	return &Bus{listeners: map[string][]listenerEntry{}, nextID: 1}
}

func (b *Bus) Subscribe(name string, listener Listener) Unsubscribe {
	if b == nil || listener == nil {
		return func() {}
	}
	entry := b.subscribeInternal(name, listener)
	return func() {
		b.unsubscribe(name, entry.id)
	}
}

func (b *Bus) SubscribeOnce(name string, listener Listener) Unsubscribe {
	if b == nil || listener == nil {
		return func() {}
	}
	once := sync.Once{}
	used := false
	mu := sync.Mutex{}

	entry := b.subscribeInternal(name, func(ctx context.Context, event interface{}) error {
		mu.Lock()
		if used {
			mu.Unlock()
			return nil
		}
		used = true
		mu.Unlock()

		var err error
		once.Do(func() {
			err = listener(ctx, event)
		})
		return err
	})

	return func() {
		mu.Lock()
		if used {
			mu.Unlock()
			return
		}
		used = true
		mu.Unlock()
		b.unsubscribe(name, entry.id)
	}
}

func (b *Bus) subscribeInternal(name string, listener Listener) listenerEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry := listenerEntry{id: b.nextID, listener: listener}
	b.nextID++
	b.listeners[name] = append(b.listeners[name], entry)
	return entry
}

func (b *Bus) unsubscribe(name string, id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	listeners := b.listeners[name]
	for i, e := range listeners {
		if e.id == id {
			b.listeners[name] = append(listeners[:i], listeners[i+1:]...)
			return
		}
	}
}

func (b *Bus) UnsubscribeAll(name string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.listeners, name)
}

func (b *Bus) Dispatch(ctx context.Context, name string, event interface{}) error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	var listeners []Listener
	for _, entry := range b.listeners[name] {
		listeners = append(listeners, entry.listener)
	}
	for _, entry := range b.listeners["*"] {
		listeners = append(listeners, entry.listener)
	}
	b.mu.RUnlock()
	for _, listener := range listeners {
		if err := listener(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bus) DispatchCollect(ctx context.Context, name string, event interface{}) []error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	var listeners []Listener
	for _, entry := range b.listeners[name] {
		listeners = append(listeners, entry.listener)
	}
	for _, entry := range b.listeners["*"] {
		listeners = append(listeners, entry.listener)
	}
	b.mu.RUnlock()
	errs := []error{}
	for _, listener := range listeners {
		if err := listener(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
