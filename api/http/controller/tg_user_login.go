package controller

import (
	"errors"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/duke-git/lancet/v2/random"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"time"
)

type GetUserTokenReq struct {
	Channel string `json:"channel"`
	UserID  string `json:"userId"`
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
	func() {
		lock := common.GetLock("tg_user_login" + req.UserID)
		lock.Lock.Lock()
		defer lock.Lock.Unlock()
		var userLogin model.TgLogin
		result := db.Model(&model.TgLogin{}).Where("tg_user_id = ?", req.UserID).First(&userLogin)
		//没有记录则创建
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			token, _ := generateToken(userLogin)
			userLogin = model.TgLogin{
				TgUserId:     req.UserID,
				Token:        token,
				GenerateTime: time.Now().Unix(),
				ExpireTime:   time.Now().Unix() + 60*5,
				IsUsed:       0,
			}
			userLogin.Token, _ = generateToken(userLogin)
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
			userLogin.Token, _ = generateToken(userLogin)
			userLogin.ExpireTime = time.Now().Unix() + 60*5
			userLogin.GenerateTime = time.Now().Unix()
			err := db.Model(&model.TgLogin{}).Where("tg_user_id = ?", req.UserID).
				Updates(map[string]interface{}{
					"token":         userLogin.Token,
					"generate_time": userLogin.GenerateTime,
					"expire_time":   userLogin.ExpireTime,
					"is_used":       0}).Error
			if err != nil {
				mylog.Errorf("tg2web过期则重新生成 update token error: %v", err)
			}
			res.Data = userLogin.Token
			c.JSON(http.StatusOK, res)
			return
		}
		//使用过的token则重新生成
		if userLogin.IsUsed == 1 {
			userLogin.Token, _ = generateToken(userLogin)
			userLogin.ExpireTime = time.Now().Unix() + 60*5
			userLogin.GenerateTime = time.Now().Unix()
			userLogin.IsUsed = 0
			err := db.Model(&model.TgLogin{}).Where("tg_user_id = ?", req.UserID).
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

	c.JSON(http.StatusOK, res)
}

type VerifyUserTokenReq struct {
	Token      string   `json:"token"`
	UserID     string   `json:"userId"`
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
	lock := common.GetLock("VerifyUserLoginToken" + req.UserID)
	lock.Lock.Lock()
	defer lock.Lock.Unlock()
	db := system.GetDb()
	var userLogin model.TgLogin
	result := db.Model(&model.TgLogin{}).Where("tg_user_id = ?", req.UserID).Where("token", req.Token).First(&userLogin)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "token not exist"
		c.JSON(http.StatusOK, res)
		return
	}
	//用过的token
	if userLogin.IsUsed == 1 {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "token already used"
		c.JSON(http.StatusOK, res)
		return
	}
	//过期
	if userLogin.ExpireTime < time.Now().Unix() {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "token expired"
		c.JSON(http.StatusOK, res)
		return
	}
	err := db.Model(&model.TgLogin{}).Where("tg_user_id = ? AND token = ?", userLogin.TgUserId, userLogin.Token).
		Updates(map[string]interface{}{"is_used": 1}).Error
	//验证通过
	if err != nil {
		mylog.Errorf("verify token error:Updated is_used error: %v", err)
	}
	reqUser := common.UserStructReq{
		UserNo:      req.UserID,
		LeastGroups: 2,
		Channel:     req.Channel,
		ExpireTime:  req.ExpireTime,
	}
	validChains := wallet.CheckAllCodes(req.ChainCodes)
	no, err := GetWalletByUserNo(db, &reqUser, validChains, appid)
	if err != nil {
		log.Printf("获取用户的钱包列表失败:%v", err)
	}

	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = no
	c.JSON(http.StatusOK, res)
}

func generateToken(login model.TgLogin) (string, error) {
	//用时间戳和用户id进行base64编码
	randInt := random.RandInt(0, 1000000)
	s := strconv.FormatInt(int64(randInt), 10)
	formatInt := strconv.FormatInt(time.Now().Unix(), 10)
	token := cryptor.Base64StdEncode(login.TgUserId + formatInt + s)
	return token, nil
}
