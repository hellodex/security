package controller

import (
	md5 "crypto/md5"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/store"
	"net/http"
	"time"
)

func CreateLimitKey(c *gin.Context) {
	var req CreateLimitKeyRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}
	limitOrderKey := req.LimitOrderKey()
	//校验
	wk, err := store.WalletKeyCheckAndGet(req.WalletKey)
	if err != nil || wk == nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	wks := model.LimitKeys{
		LimitKey: limitOrderKey,
		WalletID: wk.WalletId,
	}
	err = store.LimitKeySave(wks)
	if err != nil {
		if err != nil || wk == nil {
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "Invalid request"
			c.JSON(http.StatusOK, res)
			return
		}
	}
	//返回limitKey
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = LimitKeyResponse{LimitKey: limitOrderKey}
	c.JSON(http.StatusOK, res)
	return
}

func DelLimitOrderKeys(c *gin.Context) {
	var req LimitKeyResponse
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}
	//校验
	err := store.LimitKeyDelByKey(req.LimitKey)
	if err != nil {
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
	return
}

type CreateLimitKeyRequest struct {
	FromTokenAddrsss string `json:"fromTokenAddress"`
	WalletKey        string `json:"walletKey"`
	ToTokenAddress   string `json:"toTokenAddress"`
	OrderNo          string `json:"orderNo"`
	Channel          string `json:"channel"`
}

type LimitKeyResponse struct {
	LimitKey string `json:"limitKey"`
}

func (req *CreateLimitKeyRequest) LimitOrderKey() string {
	bytes := md5.Sum([]byte(req.FromTokenAddrsss + req.ToTokenAddress + req.OrderNo))
	return fmt.Sprintf("%x", bytes)
}
