package chflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReceive(t *testing.T) {
	t.Run("successful receive", func(t *testing.T) {
		ch := make(chan int, 1)
		ch <- 42

		ctx := t.Context()
		value, ok := Receive(ctx, ch)

		assert.True(t, ok)
		assert.Equal(t, 42, value)
	})

	t.Run("context canceled before receive", func(t *testing.T) {
		ch := make(chan int)
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		value, ok := Receive(ctx, ch)

		assert.False(t, ok)
		assert.Equal(t, 0, value) // zero value for int
	})

	t.Run("channel closed", func(t *testing.T) {
		ch := make(chan string)
		close(ch)

		ctx := t.Context()
		value, ok := Receive(ctx, ch)

		assert.False(t, ok)
		assert.Equal(t, "", value) // zero value for string
	})

	t.Run("receive with different types", func(t *testing.T) {
		// Test with struct
		type testStruct struct {
			Name string
			ID   int
		}

		structCh := make(chan testStruct, 1)
		expected := testStruct{Name: "test", ID: 123}
		structCh <- expected

		ctx := t.Context()
		result, ok := Receive(ctx, structCh)

		assert.True(t, ok)
		assert.Equal(t, expected, result)
	})
}

func TestSend(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		ch := make(chan int, 1)
		ctx := t.Context()

		ok := Send(ctx, ch, 42)

		assert.True(t, ok)
		assert.Equal(t, 42, <-ch)
	})

	t.Run("context canceled before send", func(t *testing.T) {
		ch := make(chan int) // No buffer, will block
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		ok := Send(ctx, ch, 42)

		assert.False(t, ok)

		// Verify nothing was sent
		select {
		case <-ch:
			t.Fatal("Expected no value to be sent")
		default:
			// Expected - no value should be sent
		}
	})

	t.Run("send with different types", func(t *testing.T) {
		// Test with slice
		sliceCh := make(chan []int, 1)
		expected := []int{1, 2, 3}
		ctx := t.Context()

		ok := Send(ctx, sliceCh, expected)

		assert.True(t, ok)
		result := <-sliceCh
		assert.Equal(t, expected, result)
	})

	t.Run("concurrent send and receive", func(t *testing.T) {
		ch := make(chan int)
		ctx := t.Context()

		// Start receiver in goroutine
		receiveDone := make(chan struct{})
		var receivedValue int
		var receiveOk bool

		go func() {
			receivedValue, receiveOk = Receive(ctx, ch)
			close(receiveDone)
		}()

		// Send value
		sendOk := Send(ctx, ch, 99)

		// Wait for receive to complete
		<-receiveDone

		assert.True(t, sendOk)
		assert.True(t, receiveOk)
		assert.Equal(t, 99, receivedValue)
	})

	t.Run("context canceled during concurrent operations", func(t *testing.T) {
		ch := make(chan int)
		ctx, cancel := context.WithCancel(t.Context())

		// Start receiver in goroutine
		receiveDone := make(chan struct{})
		var receiveOk bool

		go func() {
			_, receiveOk = Receive(ctx, ch)
			close(receiveDone)
		}()

		// Start sender in goroutine
		sendDone := make(chan struct{})
		var sendOk bool

		go func() {
			sendOk = Send(ctx, ch, 42)
			close(sendDone)
		}()

		// Cancel context immediately - both operations should be blocked on the channel
		cancel()

		// Wait for both operations to complete
		<-receiveDone
		<-sendDone

		// At least one operation should fail due to context cancellation
		// (It's possible that one operation completes before the other starts,
		// but at least one should be canceled)
		assert.False(t, receiveOk || sendOk, "At least one operation should fail due to context cancellation")
	})
}

func TestReceiveAndSendIntegration(t *testing.T) {
	t.Run("pipeline with context", func(t *testing.T) {
		input := make(chan int, 3)
		output := make(chan int, 3)
		ctx := t.Context()

		// Send test data
		input <- 1
		input <- 2
		input <- 3
		close(input)

		// Process data through pipeline
		go func() {
			for {
				value, ok := Receive(ctx, input)
				if !ok {
					close(output)
					return
				}

				// Transform value (multiply by 2)
				transformed := value * 2

				if !Send(ctx, output, transformed) {
					return
				}
			}
		}()

		// Collect results
		var results []int
		for {
			value, ok := Receive(ctx, output)
			if !ok {
				break
			}
			results = append(results, value)
		}

		expected := []int{2, 4, 6}
		assert.Equal(t, expected, results)
	})

	t.Run("pipeline with context cancellation", func(t *testing.T) {
		input := make(chan int)
		output := make(chan int)
		ctx, cancel := context.WithCancel(t.Context())

		// Start pipeline
		pipelineDone := make(chan struct{})
		go func() {
			for {
				value, ok := Receive(ctx, input)
				if !ok {
					close(pipelineDone)
					return
				}

				if !Send(ctx, output, value*2) {
					close(pipelineDone)
					return
				}
			}
		}()

		// Send one value
		input <- 10

		// Receive the transformed value
		result, ok := Receive(ctx, output)
		assert.True(t, ok)
		assert.Equal(t, 20, result)

		// Cancel context
		cancel()

		// Pipeline should terminate
		select {
		case <-pipelineDone:
			// Expected - pipeline should terminate
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Pipeline should terminate when context is canceled")
		}

		close(input)
	})
}

func TestFirstNonNil(t *testing.T) {
	t.Run("all nil channels", func(t *testing.T) {
		channels := []chan int{nil, nil, nil}
		got := FirstNonNil(channels...)
		assert.Nil(t, got, "FirstNonNil() should return nil when all channels are nil")
	})

	t.Run("first channel non-nil", func(t *testing.T) {
		ch1 := make(chan int)
		channels := []chan int{ch1, nil, nil}
		got := FirstNonNil(channels...)
		assert.NotNil(t, got, "FirstNonNil() should return non-nil channel when first channel is non-nil")
	})

	t.Run("middle channel non-nil", func(t *testing.T) {
		ch2 := make(chan int)
		channels := []chan int{nil, ch2, nil}
		got := FirstNonNil(channels...)
		assert.NotNil(t, got, "FirstNonNil() should return non-nil channel when middle channel is non-nil")
	})

	t.Run("last channel non-nil", func(t *testing.T) {
		ch3 := make(chan int)
		channels := []chan int{nil, nil, ch3}
		got := FirstNonNil(channels...)
		assert.NotNil(t, got, "FirstNonNil() should return non-nil channel when last channel is non-nil")
	})

	t.Run("empty list", func(t *testing.T) {
		channels := []chan int{}
		got := FirstNonNil(channels...)
		assert.Nil(t, got, "FirstNonNil() should return nil for an empty list of channels")
	})
}
