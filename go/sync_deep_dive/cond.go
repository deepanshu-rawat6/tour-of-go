package sync_deep_dive

// cond.go — sync.Cond: condition variables for goroutine signaling
//
// WHEN TO USE sync.Cond vs channels:
//
//   sync.Cond is better when:
//   • Multiple goroutines wait on a SHARED MUTABLE STATE (e.g. a queue length).
//   • You need Broadcast() to wake ALL waiters at once (e.g. config reload,
//     "gate open" events).
//   • The wake-up condition is not a one-shot event — it can become false again
//     (so a closed channel, which is permanent, doesn't model it well).
//
//   Channels are better when:
//   • You are passing ownership of a value (hand-off semantics).
//   • There is a fixed number of senders/receivers.
//   • You need select over multiple conditions.
//
// Rule of thumb: if you find yourself writing:
//   mu.Lock(); for !ready { mu.Unlock(); time.Sleep(...); mu.Lock() }
// … you want sync.Cond.

import (
	"fmt"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// Bounded queue using sync.Cond
// -----------------------------------------------------------------------------

// boundedQueue is a FIFO queue with a maximum capacity.
// It uses two Conds sharing the same Mutex:
//   - notEmpty — signaled when an item is pushed (unblocks consumers)
//   - notFull  — signaled when an item is popped  (unblocks producers)
type boundedQueue struct {
	mu       sync.Mutex
	items    []int
	capacity int

	// Both Conds share the same underlying lock (mu).
	// A Cond is NOT a channel — it does not buffer signals.
	// If Signal() is called when no goroutine is waiting, the signal is lost.
	// That is why the Wait loop always re-checks the condition.
	notEmpty *sync.Cond
	notFull  *sync.Cond
}

func newBoundedQueue(cap int) *boundedQueue {
	q := &boundedQueue{capacity: cap}
	// sync.NewCond links a Cond to an existing Locker.
	// Both conditions share q.mu so that push/pop atomically
	// check-and-modify state under the same lock.
	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)
	return q
}

// push adds an item, blocking if the queue is at capacity.
func (q *boundedQueue) push(v int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// THE LOOP PATTERN — always use for, never if:
	//
	//   for !condition { cond.Wait() }
	//
	// Why a loop? Two reasons:
	//   1. Spurious wakeups: POSIX condition variables (which Go uses under the
	//      hood on Linux/macOS) can wake a goroutine without Signal/Broadcast.
	//      A loop handles this correctly; an if does not.
	//   2. Multiple waiters: Broadcast wakes ALL waiters, but only one of them
	//      may actually get the resource. The others must re-check and re-wait.
	for len(q.items) == q.capacity {
		// Wait atomically: releases the lock AND suspends this goroutine.
		// When it wakes up (after a Signal or Broadcast), it re-acquires the
		// lock before returning. This is why it must be called while holding mu.
		q.notFull.Wait()
	}

	q.items = append(q.items, v)

	// Signal wakes ONE waiting goroutine (if any). Use Signal when only one
	// waiter needs to act. Here, popping one item frees one slot, so one
	// consumer wakeup is sufficient.
	q.notEmpty.Signal()
}

// pop removes and returns the next item, blocking if the queue is empty.
func (q *boundedQueue) pop() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 {
		q.notEmpty.Wait()
	}

	v := q.items[0]
	q.items = q.items[1:]
	q.notFull.Signal()
	return v
}

// -----------------------------------------------------------------------------
// Config-reload broadcast example
// -----------------------------------------------------------------------------

// configHolder demonstrates Broadcast: when config is reloaded, ALL goroutines
// waiting for the new config are woken simultaneously.
//
// Design: we use a sync.WaitGroup (inWait) to let main know when ALL goroutines
// have entered cond.Wait(). The goroutines call inWait.Done() INSIDE the lock,
// before calling cond.Wait(). Since the lock is exclusive, goroutines enter
// Wait sequentially. When inWait.Wait() returns, all goroutines are in cond.Wait().
type configHolder struct {
	mu      sync.Mutex
	version int
	data    string
	cond    *sync.Cond
}

