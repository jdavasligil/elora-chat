package cache

import "unsafe"

type RingBuffer[T any] struct {
	back   int
	buf    []T
	length int
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buf: make([]T, capacity),
	}
}

func (rb *RingBuffer[T]) Push(t T) {
	rb.buf[rb.back] = t
	rb.back = (rb.back + 1) % cap(rb.buf)
	if rb.length < cap(rb.buf) {
		rb.length += 1
	}
}

func (rb *RingBuffer[T]) Pop() T {
	if rb.length == 0 {
		var noop T
		return noop
	}

	rb.length -= 1

	return rb.buf[(cap(rb.buf)+(rb.back-rb.length-1))%cap(rb.buf)]
}

func (rb *RingBuffer[T]) Peek() T {
	if rb.length == 0 {
		var noop T
		return noop
	}

	return rb.buf[(cap(rb.buf)+(rb.back-rb.length))%cap(rb.buf)]
}

func (rb *RingBuffer[T]) Len() int {
	return rb.length
}

func (rb *RingBuffer[T]) IsEmpty() bool {
	return rb.length == 0
}

func (rb *RingBuffer[T]) IsFull() bool {
	return rb.length == cap(rb.buf)
}

func (rb *RingBuffer[T]) MemUsage() uintptr {
	var typeT T
	return unsafe.Sizeof(*rb) + unsafe.Sizeof(typeT)*uintptr(cap(rb.buf))
}
