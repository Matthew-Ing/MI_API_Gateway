package circuitbreaker

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type Breaker struct {
	mu          sync.Mutex
	state       State
	failures    int
	threshold   int
	openUntil   time.Time
	cooldown    time.Duration
	halfOpenTry bool // true = a probe is already in flight
}

func NewBreaker(threshold int, cooldown time.Duration) *Breaker {
	if threshold < 1 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 10 * time.Second
	}
	return &Breaker{state: StateClosed, threshold: threshold, cooldown: cooldown}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Now().After(b.openUntil) {
			b.state = StateHalfOpen
			b.halfOpenTry = true
			return true // one probe
		}
		return false
	case StateHalfOpen:
		return false // only the probe that opened the gate
	}
	return false
}

func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.halfOpenTry = false
	b.state = StateClosed
}

func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.halfOpenTry = false
	b.failures++
	if b.state == StateHalfOpen || b.failures >= b.threshold {
		b.state = StateOpen
		b.openUntil = time.Now().Add(b.cooldown)
	}
}

func (b *Breaker) Current() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (s State) String() string {
	switch s {
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

type Registry struct {
	mu       sync.Mutex
	breakers map[string]*Breaker
	thresh   int
	cooldown time.Duration
}

func NewRegistry(threshold int, cooldown time.Duration) *Registry {
	return &Registry{
		breakers: make(map[string]*Breaker),
		thresh:   threshold,
		cooldown: cooldown,
	}
}

func (r *Registry) For(name string) *Breaker {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.breakers[name]
	if !ok {
		b = NewBreaker(r.thresh, r.cooldown)
		r.breakers[name] = b
	}
	return b
}
