package controller

import (
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	logs1 "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/jordan-wright/email"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
)

var template = `<!DOCTYPE html>
            <html lang="zh">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>邮箱验证码</title>
                <style>
                    body {
                        font-family: 'Arial', sans-serif;
                        background-color: #e9ecef;
                        padding: 20px;
                        margin: 0;
                    }
                    .container {
                        max-width: 500px;
                        margin: auto;
                        background-color: #ffffff;
                        border-radius: 10px;
                        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
                        padding: 30px;
                        text-align: center;
                    }
                    h1 {
                        color: #343a40;
                    }
                    .code {
                        font-size: 32px;
                        font-weight: bold;
                        color: #28a745;
                        margin: 20px 0;
                    }
                    p {
                        color: #495057;
                    }
                    .footer {
                        margin-top: 30px;
                        font-size: 12px;
                        color: #6c757d;
                    }
                </style>
            </head>
            <body>
                <div class="container">
                    <h1>登录/注册验证码确认</h1>
                    <p>尊敬的用户，您的验证码是：</p>
                    <div class="code">{code}</div>
                    <p>请在 3 分钟内输入该验证码完成验证。</p>
                    <p>感谢您的支持！</p>
                    <div class="footer">如果您没有请求此操作，请忽略此邮件。</div>
                </div>
            </body>
            </html> `

type MailReq struct {
	SendTo string `json:"sendTo"`

	Subject string `json:"subject"`

	Text string `json:"text"`

	Type string `json:"type"`
}

func SendEmail(c *gin.Context) {

	var reqBody MailReq
	res := common.Response{}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	go func() {
		e := email.NewEmail()
		emailConfig := config.GetConfig().Mail
		code := system.GenCode(reqBody.SendTo, reqBody.Type)
		replace := strings.Replace(template, "{code}", code, -1)
		e.From = emailConfig.Name + "<" + emailConfig.Sender + ">"
		e.To = []string{reqBody.SendTo}
		e.Subject = reqBody.Subject
		e.HTML = []byte(replace)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         emailConfig.Host,
		}
		conn, err := tls.Dial("tcp", emailConfig.Host+":"+strconv.Itoa(emailConfig.Port), tlsConfig)
		if err != nil {
			logs1.Logger.Error("Failed to tls.Dial: %v", err)
			c.JSON(http.StatusOK, gin.H{"error": err.Error()})
			return
		}
		client, err := smtp.NewClient(conn, emailConfig.Host)

		if err != nil {
			logs1.Logger.Error("Failed to smtp.NewClient: %v", err)
			c.JSON(http.StatusOK, gin.H{"error": err.Error()})
			return
		}
		auth := smtp.PlainAuth("", emailConfig.UserName, emailConfig.Password, emailConfig.Host)
		if err = client.Auth(auth); err != nil {
			logs1.Logger.Error("Failed to authenticate: %v", err)
		}

		if err = client.Mail(emailConfig.Sender); err != nil {
			logs1.Logger.Error("Failed to set sender: %v", err)

		}

		if err = client.Rcpt(reqBody.SendTo); err != nil {
			logs1.Logger.Error("Failed to set recipient: %v", err)
		}

		writer, err := client.Data()
		if err != nil {
			logs1.Logger.Error("Failed to open data writer: %v", err)
		}
		bytes, _ := e.Bytes()
		_, err = writer.Write(bytes)
		if err != nil {
			logs1.Logger.Error("Failed to write body: %v", err)
		}

		err = writer.Close()
		if err != nil {
			logs1.Logger.Error("Failed to close writer: %v", err)
		}
	}()
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
	return
}

type VerifyMailReq struct {
	Account string `json:"account"`

	Captcha string `json:"captcha"`

	Type string `json:"type"`
}

func VerifyCode(c *gin.Context) {
	var reqBody VerifyMailReq
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": "400", "msg": "params error"})
		return
	}
	res := system.VerifyCode(reqBody.Account+reqBody.Type, reqBody.Captcha)
	if res {
		c.JSON(http.StatusOK, gin.H{"code": "200", "msg": "success"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "400", "msg": "fail"})
	return

}
