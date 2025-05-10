package common

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// CustomTime 包装 time.Time
type CustomTime struct {
	time.Time
}

// MarshalJSON 自定义 JSON 序列化，输出 RFC 3339 格式
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.Time.Format(time.DateTime))
}

// UnmarshalJSON 自定义 JSON 反序列化，尝试多种时间格式
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("CustomTime: cannot unmarshal %s into string: %v", string(data), err)
	}

	// 定义支持的时间格式
	formats := []string{
		time.RFC3339,                  // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,              // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05",         // "2006-01-02T15:04:05" (无时区，假设 UTC)
		"2006-01-02 15:04:05",         // "2006-01-02 15:04:05" (无时区，假设 UTC)
		"2006-01-02T15:04:05.999Z",    // "2006-01-02T15:04:05.999Z" (毫秒)
		"2006-01-02T15:04:05.999999Z", // "2006-01-02T15:04:05.999999Z" (微秒)
	}

	// 尝试每种格式解析
	for _, format := range formats {
		if parsed, err := time.Parse(format, s); err == nil {
			ct.Time = parsed.UTC() // 统一转换为 UTC
			return nil
		}
	}

	return fmt.Errorf("CustomTime: cannot parse %s as any supported time format", s)
}

// Value 实现 driver.Valuer 接口，用于数据库写入
func (ct CustomTime) Value() (driver.Value, error) {
	return ct.Time, nil
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
func (ct *CustomTime) Scan(value interface{}) error {
	if value == nil {
		ct.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		ct.Time = v
		return nil
	}
	return fmt.Errorf("cannot scan %T into CustomTime", value)
}
