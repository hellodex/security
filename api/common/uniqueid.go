package common

import (
	"crypto/rand"
	"fmt"
	"github.com/oklog/ulid/v2"
	random "math/rand"
	"sync"
	"time"
)

const charset = "ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"

var (
	entropyLock   sync.Mutex
	SnowflakeLock sync.Mutex
	randomStrLock sync.Mutex
)
var counter = NewCounter(999999)
var snowflake, _ = NewNode(1)

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

// 雪花算法生成ID
func GenerateSnowflakeId() string {
	SnowflakeLock.Lock()
	defer SnowflakeLock.Unlock()
	time.Sleep(1 * time.Microsecond)
	return snowflake.Generate().String()
}

// generateInvitationCode 生成指定长度的邀请码，允许大写和小写字母以及数字，但排除容易混淆的字符
func RandomStr(length int) string {
	randomStrLock.Lock()
	defer randomStrLock.Unlock()
	// 定义允许使用的字符集：
	// 大写字母：排除了 "I" 和 "O"
	// 小写字母：排除了 "l" 和 "o"
	// 数字：排除了 "0" 和 "1"
	code := make([]byte, length)
	for i := 0; i < length; i++ {
		index := random.Intn(len(charset))
		code[i] = charset[index]
	}
	return string(code)
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
