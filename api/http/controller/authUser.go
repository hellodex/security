package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/store"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet"
	"log"
	"net/http"
	"strings"
	"time"
)

func AuthUserLogin(c *gin.Context) {
	var req common.UserStructReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request:parameterFormatError"
		c.JSON(http.StatusOK, res)
		return
	}
	//todo 暂时只支持邮箱登陆
	if req.AccountType != EMAIL {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:AccountType not supported"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Account == "" || len(req.Account) < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:account is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Password == "" || len(req.Password) < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Password is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Captcha == "" || len(req.Captcha) < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Captcha is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验验证码类型 验证码类型 1 登陆 2修改密码  3 注册/登陆 4 注册 5 转出代币 6 提取交易返佣 8 其他
	if req.CaptchaType == "" || len(req.CaptchaType) < 1 || req.CaptchaType == "0" {
		req.CaptchaType = C_LOGIN_REGISTER
	}
	// 校验验证码

	v := VerifyMailReq{
		Captcha: req.Captcha,
		Account: req.Account,
		Type:    req.CaptchaType,
	}
	verifyRes := system.VerifyCode(v.Account+v.Type, v.Captcha)
	if !verifyRes {
		res.Code = codes.CODE_ERR_4013
		res.Msg = "Invalid request:Captcha is invalid"
		c.JSON(http.StatusOK, res)
		return
	}
	hmac := hmac.New(sha256.New, []byte(PWD_KEY))
	hmac.Write([]byte(req.Password))
	password := hex.EncodeToString(hmac.Sum(nil))
	accountsIndb, err := store.UserInfoGetByAccountId(req.Account, req.AccountType)
	if err != nil || len(accountsIndb) <= 0 {
		if req.Captcha == "" || len(req.Captcha) < 1 {
			res.Code = codes.CODE_ERR_4011
			res.Msg = "Invalid request:not found user"
			c.JSON(http.StatusOK, res)
			return
		}
	}
	authAccount := accountsIndb[0]
	if authAccount.Token != password {
		res.Code = codes.CODE_ERR_4014
		res.Msg = "Invalid request: password is invalid"
		c.JSON(http.StatusOK, res)
		return
	}
	validChains := wallet.CheckAllCodes(req.ChainCodes)
	//返回的钱包列表
	channelw, _ := c.Get("APP_ID")
	//获取用户的钱包列表
	no, err := GetWalletByUserNo(system.GetDb(), &req, validChains, channelw)
	if err != nil {
		log.Printf("获取用户的钱包列表失败:%v", err)
	}
	authAccount.Wallets = no
	authAccount.Token = ""
	authAccount.SecretKey = ""
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = authAccount
	c.JSON(http.StatusOK, res)
	return
}
func AuthUserRegister(c *gin.Context) {
	var req common.UserStructReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request:parameterFormatError"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.AccountType != EMAIL {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:AccountType not supported"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Account == "" || len(req.Account) < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:account is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Password == "" || len(req.Password) < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Password is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Captcha == "" || len(req.Captcha) < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Captcha is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验验证码类型 验证码类型 1 登陆 2修改密码  3 注册/登陆 4 注册 5 转出代币 6 提取交易返佣 8 其他
	if req.CaptchaType == "" || len(req.CaptchaType) < 1 || req.CaptchaType == "0" {
		req.CaptchaType = C_LOGIN_REGISTER
	}
	// 校验验证码

	v := VerifyMailReq{
		Captcha: req.Captcha,
		Account: req.Account,
		Type:    req.CaptchaType,
	}
	verifyRes := system.VerifyCode(v.Account+v.Type, v.Captcha)
	if !verifyRes {
		res.Code = codes.CODE_ERR_4013
		res.Msg = "Invalid request:Captcha is invalid"
		c.JSON(http.StatusOK, res)
		return
	}
	hmac := hmac.New(sha256.New, []byte(PWD_KEY))
	hmac.Write([]byte(req.Password))
	password := hex.EncodeToString(hmac.Sum(nil))
	accountsIndb, err := store.UserInfoGetByAccountId(req.Account, req.AccountType)
	if err != nil && !strings.HasSuffix(err.Error(), "record not found") {
		if req.Captcha == "" || len(req.Captcha) < 1 {
			res.Code = codes.CODE_ERR
			res.Msg = "Invalid request:sql error:" + err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
	}
	if len(accountsIndb) >= 0 {
		res.Code = codes.CODE_ERR_4018
		res.Msg = "Invalid request:sql error:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	code := generateInvitationCode()
	uuid := common.GenerateSnowflakeId()
	user := &model.UserInfo{
		UUID:           uuid,
		CreateTime:     time.Now(),
		InvitationCode: code,
		UpdateTime:     time.Now(),
	}
	err = store.UserInfoSave(user)
	if err != nil {
		res.Code = codes.CODE_ERR_4016
		res.Msg = "创建用户失败UserInfoSave:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	authAccount := &model.AuthAccount{
		UserUUID:    uuid,
		AccountID:   req.Account,
		AccountType: req.AccountType,
		Token:       password,
		Status:      0,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}
	err = store.AuthAccountSave(authAccount)
	if err != nil {

		res.Code = codes.CODE_ERR_4016
		res.Msg = "创建用户失败AuthAccountSave:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	authAccount.InvitationCode = code
	authAccount.Token = ""
	authAccount.SecretKey = ""
	validChains := wallet.CheckAllCodes(req.ChainCodes)
	//返回的钱包列表
	channelw, _ := c.Get("APP_ID")
	//获取用户的钱包列表
	no, err := GetWalletByUserNo(system.GetDb(), &req, validChains, channelw)
	if err != nil {
		log.Printf("获取用户的钱包列表失败:%v", err)
	}
	authAccount.Wallets = no
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = authAccount
	c.JSON(http.StatusOK, res)
	return
}

func generateInvitationCode() string {
	const maxAttempts = 100 // 最大尝试次数，防止死循环

	for attempts := 0; attempts < maxAttempts; attempts++ {
		str := common.RandomStr(8) // 生成8位随机邀请码
		// 检查邀请码是否已存在
		_, err := store.UserInfoGetByInvitationCode(str)
		if err != nil {
			if strings.HasSuffix(err.Error(), "record not found") {
				// 如果邀请码不存在，返回该邀请码
				return str
			} else {
				// 记录其他数据库错误
				log.Printf("数据库查询出错: %v", err)
			}
		}
	}

	// 超过最大尝试次数，返回错误或空字符串
	log.Fatalf("超过最大尝试次数，无法生成唯一的邀请码")
	return ""
}

// AccountType
var (
	PWD_KEY  = config.GetConfig().PwdKey
	APPLE    = 4
	GOOGLE   = 3
	TWITTER  = 2
	EMAIL    = 1
	TELEGRAM = 5
) // AccountType
var (
	LoginTypeRegister = 1
	LoginTypeLogin    = 0
)
var (
	/* CaptchaTypeEnum
	   	 验证码类型 1 登陆 2修改密码  3 注册登陆 4 注册 5 转出代币 6 提取交易返佣 8 其他
	   LOGIN("1", "login"),
	   MODIFY("2", "modify"),
	   LOGIN_REGISTER("3", "login_register"),
	   REGISTER("4", "register"),
	   WITHDRAW("5", "withdraw"),
	   commission("6", "commission"),
	   CODE("8", "code");
	*/
	C_LOGIN          = "1"
	C_MODIFY         = "2"
	C_LOGIN_REGISTER = "3"
	C_REGISTER       = "4"
	C_WITHDRAW       = "5"
	C_commission     = "6"
	C_CODE           = "8"
)
