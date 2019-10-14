/*
This package provides a few utility generators.
It pulls directly from Concurrency in Go by Katherine Cox-Buday, with minor modiciations.
*/
package generators

import (
	"context"
	"sync"
)

// Take will receive a value from its inputs count times.
func Take(ctx context.Context, in <-chan interface{}, count int) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
				return
			case out <- <-in:
			}
		}
	}()
	return out
}

// Repeat streams the results of repeated invocations of fn to an output channel.
func Repeat(ctx context.Context, fn func() interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case out <- fn():
			}
		}
	}()
	return out
}

// Merge takes a list of channels and merges them to a single output channel.
func Merge(ctx context.Context, channels ...<-chan interface{}) <-chan interface{} {
	var wg sync.WaitGroup
	out := make(chan interface{})

	mux := func(in <-chan interface{}) {
		defer wg.Done()
		for val := range in {
			select {
			case <-ctx.Done():
				return
			case out <- val:
			}
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go mux(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// OrDone returns a value from its input channel, gracefully allowing cancellation.
func OrDone(ctx context.Context, in <-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- val:
				case <-ctx.Done():
				}
			}
		}
	}()
	return out
}