func newConfigHolder() *configHolder {
	ch := &configHolder{data: "initial config"}
	ch.cond = sync.NewCond(&ch.mu)
	return ch
}

// reload updates the config and broadcasts to ALL waiters.
func (ch *configHolder) reload(newData string) {
	ch.mu.Lock()
	ch.version++
	ch.data = newData
	// Broadcast wakes ALL goroutines blocked in Wait(), unlike Signal which
	// only wakes one. Use Broadcast when the state change is relevant to
	// multiple consumers — e.g. "new config available, everyone re-read it."
	ch.cond.Broadcast()
	ch.mu.Unlock()
}

// waitForVersion blocks until the config version is >= target.
// inWait.Done() is called inside the lock, just before the first cond.Wait().
// This lets the caller know the goroutine is registered in the notify list.
func (ch *configHolder) waitForVersion(target int, inWait *sync.WaitGroup) string {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.version >= target {
		// Already satisfied — never enter Wait.
		return ch.data
	}

	// Signal INSIDE the lock, before Wait. The next statement (cond.Wait)
	// atomically: releases the lock, adds to the notify list, and parks.
	// After Done() returns here, no Broadcast can fire until we call Wait
	// (because main is blocked on inWait.Wait() not on the mutex).
	inWait.Done()

	// The loop handles spurious wakeups — after each wakeup, re-check.
	for ch.version < target {
		ch.cond.Wait()
	}
	return ch.data
}

// -----------------------------------------------------------------------------
// condExample: ties both demonstrations together
// -----------------------------------------------------------------------------

func condExample() {
	fmt.Println("--- sync.Cond ---")

	// --- Producer / consumer queue ---
	q := newBoundedQueue(3) // capacity 3

	var wg sync.WaitGroup

	// Producer: push 6 items into a queue of size 3 — will block on items 4-6
	// until consumers drain space.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= 6; i++ {
			q.push(i)
			fmt.Printf("  [cond] pushed %d\n", i)
		}
	}()

	// Consumer: pop 6 items with a small delay to show blocking.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 6; i++ {
			time.Sleep(20 * time.Millisecond) // simulate slow consumer
			v := q.pop()
			fmt.Printf("  [cond] popped %d\n", v)
		}
	}()

	wg.Wait()

	// --- Broadcast: config reload ---
	fmt.Println()
	cfg := newConfigHolder()

	// The correct sync.Cond Broadcast pattern:
	//   1. Goroutines call inWait.Done() INSIDE the lock, before cond.Wait().
	//   2. Main calls inWait.Wait() to know all goroutines are in cond.Wait().
	//   3. Main calls reload() which sets the condition and Broadcasts.
	//
	// Why this works:
	//   Each goroutine holds the lock exclusively when calling inWait.Done().
	//   After all 3 goroutines have called inWait.Done(), each has also entered
	//   cond.Wait() (which releases the lock). So when inWait.Wait() returns,
	//   ALL goroutines are in cond.Wait() — Broadcast is guaranteed to wake them.
	var inWait sync.WaitGroup
	inWait.Add(3)

	var wg2 sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg2.Add(1)
		go func(id int) {
			defer wg2.Done()
			// Wait for version >= 1 (reload increments 0 → 1)
			data := cfg.waitForVersion(1, &inWait)
			fmt.Printf("  [cond] worker %d got config: %q\n", id, data)
		}(i)
	}

	// Wait for all 3 goroutines to be registered in cond.Wait().
	// This is deterministic — no sleep required.
	inWait.Wait()
	// reload increments version to 1. Workers are waiting for version >= 1.
	cfg.reload("v1 config: feature_flags=on")
	wg2.Wait()

	// --- Note on the for vs if loop pattern ---
	fmt.Println()
	fmt.Println("  [cond] WHY 'for' and not 'if' in the Wait loop:")
	fmt.Println("    Spurious wakeups: OS may wake a goroutine without Signal/Broadcast.")
	fmt.Println("    Multiple waiters: Broadcast wakes all, only one may get the resource.")
	fmt.Println("    In our demo: goroutines wait for version >= 1 (not >= 2) because")
	fmt.Println("    reload() increments version by 1 each call.")
}
