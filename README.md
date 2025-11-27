# async-fetcher

`async-fetcher` is a lightweight Go package that provides an ergonomic, type-safe way to run asynchronous computations and retrieve their results using an `Async/Await`-like pattern. It is designed for concurrent execution of multiple functions, with safe result fetching and panic handling.

## Features

- **Async/Await Syntax:** Easily start async computations and await their results.
- **Type Safety:** Generics ensure compile-time type safety for arguments and return values.
- **Parallel Execution:** Run multiple fetchers in parallel and await their results independently.
- **Panic Handling:** Register global panic handlers for robust error recovery.
- **Context Support:** Integrates with Go's `context.Context` for cancellation and deadlines.
- **No External Dependencies:** Pure Go, no third-party libraries required.

## Installation

```bash
go get github.com/ashutosh-bh/async-fetcher/async
```

## Usage

### Basic Example

```go
package main

import (
	"context"
	"fmt"
	"time"
	"yourmodule/async"
)

func fetchUser(ctx context.Context, id int) (string, error) {
	time.Sleep(100 * time.Millisecond)
	if id <= 0 {
		return "", fmt.Errorf("invalid user ID")
	}
	return fmt.Sprintf("user-%d", id), nil
}

func main() {
	ctx := context.Background()
	fetcher := async.Async(fetchUser, 42).Run(ctx)
	val, err := fetcher.Await()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Fetched:", val)
}
```

### Running Multiple Fetchers in Parallel

```go
ctx := context.Background()
f1 := async.Async(fetchUser, 1).Run(ctx)
f2 := async.Async(fetchUser, 2).Run(ctx)
f3 := async.Async(fetchUser, 3).Run(ctx)

v1, err1 := f1.Await()
v2, err2 := f2.Await()
v3, err3 := f3.Await()
```

### Panic Handling

Register global panic handlers to catch panics from async computations:

```go
async.SetPanicHandlers(func(ctx context.Context, r interface{}) {
	fmt.Printf("Panic caught: %v\n", r)
})
```

## API Reference

### `Async(fn, arg)`

Creates a new `Fetcher` for the provided function and argument. The computation does not start until `Run` is called.

- `fn`: Function with signature `func(context.Context, A) (T, error)`
- `arg`: Argument of type `A`

### `Run(ctx)`

Starts the asynchronous computation in a new goroutine. Returns the `Fetcher` itself for chaining.

### `Await()`

Blocks until the computation is complete and returns the result and error. Panics if called before `Run`.

### `SetPanicHandlers(handlers...)`

Sets global panic handlers to be invoked if a panic occurs in any async computation.

## Design Notes

- **Concurrency:** Each fetcher runs in its own goroutine. Results are safely retrievable from multiple goroutines.
- **Context Awareness:** If the context is cancelled before or during execution, the error is set accordingly.
- **Single Await:** Once the computation is complete, `Await` can be called multiple times to retrieve the result.

## Optimization Details

This package is highly optimized for concurrent usage and memory efficiency:

- **Channel Used Only for Signaling:** The internal channel in each `Fetcher` is used solely to signal completion of the asynchronous job. No data is sent through the channel; it is simply closed when the job is done.
- **Immediate Channel Closure:** As soon as the computation finishes (success, error, or panic), the channel is closed. This allows all goroutines waiting on `Await()` to proceed without delay.
- **Garbage Collector Friendly:** Since the channel is only used for signaling and is closed immediately, the Go garbage collector is never blocked, even if multiple goroutines are reading from the same channel. There is no risk of goroutine leaks or memory retention due to lingering channel references.
- **Safe Multiple Reads:** Multiple calls to `Await()` are safe and efficient. All readers are simply waiting for the channel to close, which is a non-blocking operation for the garbage collector and runtime.

This design ensures minimal memory footprint, fast signaling, and robust concurrent access, making `async-fetcher` suitable for high-performance applications.

## Testing

Unit tests are provided in `async/fetcher_test.go` covering success, error, panic, concurrent awaits, and panic handler invocation.

Run tests with:

```bash
go test ./async
```

## Contributing

Contributions are welcome! Please open issues or submit pull requests for improvements or bug fixes.

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/fooBar`).
3. Commit your changes (`git commit -am 'Add some fooBar'`).
4. Push to the branch (`git push origin feature/fooBar`).
5. Open a pull request.

## License

MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgements

Inspired by async/await patterns in other languages, adapted for Go's concurrency model.
