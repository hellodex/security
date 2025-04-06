package common

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

type AuthGetBackWallet struct {
	WalletAddr string `json:"walletAddr"`
	WalletId   uint64 `json:"walletId"`
	GroupID    uint64 `json:"groupId"`
	VaultType  int    `json:"vaultType"`
	ChainCode  string `json:"chainCode"`
	WalletKey  string `json:"walletKey"`
	ExpireTime int64  `json:"expireTime"`
}
