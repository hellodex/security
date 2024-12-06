package common

import (
	"crypto/rand"
	"fmt"
	"github.com/oklog/ulid/v2"
	"sync"
	"time"
)

var (
	entropyLock sync.Mutex
)
var counter = NewCounter(999999)

func MyIDStr() string {
	return GenerateULID() + counter.Next()
}
func GenerateULID() string {
	entropyLock.Lock()
	defer entropyLock.Unlock()
	entropy := ulid.Monotonic(rand.Reader, 0)
	t := time.Now()
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

type Counter struct {
	mu       sync.Mutex
	value    int
	maxValue int
}

func NewCounter(maxValue int) *Counter {
	return &Counter{
		value:    0,
		maxValue: maxValue,
	}
}

func (c *Counter) Next() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Increment the counter
	c.value++
	if c.value > c.maxValue {
		c.value = 1 // Reset to 1 when exceeding maxValue
	}

	// Format as zero-padded string
	return fmt.Sprintf("%06d", c.value)
}
