package gmcore_events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewBus(t *testing.T) {
	b := NewBus()
	if b == nil {
		t.Fatal("expected non-nil bus")
	}
	if b.listeners == nil {
		t.Error("expected listeners map to be initialized")
	}
}

func TestBus_Subscribe(t *testing.T) {
	b := NewBus()
	var called int32

	unsub := b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected called 1 time, got %d", called)
	}

	unsub()
	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("after unsubscribe expected called 1 time, got %d", called)
	}
}

func TestBus_Subscribe_ReturnsUnsubscribe(t *testing.T) {
	b := NewBus()
	var called int32

	unsub := b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	b.Dispatch(context.Background(), "test", nil)
	unsub()
	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected called exactly once, got %d", called)
	}
}

func TestBus_Subscribe_Multiple(t *testing.T) {
	b := NewBus()
	var count int32

	unsub1 := b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	unsub2 := b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&count, 10)
		return nil
	})

	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&count) != 11 {
		t.Errorf("expected count 11, got %d", count)
	}

	unsub1()
	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&count) != 21 {
		t.Errorf("expected count 21, got %d", count)
	}

	unsub2()
	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&count) != 21 {
		t.Errorf("after all unsubscribed count should stay 21, got %d", count)
	}
}

func TestBus_SubscribeOnce(t *testing.T) {
	b := NewBus()
	var called int32

	b.SubscribeOnce("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	b.Dispatch(context.Background(), "test", nil)
	b.Dispatch(context.Background(), "test", nil)
	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected called once, got %d", called)
	}
}

func TestBus_SubscribeOnce_ReturnsUnsubscribe(t *testing.T) {
	b := NewBus()
	var called int32

	unsub := b.SubscribeOnce("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	unsub()
	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&called) != 0 {
		t.Errorf("after unsubscribe before dispatch, expected called 0, got %d", called)
	}
}

func TestBus_SubscribeOnce_MultipleEvents(t *testing.T) {
	b := NewBus()
	var calls int32

	b.SubscribeOnce("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("first error")
	})
	b.SubscribeOnce("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&calls, 10)
		return nil
	})

	err := b.Dispatch(context.Background(), "test", nil)

	if err == nil {
		t.Error("expected error from first listener")
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected calls 1 (first error stops dispatch), got %d", calls)
	}

	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&calls) != 11 {
		t.Errorf("second dispatch should call second listener, expected 11, got %d", calls)
	}
}

func TestBus_UnsubscribeAll(t *testing.T) {
	b := NewBus()
	var called int32

	b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	b.Subscribe("other", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&called, 100)
		return nil
	})

	b.Dispatch(context.Background(), "test", nil)
	b.UnsubscribeAll("test")
	b.Dispatch(context.Background(), "test", nil)
	b.Dispatch(context.Background(), "other", nil)

	if atomic.LoadInt32(&called) != 101 {
		t.Errorf("expected 101 (1 from first dispatch + 100 from other), got %d", called)
	}
}

func TestBus_Dispatch(t *testing.T) {
	b := NewBus()
	var received interface{}

	b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		received = event
		return nil
	})

	event := "hello world"
	b.Dispatch(context.Background(), "test", event)

	if received != event {
		t.Errorf("expected %v, got %v", event, received)
	}
}

func TestBus_Dispatch_Wildcard(t *testing.T) {
	b := NewBus()
	var called int32

	b.Subscribe("*", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	b.Dispatch(context.Background(), "any.event", nil)
	b.Dispatch(context.Background(), "another.event", nil)

	if atomic.LoadInt32(&called) != 2 {
		t.Errorf("expected 2, got %d", called)
	}
}

func TestBus_Dispatch_Error(t *testing.T) {
	b := NewBus()

	expectedErr := errors.New("test error")
	var gotErr error

	b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		return expectedErr
	})
	b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		gotErr = errors.New("should not be called")
		return nil
	})

	err := b.Dispatch(context.Background(), "test", nil)

	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if gotErr != nil {
		t.Error("second listener should not be called after error")
	}
}

func TestBus_DispatchCollect(t *testing.T) {
	b := NewBus()
	var calls int32

	b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("err1")
	})
	b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&calls, 10)
		return errors.New("err2")
	})
	b.Subscribe("test", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&calls, 100)
		return nil
	})

	errs := b.DispatchCollect(context.Background(), "test", nil)

	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}
	if atomic.LoadInt32(&calls) != 111 {
		t.Errorf("expected 111, got %d", calls)
	}
}

func TestBus_Dispatch_NilBus(t *testing.T) {
	var b *Bus
	err := b.Dispatch(context.Background(), "test", nil)
	if err != nil {
		t.Error("expected nil error from nil bus")
	}

	errs := b.DispatchCollect(context.Background(), "test", nil)
	if errs != nil {
		t.Error("expected nil errors from nil bus")
	}
}

func TestBus_Subscribe_NilListener(t *testing.T) {
	b := NewBus()
	unsub := b.Subscribe("test", nil)
	unsub()

	b.SubscribeOnce("test", nil)

	b.Dispatch(context.Background(), "test", nil)
}

func TestBus_Concurrent(t *testing.T) {
	b := NewBus()
	var count int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Subscribe("test", func(ctx context.Context, event interface{}) error {
				atomic.AddInt32(&count, 1)
				return nil
			})
		}()
	}
	wg.Wait()

	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&count) != 100 {
		t.Errorf("expected 100, got %d", count)
	}
}

func TestBus_SubscribeOnce_Concurrent(t *testing.T) {
	b := NewBus()
	var count int32

	for i := 0; i < 10; i++ {
		b.SubscribeOnce("test", func(ctx context.Context, event interface{}) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("expected 10 (all listeners fire once), got %d", count)
	}

	b.Dispatch(context.Background(), "test", nil)

	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("second dispatch should not call any (all already used), got %d", count)
	}
}

func TestBus_Unsubscribe_InSubscription(t *testing.T) {
	b := NewBus()
	var count int32
	var innerUnsub Unsubscribe

	b.Subscribe("outer", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&count, 1)
		innerUnsub()
		return nil
	})

	innerUnsub = b.Subscribe("inner", func(ctx context.Context, event interface{}) error {
		atomic.AddInt32(&count, 100)
		return nil
	})

	b.Dispatch(context.Background(), "outer", nil)

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	b.Dispatch(context.Background(), "inner", nil)

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("inner listener should have been unsubscribed, count is %d", count)
	}
}
