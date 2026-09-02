package circuitbreaker

import (
	"testing"
	"time"
)

func TestOpensAfterThreshold(t *testing.T) {
	b := NewBreaker(3, time.Hour)
	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatal("closed should allow")
		}
		b.RecordFailure()
	}
	if b.Allow() {
		t.Fatal("open should reject")
	}
	if b.Current() != StateOpen {
		t.Fatal(b.Current())
	}
}

func TestSuccessResets(t *testing.T) {
	b := NewBreaker(3, time.Hour)
	b.RecordFailure()
	b.RecordFailure()
	b.RecordSuccess()
	b.RecordFailure()
	if !b.Allow() {
		t.Fatal("counter should have reset")
	}
}

func TestHalfOpenOneProbe(t *testing.T) {
	b := NewBreaker(1, 20*time.Millisecond)
	b.RecordFailure()
	if b.Allow() {
		t.Fatal("still in cooldown")
	}
	time.Sleep(30 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("first probe should pass")
	}
	if b.Allow() {
		t.Fatal("second concurrent probe should not")
	}
}

func TestRegistrySameInstance(t *testing.T) {
	r := NewRegistry(5, time.Second)
	if r.For("orders") != r.For("orders") {
		t.Fatal("expected same breaker")
	}
}