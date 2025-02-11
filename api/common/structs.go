package common

type UserStructReq struct {
	UserNo      string   `json:"userNo"`
	Password    string   `json:"password"`
	Account     string   `json:"account"`
	AccountType int      `json:"accountType"`
	Captcha     string   `json:"captcha"`
	LoginType   int      `json:"loginType"`
	CaptchaType string   `json:"captchaType"`
	Channel     string   `json:"channel"`
	LeastGroups int      `json:"leastGroups"`
	ExpireTime  int64    `json:"expireTime"`
	ChainCodes  []string `json:"chainCodes"`
}

type AuthGetBackWallet struct {
	WalletAddr string `json:"walletAddr"`
	WalletId   uint64 `json:"walletId"`
	GroupID    uint64 `json:"groupId"`
	ChainCode  string `json:"chainCode"`
	WalletKey  string `json:"walletKey"`
	ExpireTime int64  `json:"expireTime"`
}
