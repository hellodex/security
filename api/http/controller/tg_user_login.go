package controller

import (
	"errors"
	"fmt"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/store"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet"
	"gorm.io/gorm"
	"log"
	"net/http"
	"time"
)

type GetUserTokenReq struct {
	Channel   string `json:"channel"`
	AccountID string `json:"accountID"`
}

func GetUserLoginToken(c *gin.Context) {
	res := common.Response{}
	var req GetUserTokenReq
	appid, exists := c.Get("appId")
	if !exists {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "channel err, appid is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if appid == nil || appid.(string) != "tg" {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "channel err, appid is  not tg"
		c.JSON(http.StatusOK, res)
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()
	authAccounts, err1 := store.UserInfoGetByAccountId(req.AccountID, TELEGRAM)
	if err1 != nil || authAccounts == nil || len(authAccounts) <= 0 {
		res.Code = codes.CODE_ERR_102
		res.Msg = "authAccount Not Found"
		c.JSON(http.StatusOK, res)
		return
	}
	authAccount := authAccounts[0]
	// 校验账户是否被冻结
	if authAccount.Status > 0 {
		res.Code = codes.CODE_ERR_4011
		res.Msg = "账户已关闭"
		c.JSON(http.StatusOK, res)
		return
	}
	token := generateTGTemporaryToken(authAccount.UserUUID, db)
	func() {
		lock := common.GetLock("tg_user_login" + authAccount.UserUUID)
		lock.Lock.Lock()
		defer lock.Lock.Unlock()
		var userLogin model.TgLogin
		result := db.Model(&model.TgLogin{}).Where("account_id = ?", req.AccountID).First(&userLogin)
		//没有记录则创建
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userLogin = model.TgLogin{
				AccountID:    req.AccountID,
				Token:        token,
				GenerateTime: time.Now().Unix(),
				ExpireTime:   time.Now().Unix() + 60*5,
				IsUsed:       0,
			}
			userLogin.Token = token
			create := db.Create(&userLogin)
			if create.Error != nil {
				res.Code = codes.CODE_ERR_102
				res.Msg = "db error"
				c.JSON(http.StatusOK, res)
				return
			}
		}
		//过期则重新生成
		if userLogin.ExpireTime < time.Now().Unix() {
			userLogin.Token = token
			userLogin.ExpireTime = time.Now().Unix() + 60*5
			userLogin.GenerateTime = time.Now().Unix()
			err := db.Model(&model.TgLogin{}).Where("account_id = ?", req.AccountID).
				Updates(map[string]interface{}{
					"token":         userLogin.Token,
					"generate_time": userLogin.GenerateTime,
					"expire_time":   userLogin.ExpireTime,
					"is_used":       0}).Error
			if err != nil {
				mylog.Errorf("tg2web过期则重新生成 update token error: %v", err)
			}
			res.Code = codes.CODE_SUCCESS_200
			res.Data = userLogin.Token
			c.JSON(http.StatusOK, res)
			return
		}
		//使用过的token则重新生成
		if userLogin.IsUsed == 1 {
			userLogin.Token = token
			userLogin.ExpireTime = time.Now().Unix() + 60*5
			userLogin.GenerateTime = time.Now().Unix()
			userLogin.IsUsed = 0
			err := db.Model(&model.TgLogin{}).Where("account_id = ?", req.AccountID).
				Updates(map[string]interface{}{
					"token":         userLogin.Token,
					"generate_time": userLogin.GenerateTime,
					"expire_time":   userLogin.ExpireTime,
					"is_used":       0}).Error
			if err != nil {
				mylog.Errorf("使用过的token则重新生成 update token error: %v", err)
			}
			c.JSON(http.StatusOK, res)
			return
		}
		res.Data = userLogin.Token

	}()
	res.Code = codes.CODE_SUCCESS_200
	c.JSON(http.StatusOK, res)
}

type VerifyUserTokenReq struct {
	Token      string   `json:"token"`
	Channel    string   `json:"channel"`
	ExpireTime int64    `json:"expireTime"`
	ChainCodes []string `json:"chainCodes"`
}

