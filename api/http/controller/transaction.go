package controller

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/hellodex/HelloSecurity/store"

	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/chain"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
)

type TokenTransfer struct {
	WalletID uint64          `json:"wallet_id"`
	Token    string          `json:"token"`
	To       string          `json:"to"`
	Amount   *big.Int        `json:"amount"`
	Config   common.OpConfig `json:"config"`
}

type TokenTransferReq struct {
	WalletKey string          `json:"walletKey"`
	Token     string          `json:"token"`
	To        string          `json:"to"`
	Amount    *big.Int        `json:"amount"`
	Config    common.OpConfig `json:"config"`
	UserID    string          `json:"userId"`
	Channel   string          `json:"channel"`
	ChainCode string          `json:"chainCode"`
	Admin     string          `json:"admin"`
	TwoFACode string          `json:"twoFACode"`
}

func AuthTransfer(c *gin.Context) {
	var req TokenTransferReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.To) == 0 {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Channel == "" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters: channel is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.UserID == "" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters: userId is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	wk, err2 := store.WalletKeyCheckAndGet(req.WalletKey)
	if err2 != nil || wk.WalletId == 0 {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = err2.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	if wk.UserId != req.UserID {
		store.WalletKeyDelByUserIdAndChannel(req.UserID, req.Channel)
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "user id not match"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()
	var wg model.WalletGenerated
	db.Model(&model.WalletGenerated{}).Where("id = ? and status = ?", wk.WalletId, "00").First(&wg)
	if wg.ID == 0 {
		res.Code = codes.CODES_ERR_OBJ_NOT_FOUND
		res.Msg = fmt.Sprintf("unable to find wallet object with %d", wk.WalletId)
		c.JSON(http.StatusOK, res)
		return
	}

	if wg.UserID != req.UserID {
		store.WalletKeyDelByUserIdAndChannel(req.UserID, req.Channel)
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "user id not match"
		c.JSON(http.StatusOK, res)
		return
	}
	// 检查基金钱包充值的金额 并记录基金钱包充值历史
	//if !CheckMemeVaultWalletTransfer(db, req, wg) {
	//	res.Code = codes.CODE_ERR
	//	res.Msg = "冲狗基金钱包不允许转账"
	//	c.JSON(http.StatusOK, res)
	//	return
	//}
	chainConfig := config.GetRpcConfig(wg.ChainCode)

	txhash, err := chain.HandleTransfer(chainConfig, req.To, req.Token, req.Amount, &wg, &req.Config)

	if err != nil {
		log.Error("transfer error:", req, err)
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = fmt.Sprintf("unknown error %s", err.Error())
		c.JSON(http.StatusOK, res)
		return
	}

	reqdata, _ := json.Marshal(req)

	wl := &model.WalletLog{
		WalletID:  int64(wk.WalletId),
		Wallet:    wg.Wallet,
		Data:      string(reqdata),
		ChainCode: wg.ChainCode,
		Operation: "transfer",
		OpTime:    time.Now(),
		TxHash:    txhash,
	}

	err = db.Model(&model.WalletLog{}).Save(wl).Error
	if err != nil {
		log.Error("save log error ", err)
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = struct {
		Wallet string `json:"wallet"`
		Tx     string `json:"tx"`
	}{
		Wallet: wg.Wallet,
		Tx:     txhash,
	}
	c.JSON(http.StatusOK, res)
}
func AuthAdminTransfer(c *gin.Context) {
	var req TokenTransferReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.To) == 0 {
		res.Code = codes.CODE_ERR
		res.Msg = "bad request parameters"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.Channel == "" {
		res.Code = codes.CODE_ERR
		res.Msg = "bad request parameters: channel is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.UserID == "" {
		res.Code = codes.CODE_ERR
		res.Msg = "bad request parameters: userId is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.ChainCode == "" {
		res.Code = codes.CODE_ERR
		res.Msg = "bad request parameters: userId is empty"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()
	var wg model.WalletGenerated

	reqStr, _ := json.Marshal(req)
	if ok, errV := Verify2fa(req.Admin, req.TwoFACode, "AuthAdminTransfer:"+string(reqStr)); !ok {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("2fa verify failed,err:%v", errV)
		c.JSON(http.StatusOK, res)
		return
	}
	// 查询hello的基金钱包
	var fWg model.WalletGenerated
	err := db.Model(&model.WalletGenerated{}).Where("group_id = ? and chain_code=? and status = ?",
		config.GetConfig().CommissionWalletGroupID, req.ChainCode, "00").First(&fWg).Error
	if err != nil || fWg.ID < 1 {
		res.Code = codes.CODE_ERR
		mylog.Error("bad request : fromWallet is nil " + strconv.FormatInt(int64(config.GetConfig().MemeVaultFrom), 10))
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	chainConfig := config.GetRpcConfig(wg.ChainCode)

	txhash, err := chain.HandleTransfer(chainConfig, req.To, req.Token, req.Amount, &fWg, &req.Config)

	if err != nil {
		log.Error("transfer error:", req, err)
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = fmt.Sprintf("unknown error %s", err.Error())
		c.JSON(http.StatusOK, res)
		return
	}

	wl := &model.WalletLog{
		WalletID:  int64(fWg.ID),
		Wallet:    req.To,
		Data:      string(reqStr),
		ChainCode: wg.ChainCode,
		Operation: "AdminTransfer",
		OpTime:    time.Now(),
		TxHash:    txhash,
	}

	err = db.Model(&model.WalletLog{}).Save(wl).Error
	if err != nil {
		log.Error("save log error ", err)
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = struct {
		Wallet string `json:"wallet"`
		Tx     string `json:"tx"`
	}{
		Wallet: wg.Wallet,
		Tx:     txhash,
	}
	c.JSON(http.StatusOK, res)
}

func Verify2fa(admin string, code string, msg string) (bool, error) {
	var verify bool
	var err error
	defer func() {
		mylog.Infof("verify 2fa,admin:%s,code:%s,verify:%v,err:%v,msg:%s", admin, code, verify, err, msg)
	}()
	// 2fa
	if len(admin) < 1 || len(code) < 1 {
		err = fmt.Errorf("bad request parameters:TwoFA or Admin is empty")
		return false, err
	}
	var adminInDb model.AdminUser
	db := system.GetDb()
	err2 := db.Model(&model.AdminUser{}).Where("uuid = ?", admin).First(&adminInDb).Error
	if err2 != nil || adminInDb.ID < 1 || adminInDb.TwoFA == "" {
		verify = false
		err = fmt.Errorf("not found admin or admin not enable 2fa,err:%v", err2)
		return verify, err
	}
	if !common.TwoFAVerifyCode(adminInDb.TwoFA, code, 0) {
		verify = false
		err = fmt.Errorf("2fa verify failed")
		return verify, err
	}
	verify = true
	err = nil
	return true, nil
}
