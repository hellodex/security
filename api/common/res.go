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
