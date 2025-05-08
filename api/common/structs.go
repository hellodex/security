package common

import (
	"github.com/shopspring/decimal"
	"math/big"
	"time"
)

type UserStructReq struct {
	Uuid        string   `json:"uuid"`
	Password    string   `json:"password"`
	Account     string   `json:"account"`
	AccountType int      `json:"accountType"`
	Captcha     string   `json:"captcha"`
	LoginType   int      `json:"loginType"`
	CaptchaType string   `json:"captchaType"`
	Channel     string   `json:"channel"`
	LeastGroups int      `json:"leastGroups"` //最少多少钱包 默认2
	ExpireTime  int64    `json:"expireTime"`  //过期时间 Tg默认超长过期时间
	ChainCodes  []string `json:"chainCodes"`  //链码列表
}
type AdminStructReq struct {
	Admin     string `gorm:"-" json:"admin"`
	TwoFACode string `gorm:"-" json:"twoFACode"`
	Msg       string `gorm:"-" json:"msg"`
}

type AuthGetBackWallet struct {
	WalletAddr string `json:"walletAddr"`
	WalletId   uint64 `json:"walletId"`
	GroupID    uint64 `json:"groupId"`
	VaultType  int    `json:"vaultType"`
	ChainCode  string `json:"chainCode"`
	WalletKey  string `json:"walletKey"`
	ExpireTime int64  `json:"expireTime"`
}
type TransferParsed struct {
	Tx         string          `json:"tx"`
	From       string          `json:"from"`
	To         string          `json:"to"`
	Amount     *big.Int        `json:"amount"`
	Contract   string          `json:"contract"`
	Symbol     string          `json:"symbol"`
	Decimals   uint8           `json:"decimals"`
	Block      uint64          `json:"block"`
	BlockTime  uint64          `json:"blockTime"`
	Price      decimal.Decimal `json:"price"`
	GasUsed    uint64          `json:"gasUsed"`
	GasPrice   uint64          `json:"gasPrice"`
	ChainCode  string          `json:"chainCode"`
	ParsedTime time.Time       `json:"parsedTime"`
}
type TokenHold struct {
	Address      string          `json:"address"`
	TokenAddress string          `json:"tokenAddress"`
	Amount       decimal.Decimal `json:"amount"`
	Decimals     uint8           `json:"decimals"`
}
type TempAccountData struct {
	Account  string
	Mint     string
	Owner    string
	Decimals uint8
}
