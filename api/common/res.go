package common

import "github.com/shopspring/decimal"

type Response struct {
	Code      int64       `json:"code"`
	Msg       string      `json:"msg"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}
type SignRes struct {
	Signature   string                 `json:"signature"`
	Wallet      string                 `json:"wallet"`
	Tx          string                 `json:"tx"`
	CallData    map[string]interface{} `json:"callData"`
	UserReceive decimal.Decimal        `json:"userReceive"`
}
type PaginatedResult[T any] struct {
	Current int `json:"current"` // 当前页
	Size    int `json:"size"`    // 每页记录数
	Total   int `json:"total"`   // 总页数
	Records []T `json:"records"` // 当前页数据
}
