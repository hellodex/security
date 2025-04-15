package common

import (
	"fmt"
	"github.com/suanju/googleAuthenticator"
)

type TwoFA struct {
	Secret     string `json:"secret"`
	CodeLength int
	QRCode     string `json:"qrcode"`
}

func TwoFACreateSecret(length int, account string) *TwoFA {
	authenticator := googleAuthenticator.NewGoogleAuthenticator(6)
	if length < 16 {
		length = 16
	}
	// 创建一个 16 字节的随机密钥
	secret, err := authenticator.CreateSecret(length)
	if err != nil {
		fmt.Println("创建密钥时出错:", err)
		return nil
	}
	code, err := authenticator.GenerateQRCode("helloAuth:"+account, secret)
	if err != nil {
		fmt.Println("创建二维码时出错:", err)
	}

	return &TwoFA{
		Secret:     secret,
		CodeLength: 6,
		QRCode:     code,
	}
}
func TwoFAVerifyCode(secret string, code string, currentTime int64) bool {
	authenticator := googleAuthenticator.NewGoogleAuthenticator(6)
	verifyCode := authenticator.VerifyCode(secret, code, 0, currentTime)
	return verifyCode
}