func VerifyUserLoginToken(c *gin.Context) {

	res := common.Response{}
	var req VerifyUserTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}
	appid, exists := c.Get("appId")
	if !exists {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "channel err, appid is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if appid == nil || appid.(string) != "tg" {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "channel err, appid is  not tg"
		c.JSON(http.StatusOK, res)
		return
	}
	db := system.GetDb()
	lock := common.GetLock("VerifyUserLoginToken" + req.Token)
	lock.Lock.Lock()
	defer lock.Lock.Unlock()
	tokenValidAccountId, err2 := VerifyTGUserLoginToken(db, req)
	if err2 != nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = err2.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	if len(tokenValidAccountId) <= 0 {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "请重新通过TG Bot登录,code:4005"
		c.JSON(http.StatusOK, res)
		return
	}

	as, err2 := store.UserInfoGetByAccountId(tokenValidAccountId, TELEGRAM)
	if err2 != nil || as == nil || as[0].ID <= 0 {
		res.Code = codes.CODE_ERR_4011
		res.Msg = "获取用户信息失败" + err2.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	authAccount := as[0]
	// 校验账户是否被冻结
	if authAccount.Status > 0 {
		res.Code = codes.CODE_ERR_4011
		res.Msg = "账户已关闭"
		c.JSON(http.StatusOK, res)
		return
	}
	authAccount.Token = ""
	authAccount.SecretKey = ""
	res.Code = codes.CODE_ERR_102
	validChains := wallet.CheckAllCodes(req.ChainCodes)
	reqUser := common.UserStructReq{
		Uuid:        authAccount.UserUUID,
		LeastGroups: 2,
		Channel:     req.Channel,
		ExpireTime:  req.ExpireTime,
	}
	no, err := GetWalletByUserNo(db, &reqUser, validChains, appid)
	if err != nil {
		log.Printf("获取用户的钱包列表失败:%v", err)
	}
	authAccount.Wallets = no
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = authAccount
	c.JSON(http.StatusOK, res)
}
func VerifyTGUserLoginToken(db *gorm.DB, req VerifyUserTokenReq) (string, error) {
	lock := common.GetLock("VerifyTgUserLoginToken" + req.Token)
	lock.Lock.Lock()
	defer lock.Lock.Unlock()
	var userLogin model.TgLogin
	err2 := db.Model(&model.TgLogin{}).Where("token", req.Token).First(&userLogin).Error
	if err2 != nil {
		//是否有此token
		if errors.Is(err2, gorm.ErrRecordNotFound) {
			return "", errors.New("请重新通过TG Bot登录,code:4001")
		} else {
			mylog.Errorf("校验查询TG2WEBtoken失败: %v", err2)
			return "", errors.New("TG Bot登录失败,请联系客服,code:4002")
		}
	}
	if userLogin.ID <= 0 {
		mylog.Errorf("校验查询TG2WEBtoken失败: %s", "userLogin.ID <= 0")
		return "", errors.New("请重新通过TG Bot登录")
	}

	//用过的token
	if userLogin.IsUsed == 1 {
		return "", errors.New("请重新通过TG Bot登录,code:4003")
	}
	//过期
	if userLogin.ExpireTime < time.Now().Unix() {
		return "", errors.New("请重新通过TG Bot登录,code:4004")
	}
	//通过更新状态
	err := db.Model(&model.TgLogin{}).Where("account_id = ? AND token = ?", userLogin.AccountID, userLogin.Token).
		Updates(map[string]interface{}{"is_used": 1}).Error
	//验证通过
	if err != nil {
		mylog.Errorf("verify token error:Updated is_used error: %v", err)
	}
	return userLogin.AccountID, nil
}

func generateTGTemporaryToken(UUID string, db *gorm.DB) string {
	const maxAttempts = 20 // 最大尝试次数，防止死循环

	for attempts := 0; attempts < maxAttempts; attempts++ {
		newToken := cryptor.Base64StdEncode(fmt.Sprintf("%s%s%d", common.RandomStr(30), UUID, time.Now().Unix()))
		// 检查邀请码是否已存在
		var t model.TgLogin
		err := db.Model(&model.TgLogin{}).Where("token = ?", newToken).First(&t).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) { // 如果邀请码不存在，返回该邀请码
				return newToken
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
