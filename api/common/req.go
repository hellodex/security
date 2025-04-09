package common

import (
	"github.com/shopspring/decimal"
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
	VaultTip  *big.Int `json:"vaultTip"`
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
	Amount                  *big.Int        `json:"amount"`
	FromTokenAddress        string          `json:"fromTokenAddress"`
	ToTokenAddress          string          `json:"toTokenAddress"`
	IsBuy                   bool            `json:"isBuy"`
	IsMemeVaultWalletTrade  bool            `json:"isMemeVaultWalletTrade"`
	Slippage                string          `json:"slippage"`
	ChainCode               string          `json:"chainCode"`
	ChainId                 string          `json:"chainId"`
	OrderNo                 string          `json:"orderNo"`
	UserWalletAddress       string          `json:"userWalletAddress"`
	ReqUri                  string          `json:"reqUri"`
	LimitOrderKey           string          `json:"limitOrderKey"`
	CurrTime                string          `json:"currTime"`
	FeeAccount              string          `json:"feeAccount"`
	FeeToken                string          `json:"feeToken"`
	ShouldOkx               bool            `json:"shouldOkx"`
	DynamicSlippage         bool            `json:"dynamicSlippage"`
	DynamicComputeUnitLimit bool            `json:"DynamicComputeUnitLimit"`
	AvgPrice                decimal.Decimal `json:"avgPrice"`
	FromTokenDecimals       int64           `json:"fromTokenDecimals"`
	ToTokenDecimals         int64           `json:"toTokenDecimals"`
	MemeVaultTrade          bool            `json:"memeVaultTrade"`
	JitoTipLamports         *big.Int        `json:"jitoTipLamports"`
}
