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
	Page      int `json:"page"`      // 当前页
	PageSize  int `json:"pageSize"`  // 每页记录数
	TotalPage int `json:"totalPage"` // 总页数
	Data      []T `json:"data"`      // 当前页数据
}
