// Package chflow provides context-aware helpers for receiving from and
// sending to Go channels. It helps ensure that operations respect
// cancellation and deadlines via context.Context.
package chflow

import "context"

// Receive waits to receive a value from the provided channel or for the context to be canceled.
// It returns the value (zero value if canceled) and a boolean indicating if the receive was successful.
func Receive[T any](ctx context.Context, ch <-chan T) (T, bool) {
	var data T
	select {
	case <-ctx.Done():
		return data, false
	case data, ok := <-ch:
		return data, ok
	}
}

// Send attempts to send a value to the provided channel unless the context is canceled first.
// It returns true if the send was successful, false if the context was done before sent.
func Send[T any](ctx context.Context, ch chan<- T, data T) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- data:
		return true
	}
}
