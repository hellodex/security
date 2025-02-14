package common

import (
	"math/big"
)

const (
	TYPE_CHAT_INITIAL = "chat_init"
	TYPE_CHAT_APPEND  = "chat_follow"
	METHOD_GPT        = "chatGPT"

	CODE_DIRECTION_IN  = "1"
	CODE_DIRECTION_OUT = "2"
)

type Request struct {
	Type      string `json:"type"`
	Method    string `json:"method"`
	Timestamp int64  `json:"timestamp"`
	Ascode    string `json:"ascode"`
	Lan       string `json:"lan"`
	DevId     string `json:"devid"`
	UserId    uint64 `json:"userid"`
	Data      string `json:"data"`
}

type OpConfig struct {
	UnitPrice *big.Int `json:"unit_price"`
	UnitLimit *big.Int `json:"unit_limit"`
	Rpc       string   `json:"rpc"`
	Type      string   `json:"type"`
	Tip       *big.Int `json:"tip"`
}
type AuthSigWalletRequest struct {
	UserId           string          `json:"userId"`
	Message          string          `json:"message"`
	Type             string          `json:"type"`
	WalletKey        string          `json:"walletKey"`
	To               string          `json:"to"`
	Amount           *big.Int        `json:"amount"`
	Channel          string          `json:"channel"`
	Config           OpConfig        `json:"config"`
	LimitOrderParams LimitOrderParam `json:"limitOrderParam"`
}
type LimitOrderParam struct {
	Amount            uint64 `json:"amount"`
	FromTokenAddress  string `json:"fromTokenAddress"`
	ToTokenAddress    string `json:"toTokenAddress"`
	Slippage          string `json:"slippage"`
	ChainCode         string `json:"chainCode"`
	OrderNo           string `json:"orderNo"`
	UserWalletAddress string `json:"userWalletAddress"`
	ReqUri            string `json:"reqUri"`
	LimitOrderKey     string `json:"limitOrderKey"`
	CurrTime          string `json:"currTime"`
}
