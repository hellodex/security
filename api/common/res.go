package common

type Response struct {
	Code      int64       `json:"code"`
	Msg       string      `json:"msg"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}
type SignRes struct {
	Signature string                 `json:"signature"`
	Wallet    string                 `json:"wallet"`
	Tx        string                 `json:"tx"`
	CallData  map[string]interface{} `json:"callData"`
}
type PaginatedResult[T any] struct {
	Page      int // 当前页
	PageSize  int // 每页记录数
	TotalPage int // 总页数
	Data      []T // 当前页数据
}
