/*
This package provides a few utility generators.
It pulls directly from Concurrency in Go by Katherine Cox-Buday, with minor modiciations.
*/
package generators

import (
	"sync"
	"time"
)

// Take will receive a value from its inputs count times.
func Take(done, in <-chan interface{}, count int) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for i := 0; i < count; i++ {
			select {
			case <-done:
				return
			case out <- <-in:
			}
		}
	}()
	return out
}

// Repeat streams the results of repeated invocations of fn to an output channel.
func Repeat(done <-chan interface{}, fn func() interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case out <- fn():
			}
		}
	}()
	return out
}

// Merge takes a list of channels and merges them to a single output channel.
func Merge(done <-chan interface{}, channels ...<-chan interface{}) <-chan interface{} {
	var wg sync.WaitGroup
	out := make(chan interface{})

	mux := func(in <-chan interface{}) {
		defer wg.Done()
		for val := range in {
			select {
			case <-done:
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
func OrDone(done, in <-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case val, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- val:
				case <-done:
				}
			}
		}
	}()
	return out
}

type WorkFunc func(done <-chan interface{}, interval time.Duration) (heartbeat <-chan interface{})

// Steward takes a timeout and a long running work function.
// It watches for heartbeats from the work function, restarting it when none
// have been received for provided timeout. The steward returns its own
// heartbeat.
func Steward(timeout time.Duration, work WorkFunc) WorkFunc {
	return func(done <-chan interface{}, interval time.Duration) <-chan interface{} {
		heartbeat := make(chan interface{})
		go func() {
			defer close(heartbeat)
			var wardDone chan interface{}
			var wardHeartbeat <-chan interface{}
			startWard := func() {
				wardDone = make(chan interface{})
				wardHeartbeat = work(OrDone(done, wardDone), timeout/2)
			}
			startWard()
			pulse := time.NewTicker(interval).C
		loop:
			for {
				timeoutSignal := time.After(timeout)
				for {
					select {
					case <-pulse:
						select {
						case heartbeat <- struct{}{}:
						default:
						}
					case <-wardHeartbeat:
						continue loop
					case <-timeoutSignal:
						close(wardDone)
						startWard()
						continue loop
					case <-done:
						return
					}
				}
			}
		}()
		return heartbeat
	}
}
