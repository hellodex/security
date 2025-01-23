package controller

import (
	"errors"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"gorm.io/gorm"
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
		db.Save(&userLogin)
		c.JSON(http.StatusOK, res)
		return
	}
	//使用过的token则重新生成
	if userLogin.IsUsed == 1 {
		userLogin.Token, _ = generateToken(userLogin)
		userLogin.ExpireTime = time.Now().Unix() + 60*5
		userLogin.GenerateTime = time.Now().Unix()
		userLogin.IsUsed = 0
		db.Save(&userLogin)
		c.JSON(http.StatusOK, res)
		return
	}
	res.Data = userLogin.Token
	c.JSON(http.StatusOK, res)
}

type VerifyUserTokenReq struct {
	Token  string `json:"token"`
	UserID string `json:"userId"`
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
	//验证通过
	userLogin.IsUsed = 1
	db.Save(&userLogin)
	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	c.JSON(http.StatusOK, res)

}

func generateToken(login model.TgLogin) (string, error) {
	//用时间戳和用户id进行base64编码
	formatInt := strconv.FormatInt(time.Now().Unix(), 10)
	token := cryptor.Base64StdEncode(login.TgUserId + formatInt)
	return token, nil
}
