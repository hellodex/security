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

func AuthUserLoginCancel(c *gin.Context) {
	var req common.UserStructReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request:parameterFormatError"
		c.JSON(http.StatusOK, res)
		return
	}
	//校验账户类型
	_, exists := AccountTypeMap[req.AccountType]
	if !exists {
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
	// 目前只有email 有密码
	if req.AccountType != EMAIL &&
		(req.Password == "" || len(req.Password) < 1) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Password is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	//
	if req.AccountType != EMAIL &&
		(req.Captcha == "" || len(req.Captcha) < 1) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Captcha is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验验证码类型 验证码类型 1 登陆 2修改密码  3 注册/登陆 4 注册 5 转出代币 6 提取交易返佣 8 其他
	if req.CaptchaType == "" || len(req.CaptchaType) < 1 || req.CaptchaType == "0" {
		req.CaptchaType = C_LOGIN_REGISTER
	}
	db := system.GetDb()
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
	// 校验验证码
	switch req.AccountType {
	case EMAIL:
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

	case GOOGLE:
		// 处理GOOGLE类型的账号请求
	case APPLE:
		// 处理APPLE类型的账号请求
	case TWITTER:
		// 处理TWITTER类型的账号请求
	case TELEGRAM:
		// 处理TELEGRAM类型的账号请求
		v := VerifyUserTokenReq{
			Token:      req.Captcha,
			UserID:     req.UserNo,
			Channel:    req.Channel,
			ExpireTime: req.ExpireTime,
			ChainCodes: req.ChainCodes,
		}
		tokenValid, err2 := VerifyTGUserLoginToken(db, v)
		if err2 != nil {
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "校验失败" + err2.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		if !tokenValid {
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "校验失败"
			c.JSON(http.StatusOK, res)
			return
		}

	default:
		res.Code = codes.CODE_ERR
		res.Msg = "AccountType not supported"
		c.JSON(http.StatusOK, res)
		return
	}
	err = store.AuthAccountCancel(&authAccount)
	if err != nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
	return
}
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
	//校验账户类型
	_, exists := AccountTypeMap[req.AccountType]
	if !exists {
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
	// 目前只有email 有密码
	if req.AccountType != EMAIL &&
		(req.Password == "" || len(req.Password) < 1) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Password is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	//
	if req.AccountType != EMAIL &&
		(req.Captcha == "" || len(req.Captcha) < 1) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Captcha is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验验证码类型 验证码类型 1 登陆 2修改密码  3 注册/登陆 4 注册 5 转出代币 6 提取交易返佣 8 其他
	if req.CaptchaType == "" || len(req.CaptchaType) < 1 || req.CaptchaType == "0" {
		req.CaptchaType = C_LOGIN_REGISTER
	}
	db := system.GetDb()
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
	if authAccount.Status > 0 {
		res.Code = codes.CODE_ERR_4011
		res.Msg = "账户已关闭"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验验证码
	switch req.AccountType {
	case EMAIL:
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

		if authAccount.Token != password {
			res.Code = codes.CODE_ERR_4014
			res.Msg = "Invalid request: password is invalid"
			c.JSON(http.StatusOK, res)
			return
		}
	case GOOGLE:
		// 处理GOOGLE类型的账号请求
	case APPLE:
		// 处理APPLE类型的账号请求
	case TWITTER:
		// 处理TWITTER类型的账号请求
	case TELEGRAM:
		// 处理TELEGRAM类型的账号请求
		if req.CaptchaType == C_TG2WEB {
			v := VerifyUserTokenReq{
				Token:      req.Captcha,
				UserID:     req.UserNo,
				Channel:    req.Channel,
				ExpireTime: req.ExpireTime,
				ChainCodes: req.ChainCodes,
			}
			tokenValid, err2 := VerifyTGUserLoginToken(db, v)
			if err2 != nil {
				res.Code = codes.CODE_ERR_INVALID
				res.Msg = "校验失败" + err2.Error()
				c.JSON(http.StatusOK, res)
				return
			}
			if !tokenValid {
				res.Code = codes.CODE_ERR_INVALID
				res.Msg = "校验失败"
				c.JSON(http.StatusOK, res)
				return
			}
		}

	default:
		res.Code = codes.CODE_ERR
		res.Msg = "AccountType not supported"
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
	//校验账户类型
	_, exists := AccountTypeMap[req.AccountType]
	if !exists {
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
	// 目前只有email 有密码
	if req.AccountType != EMAIL &&
		(req.Password == "" || len(req.Password) < 1) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Password is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	//
	if req.AccountType != EMAIL &&
		(req.Captcha == "" || len(req.Captcha) < 1) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:Captcha is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验验证码类型 验证码类型 1 登陆 2修改密码  3 注册/登陆 4 注册 5 转出代币 6 提取交易返佣 8 其他
	if req.CaptchaType == "" || len(req.CaptchaType) < 1 || req.CaptchaType == "0" {
		req.CaptchaType = C_LOGIN_REGISTER
	}
	password := ""
	// 校验验证码
	switch req.AccountType {
	case EMAIL:
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
		password = hex.EncodeToString(hmac.Sum(nil))
	case GOOGLE:
		// 处理GOOGLE类型的账号请求
	case APPLE:
		// 处理APPLE类型的账号请求
	case TWITTER:
		// 处理TWITTER类型的账号请求
	case TELEGRAM:
		// 处理TELEGRAM类型的账号请求

	default:
		res.Code = codes.CODE_ERR
		res.Msg = "AccountType not supported"
		c.JSON(http.StatusOK, res)
		return
	}

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
		res.Msg = "账户已注册,请登录"
		c.JSON(http.StatusOK, res)
		return
	}
	uuid := common.GenerateSnowflakeId()
	user := &model.UserInfo{
		UUID:       uuid,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
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

// AccountType
var (
	PWD_KEY        = config.GetConfig().PwdKey
	AccountTypeMap = map[int]string{
		4: "APPLE",
		3: "GOOGLE",
		2: "TWITTER",
		1: "EMAIL",
		5: "TELEGRAM",
	}
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
	C_TG2WEB         = "9"
)
