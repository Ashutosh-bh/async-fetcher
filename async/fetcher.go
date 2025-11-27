package async

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

// Package async provides utilities for running asynchronous computations

var (
	globalPanicHandlers []func(context.Context, interface{})
	handlersMu          sync.RWMutex
)

// SetPanicHandlers sets the global panic handlers to be used by all Fetchers.
// This should be called once during service initialization.
func SetPanicHandlers(handlers ...func(context.Context, interface{})) {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	globalPanicHandlers = handlers
}

type Result[T any] struct {
	Value T
	Err   error
}

// Fetcher represents a one-time asynchronous computation whose result
// can be safely fetched concurrently after starting with Run.
//
// Example function signature:
//
//	func fetchUser(ctx context.Context, id int) (string, error) {
//		return fmt.Sprintf("user-%d", id), nil
//	}
//
// Example usage:
//
//	f := Async(fetchUser, 42).Run(ctx)
//	val, err := f.Await()
type Fetcher[T any, A any] struct {
	Fn     func(context.Context, A) (T, error)
	Arg    A
	result Result[T]
	ch     *chan struct{}
}

// Async creates a new Fetcher for the provided function and argument.
// The computation does not start until Run is called.
func Async[T any, A any](fn func(context.Context, A) (T, error), arg A) *Fetcher[T, A] {
	return &Fetcher[T, A]{
		Fn:  fn,
		Arg: arg,
	}
}

// Run starts the asynchronous computation in a new goroutine.
// If called multiple times, only the first call starts the computation.
// Returns the Fetcher itself for chaining.
func (f *Fetcher[T, A]) Run(ctx context.Context) *Fetcher[T, A] {
	if f.ch != nil {
		return f
	}

	f.ch = ToPtr(make(chan struct{}))
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handlePanicErr(ctx, r)
				f.result.Err = fmt.Errorf("[Fetcher.Async]: panic recovered: %v", r)
			}

			close(*f.ch)
		}()

		select {
		case <-ctx.Done():
			var zero T
			f.result = Result[T]{Value: zero, Err: ctx.Err()}
			return
		default:
			// Proceed with computation.
		}

		val, err := f.Fn(ctx, f.Arg)

		// Check again in case f.Fn is context-aware and returns quickly on cancel
		if ctx.Err() != nil && err == nil {
			err = ctx.Err()
		}

		f.result = Result[T]{Value: val, Err: err}
	}()

	return f
}

// Await blocks until the computation is complete and returns the result and error.
// Panics if called before Run. Safe for concurrent use after Run.
func (f *Fetcher[T, A]) Await() (T, error) {
	if f.ch == nil {
		panic("fetcher not started, call Run() first")
	}

	<-*f.ch // wait until ready
	return f.result.Value, f.result.Err
}

func handlePanicErr(
	ctx context.Context,
	r interface{},
) {
	log.Printf("[ERROR] Panic recovered - %+v\n%s", r, string(debug.Stack()))
	for _, fn := range globalPanicHandlers {
		fn(ctx, r)
	}
}

// ToPtr converts non pointer value to pointer
func ToPtr[T any](v T) *T {
	return &v
}
