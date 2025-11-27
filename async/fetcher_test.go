package async

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func fetchUser(ctx context.Context, id int) (string, error) {
	time.Sleep(100 * time.Millisecond)
	if id <= 0 {
		return "", errors.New("invalid user ID")
	}
	return fmt.Sprintf("user-%d", id), nil
}

func TestFetcher_Success(t *testing.T) {
	ctx := context.Background()
	f := Async(fetchUser, 5).Run(ctx)

	for i := 0; i < 5; i++ {
		val, err := f.Await()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if val != "user-5" {
			t.Errorf("expected 'user-5', got %v", val)
		}
	}
}

func TestFetcher_Error(t *testing.T) {
	ctx := context.Background()
	f := Async(fetchUser, -1).Run(ctx)
	val, err := f.Await()
	if err == nil {
		t.Error("expected error, got nil")
	}
	if val != "" {
		t.Errorf("expected empty value, got %v", val)
	}
}

func TestFetcher_Panic(t *testing.T) {
	ctx := context.Background()
	panicFn := func(ctx context.Context, a int) (string, error) {
		panic("boom")
	}
	f := Async(panicFn, 1).Run(ctx)
	val, err := f.Await()
	if err == nil {
		t.Error("expected error from panic, got nil")
	}
	if err != nil && !contains(err.Error(), "panic recovered") {
		t.Errorf("expected panic recovered in error, got %v", err)
	}
	if val != "" {
		t.Errorf("expected empty value, got %v", val)
	}
}

func TestFetcher_AwaitBeforeRun_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Await called before Run")
		}
	}()
	f := Async(fetchUser, 1)
	_, _ = f.Await()
}

func TestFetcher_ConcurrentAwait(t *testing.T) {
	ctx := context.Background()
	f := Async(fetchUser, 7).Run(ctx)
	done := make(chan struct{})
	go func() {
		val, err := f.Await()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if val != "user-7" {
			t.Errorf("expected 'user-7', got %v", val)
		}
		close(done)
	}()
	val, err := f.Await()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if val != "user-7" {
		t.Errorf("expected 'user-7', got %v", val)
	}
	<-done
}

func TestMultipleFetchers_RunInParallel(t *testing.T) {
	ctx := context.Background()
	start := time.Now()

	f1 := Async(fetchUser, 1).Run(ctx)
	f2 := Async(fetchUser, 2).Run(ctx)
	f3 := Async(fetchUser, 3).Run(ctx)

	// Await all
	_, err1 := f1.Await()
	_, err2 := f2.Await()
	_, err3 := f3.Await()

	if err1 != nil {
		t.Errorf("expected no error for f1, got %v", err1)
	}
	if err2 != nil {
		t.Errorf("expected no error for f2, got %v", err2)
	}
	if err3 != nil {
		t.Errorf("expected no error for f3, got %v", err3)
	}

	elapsed := time.Since(start)
	if elapsed.Milliseconds() >= 102 {
		t.Errorf("fetchers should run in parallel, took %dms", elapsed.Milliseconds())
	}
}

func TestFetcher_PanicHandlerCalled(t *testing.T) {
	ctx := context.Background()
	handlerCalled := false
	var recoveredValue interface{}

	SetPanicHandlers(func(c context.Context, r interface{}) {
		handlerCalled = true
		recoveredValue = r
	})

	panicFn := func(ctx context.Context, a int) (string, error) {
		panic("test-panic")
	}
	f := Async(panicFn, 1).Run(ctx)
	_, err := f.Await()
	if !handlerCalled {
		t.Error("expected panic handler to be called")
	}
	if recoveredValue != "test-panic" {
		t.Errorf("expected recovered value to be 'test-panic', got %v", recoveredValue)
	}
	if err == nil || err.Error() == "" {
		t.Error("expected error from panic, got nil or empty error")
	}

	// Reset handlers after test
	SetPanicHandlers()
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (contains(s[1:], substr) || contains(s[:len(s)-1], substr))))
}
