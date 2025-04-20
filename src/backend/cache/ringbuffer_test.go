package cache

import (
	"testing"
)

func TestRingBuffer(t *testing.T) {
	maxcap := 16
	rb := NewRingBuffer[int](maxcap)
	if rb == nil {
		t.Error("Failed to create ring buffer.")
	}
	if rb.Len() != 0 {
		t.Errorf("Initial length is not zero. Len: %d\n", rb.Len())
	}
	for i := 1; i <= maxcap; i++ {
		rb.Push(i)
	}
	if rb.Len() != maxcap {
		t.Errorf("Max capacity length is not %d. Len: %d\n", maxcap, rb.Len())
	}
	v := rb.Pop()
	if v != 1 {
		t.Errorf("Expected Pop: 1. Got %d", v)
	}
	if rb.Peek() != 2 {
		t.Errorf("Expected Peek: 2. Got %d", v)
	}
	v = rb.Pop()
	if v != 2 {
		t.Errorf("Expected Pop: 2. Got %d", v)
	}
	for i := maxcap + 1; i <= maxcap+3; i++ {
		rb.Push(i)
	}
	if rb.Peek() != 4 {
		t.Errorf("Expected Peek: 4. Got %d", v)
	}

	for rb.Len() > 0 {
		rb.Pop()
	}
}
