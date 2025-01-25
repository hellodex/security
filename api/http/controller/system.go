package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hellodex/HelloSecurity/store"
	"github.com/hellodex/HelloSecurity/swapData"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"

	"github.com/hellodex/HelloSecurity/api/common"
	chain "github.com/hellodex/HelloSecurity/chain"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet"
	"github.com/hellodex/HelloSecurity/wallet/enc"

	"github.com/gin-gonic/gin"
)

type CreateWalletRequest struct {
	UserID    string `json:"user_id"`
	ChainCode string `json:"chain_code"`
	GroupID   int    `json:"group_id"`
	Nop       string `json:"nop"`
}

type SigWalletRequest struct {
	Message  string          `json:"message"`
	Type     string          `json:"type"`
	WalletID uint64          `json:"wallet_id"`
	To       string          `json:"to"`
	Amount   *big.Int        `json:"amount"`
	Config   common.OpConfig `json:"config"`
}

type CreateBatchWalletRequest struct {
	UserID     string   `json:"user_id"`
	ChainCodes []string `json:"chain_codes"`
	GroupID    int      `json:"group_id"`
	Nop        string   `json:"nop"`
}

func CreateWallet(c *gin.Context) {
	var req CreateWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()
	var walletGroup *model.WalletGroup
	var walletGroups []model.WalletGroup
	err := db.Model(&model.WalletGroup{}).Where("user_id = ?", req.UserID).Find(&walletGroups).Error

	if req.GroupID > 0 {
		for _, v := range walletGroups {
			if v.ID == uint64(req.GroupID) {
				walletGroup = &v
			}
		}
		if err != nil {
			res.Code = codes.CODE_ERR_UNKNOWN
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		if walletGroup == nil {
			res.Code = codes.CODES_ERR_OBJ_NOT_FOUND
			res.Msg = fmt.Sprintf("can not find by group id:%d", req.GroupID)
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		if req.Nop == "Y" || len(walletGroups) == 0 {
			strmneno, err := enc.NewKeyStories()
			if err != nil {
				res.Code = codes.CODE_ERR_UNKNOWN
				res.Msg = fmt.Sprintf("can not create wallet group : %s", err.Error())
				c.JSON(http.StatusOK, res)
				return
			}
			walletGroup = &model.WalletGroup{
				UserID:         req.UserID,
				CreateTime:     time.Now(),
				EncryptMem:     strmneno,
				EncryptVersion: fmt.Sprintf("AES:%d", 1),
				Nonce:          int(enc.Porter().GetNonce()),
			}
			db.Save(walletGroup)
		} else {
			walletGroup = &walletGroups[0]
		}
	}

	var wgs []model.WalletGenerated
	db.Model(&model.WalletGenerated{}).Where("user_id = ? and group_id = ? and status = ?", req.UserID, walletGroup.ID, "00").Find(&wgs)

	var exist *model.WalletGenerated
	for _, v := range wgs {
		if v.ChainCode == req.ChainCode {
			exist = &v
		}
	}
	if exist != nil {
		res.Code = codes.CODE_ERR_EXIST_OBJ
		res.Msg = "exist wallet for this chain code"
		c.JSON(http.StatusOK, res)
		return
	}
	// var encmno string = walletGroup.EncryptMem

	wal, err := wallet.Generate(walletGroup, wallet.ChainCode(req.ChainCode))
	if err != nil {
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	channel, _ := c.Get("APP_ID")
	wg := model.WalletGenerated{
		UserID:         req.UserID,
		ChainCode:      req.ChainCode,
		Wallet:         wal.Address,
		EncryptPK:      wal.GetPk(),
		EncryptVersion: wal.Epm,
		CreateTime:     time.Now(),
		Channel:        fmt.Sprintf("%v", channel),
		CanPort:        false,
		Status:         "00",
		GroupID:        walletGroup.ID,
		Nonce:          walletGroup.Nonce,
	}

	err = db.Model(&model.WalletGenerated{}).Save(&wg).Error
	if err != nil {
		mylog.Errorf("create wallet error %v", err)
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = struct {
		WalletAddr string `json:"wallet_addr"`
		WalletId   uint64 `json:"wallet_id"`
		GroupID    uint64 `json:"group_id"`
	}{
		WalletAddr: wg.Wallet,
		WalletId:   wg.ID,
		GroupID:    walletGroup.ID,
	}

	c.JSON(http.StatusOK, res)
}

func CreateBatchWallet(c *gin.Context) {
	var req CreateBatchWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.ChainCodes) == 0 {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "chain list empty"
		c.JSON(http.StatusOK, res)
		return
	}
	validChains := wallet.CheckAllCodes(req.ChainCodes)
	if len(validChains) == 0 {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "chain list all invalid"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()
	var walletGroup *model.WalletGroup
	var walletGroups []model.WalletGroup
	err := db.Model(&model.WalletGroup{}).Where("user_id = ?", req.UserID).Find(&walletGroups).Error

	if req.GroupID > 0 {
		for _, v := range walletGroups {
			if v.ID == uint64(req.GroupID) {
				walletGroup = &v
			}
		}
		if err != nil {
			res.Code = codes.CODE_ERR_UNKNOWN
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		if walletGroup == nil {
			res.Code = codes.CODES_ERR_OBJ_NOT_FOUND
			res.Msg = fmt.Sprintf("can not find by group id:%d", req.GroupID)
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		if req.Nop == "Y" || len(walletGroups) == 0 {
			strmneno, err := enc.NewKeyStories()
			if err != nil {
				res.Code = codes.CODE_ERR_UNKNOWN
				res.Msg = fmt.Sprintf("can not create wallet group : %s", err.Error())
				c.JSON(http.StatusOK, res)
				return
			}
			walletGroup = &model.WalletGroup{
				UserID:         req.UserID,
				CreateTime:     time.Now(),
				EncryptMem:     strmneno,
				EncryptVersion: fmt.Sprintf("AES:%d", 1),
				Nonce:          int(enc.Porter().GetNonce()),
			}
			db.Save(walletGroup)
		} else {
			walletGroup = &walletGroups[0]
		}
	}

	var wgs []model.WalletGenerated
	db.Model(&model.WalletGenerated{}).
		Where("user_id = ? and group_id = ? and status = ? and chain_code IN ?", req.UserID, walletGroup.ID, "00", validChains).Find(&wgs)

	needCreates := make([]string, 0)
	for _, v := range validChains {
		exist := false
		for _, w := range wgs {
			if v == w.ChainCode {
				exist = true
				break
			}
		}
		if !exist {
			needCreates = append(needCreates, v)
		}
	}
	mylog.Info("need create: ", needCreates)
	type GetBackWallet struct {
		WalletAddr string `json:"wallet_addr"`
		WalletId   uint64 `json:"wallet_id"`
		GroupID    uint64 `json:"group_id"`
		ChainCode  string `json:"chain_code"`
	}
	if len(needCreates) == 0 {
		resultList := make([]GetBackWallet, 0)
		for _, w := range wgs {
			resultList = append(resultList, GetBackWallet{
				WalletAddr: w.Wallet,
				WalletId:   w.ID,
				GroupID:    w.GroupID,
				ChainCode:  w.ChainCode,
			})
		}
		res.Code = codes.CODE_SUCCESS
		res.Msg = "success"
		res.Data = resultList

		c.JSON(http.StatusOK, res)
		return
	}

	// newWgs := make([]model.WalletGenerated, 0)
	for _, v := range needCreates {
		wal, err := wallet.Generate(walletGroup, wallet.ChainCode(v))
		if err != nil {
			res.Code = codes.CODE_ERR_UNKNOWN
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		channel, _ := c.Get("APP_ID")
		wg := model.WalletGenerated{
			UserID:         req.UserID,
			ChainCode:      v,
			Wallet:         wal.Address,
			EncryptPK:      wal.GetPk(),
			EncryptVersion: wal.Epm,
			CreateTime:     time.Now(),
			Channel:        fmt.Sprintf("%v", channel),
			CanPort:        false,
			Status:         "00",
			GroupID:        walletGroup.ID,
			Nonce:          walletGroup.Nonce,
		}

		err = db.Model(&model.WalletGenerated{}).Save(&wg).Error
		if err != nil {
			mylog.Errorf("create wallet error %v", err)
		} else {
			wgs = append(wgs, wg)
		}
	}

	resultList := make([]GetBackWallet, 0)
	for _, w := range wgs {
		resultList = append(resultList, GetBackWallet{
			WalletAddr: w.Wallet,
			WalletId:   w.ID,
			GroupID:    w.GroupID,
			ChainCode:  w.ChainCode,
		})
	}
	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = resultList

	c.JSON(http.StatusOK, res)
}

type AuthCreateBatchWalletRequest struct {
	Account     string   `json:"account"`
	ChainCodes  []string `json:"chainCodes"`
	Captcha     string   `json:"captcha"`
	Channel     string   `json:"channel"`
	UserID      string   `json:"userId"`
	Type        string   `json:"type"`
	TwoFA       string   `json:"twofa"`
	LeastGroups int      `json:"leastGroups"`
	ExpireTime  int64    `json:"expireTime"`
}

func AuthCreateBatchWallet(c *gin.Context) {
	var req AuthCreateBatchWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	//返回的钱包列表
	resultList := make([]*AuthGetBackWallet, 0)
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.ChainCodes) == 0 {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "chain list empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if len(req.UserID) == 0 || req.UserID == "" {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "user id is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	validChains := wallet.CheckAllCodes(req.ChainCodes)
	if len(validChains) == 0 {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "chain list all invalid"
		c.JSON(http.StatusOK, res)
		return
	}
	//如果没有验证码，则校验是否有钱包,有则表示用户不是新用户,不是注册后调用 返回空数组
	db := system.GetDb()
	if req.Captcha == "" {
		var count int64
		db.Model(&model.WalletGenerated{}).Where("user_id = ? and status = ? ", req.UserID, "00").Count(&count)
		// 数据库
		if count > 0 {
			res.Code = codes.CODE_ERR_AUTH_FAIL
			res.Msg = "get error,no auth"
			c.JSON(http.StatusOK, res)
			return
		}
	} else {
		// 验证码校验 登陆调用
		isValid := system.VerifyCode(req.Account+req.Type, req.Captcha)
		if !isValid {
			res.Code = codes.CODE_ERR_VERIFY_FAIL
			res.Msg = "captcha error"
			c.JSON(http.StatusOK, res)
			return
		}
	}

	//获取用户所有的钱包组
	var walletGroups []model.WalletGroup
	err := db.Model(&model.WalletGroup{}).Where("user_id = ?", req.UserID).Find(&walletGroups).Error

	if err != nil {
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	groupSize := len(walletGroups)
	//没有达到最少组数，创建新的组
	if groupSize < req.LeastGroups {
		needCreateGropesSize := req.LeastGroups - groupSize
		for _ = range needCreateGropesSize {
			strmneno, err := enc.NewKeyStories()
			if err != nil {
				res.Code = codes.CODE_ERR_UNKNOWN
				res.Msg = fmt.Sprintf("can not create wallet group : %s", err.Error())
				c.JSON(http.StatusOK, res)
				return
			}
			te := &model.WalletGroup{
				UserID:         req.UserID,
				CreateTime:     time.Now(),
				EncryptMem:     strmneno,
				EncryptVersion: fmt.Sprintf("AES:%d", 1),
				Nonce:          int(enc.Porter().GetNonce()),
			}
			db.Save(te)
			walletGroups = append(walletGroups, *te)
		}
	}
	//获取每一组的钱包地址
	for _, g := range walletGroups {
		var wgs []model.WalletGenerated
		db.Model(&model.WalletGenerated{}).
			Where("user_id = ? and group_id = ? and status = ? and chain_code IN ?", req.UserID, g.ID, "00", validChains).Find(&wgs)

		//校验每一个 chaincode 对应的钱包是否已经存在
		needCreates := make([]string, 0)
		for _, v := range validChains {
			exist := false
			for _, w := range wgs {
				if v == w.ChainCode {
					exist = true
					break
				}
			}
			if !exist {
				needCreates = append(needCreates, v)
			}
		}
		mylog.Info("need create: ", needCreates)

		if len(needCreates) == 0 {
			for _, w := range wgs {
				resultList = append(resultList, &AuthGetBackWallet{
					WalletAddr: w.Wallet,
					WalletId:   w.ID,
					GroupID:    w.GroupID,
					ChainCode:  w.ChainCode,
				})
			}
			continue
		}

		// 需要创建的chaincode钱包
		for _, v := range needCreates {
			wal, err := wallet.Generate(&g, wallet.ChainCode(v))
			if err != nil {
				res.Code = codes.CODE_ERR_UNKNOWN
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			channel, _ := c.Get("APP_ID")
			wg := model.WalletGenerated{
				UserID:         req.UserID,
				ChainCode:      v,
				Wallet:         wal.Address,
				EncryptPK:      wal.GetPk(),
				EncryptVersion: wal.Epm,
				CreateTime:     time.Now(),
				Channel:        fmt.Sprintf("%v", channel),
				CanPort:        false,
				Status:         "00",
				GroupID:        g.ID,
				Nonce:          g.Nonce,
			}

			err = db.Model(&model.WalletGenerated{}).Save(&wg).Error
			if err != nil {
				mylog.Errorf("create wallet error %v", err)
			} else {
				wgs = append(wgs, wg)
				resultList = append(resultList, &AuthGetBackWallet{
					WalletAddr: wg.Wallet,
					WalletId:   wg.ID,
					GroupID:    wg.GroupID,
					ChainCode:  wg.ChainCode,
				})
			}
		}

	}

	walletKeys := make([]model.WalletKeys, 0)
	for _, r := range resultList {
		walletKey := common.MyIDStr()
		time.Sleep(time.Millisecond)
		walletKeys = append(walletKeys, model.WalletKeys{
			WalletKey:  walletKey,
			WalletId:   r.WalletId,
			Channel:    req.Channel,
			ExpireTime: req.ExpireTime,
			UserId:     req.UserID,
		})
		r.WalletKey = walletKey
		r.ExpireTime = req.ExpireTime
	}
	mylog.Info("重新登陆删除过期的walletKeys: ", req.UserID, req.Channel)
	store.WalletKeyDelByUserIdAndChannel(req.UserID, req.Channel)
	err = store.WalletKeySaveBatch(walletKeys)
	if err != nil {
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = "walletkey save err"
		c.JSON(http.StatusOK, res)
		return
	}
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = resultList

	c.JSON(http.StatusOK, res)
	return
}

//	func AuthCreateBatchTgWallet(c *gin.Context) {
//		var req AuthCreateBatchWalletRequest
//		res := common.Response{}
//		res.Timestamp = time.Now().Unix()
//		//返回的钱包列表
//		resultList := make([]*AuthGetBackWallet, 0)
//		if err := c.ShouldBindJSON(&req); err != nil {
//			res.Code = codes.CODE_ERR_REQFORMAT
//			res.Msg = "Invalid request"
//			c.JSON(http.StatusOK, res)
//			return
//		}
//
//		if len(req.ChainCodes) == 0 {
//			res.Code = codes.CODE_ERR_REQFORMAT
//			res.Msg = "chain list empty"
//			c.JSON(http.StatusOK, res)
//			return
//		}
//		if len(req.UserID) == 0 || req.UserID == "" {
//			res.Code = codes.CODE_ERR_AUTH_FAIL
//			res.Msg = "user id is empty"
//			c.JSON(http.StatusOK, res)
//			return
//		}
//		validChains := wallet.CheckAllCodes(req.ChainCodes)
//		if len(validChains) == 0 {
//			res.Code = codes.CODE_ERR_BAT_PARAMS
//			res.Msg = "chain list all invalid"
//			c.JSON(http.StatusOK, res)
//			return
//		}
//		//如果没有验证码，则校验是否有钱包,有则表示用户不是新用户,不是注册后调用 返回空数组
//		db := system.GetDb()
//		// todo  临时使用 缺失安全校验
//		//if req.Captcha == "" {
//		//	var count int64
//		//	db.Model(&model.WalletGenerated{}).Where("user_id = ? and status = ? ", req.UserID, "00").Count(&count)
//		//	// 数据库
//		//	if count > 0 {
//		//		res.Code = codes.CODE_ERR_AUTH_FAIL
//		//		res.Msg = "get error,no auth"
//		//		c.JSON(http.StatusOK, res)
//		//		return
//		//	}
//		//}
//		//else {
//		//	// 验证码校验 登陆调用
//		//	isValid := system.VerifyCode(req.Account+req.Type, req.Captcha)
//		//	if !isValid {
//		//		res.Code = codes.CODE_ERR_VERIFY_FAIL
//		//		res.Msg = "captcha error"
//		//		c.JSON(http.StatusOK, res)
//		//		return
//		//	}
//		//}
//
//		//获取用户所有的钱包组
//		var walletGroups []model.WalletGroup
//		err := db.Model(&model.WalletGroup{}).Where("user_id = ?", req.UserID).Find(&walletGroups).Error
//
//		if err != nil {
//			res.Code = codes.CODE_ERR_UNKNOWN
//			res.Msg = err.Error()
//			c.JSON(http.StatusOK, res)
//			return
//		}
//
//		groupSize := len(walletGroups)
//		//没有达到最少组数，创建新的组
//		if groupSize < req.LeastGroups {
//			needCreateGropesSize := req.LeastGroups - groupSize
//			for _ = range needCreateGropesSize {
//				strmneno, err := enc.NewKeyStories()
//				if err != nil {
//					res.Code = codes.CODE_ERR_UNKNOWN
//					res.Msg = fmt.Sprintf("can not create wallet group : %s", err.Error())
//					c.JSON(http.StatusOK, res)
//					return
//				}
//				te := &model.WalletGroup{
//					UserID:         req.UserID,
//					CreateTime:     time.Now(),
//					EncryptMem:     strmneno,
//					EncryptVersion: fmt.Sprintf("AES:%d", 1),
//					Nonce:          int(enc.Porter().GetNonce()),
//				}
//				db.Save(te)
//				walletGroups = append(walletGroups, *te)
//			}
//		}
//		//获取每一组的钱包地址
//		for _, g := range walletGroups {
//			var wgs []model.WalletGenerated
//			db.Model(&model.WalletGenerated{}).
//				Where("user_id = ? and group_id = ? and status = ? and chain_code IN ?", req.UserID, g.ID, "00", validChains).Find(&wgs)
//
//			//校验每一个 chaincode 对应的钱包是否已经存在
//			needCreates := make([]string, 0)
//			for _, v := range validChains {
//				exist := false
//				for _, w := range wgs {
//					if v == w.ChainCode {
//						exist = true
//						break
//					}
//				}
//				if !exist {
//					needCreates = append(needCreates, v)
//				}
//			}
//			mylog.Info("need create: ", needCreates)
//
//			if len(needCreates) == 0 {
//				for _, w := range wgs {
//					resultList = append(resultList, &AuthGetBackWallet{
//						WalletAddr: w.Wallet,
//						WalletId:   w.ID,
//						GroupID:    w.GroupID,
//						ChainCode:  w.ChainCode,
//					})
//				}
//				continue
//			}
//
//			// 需要创建的chaincode钱包
//			for _, v := range needCreates {
//				wal, err := wallet.Generate(&g, wallet.ChainCode(v))
//				if err != nil {
//					res.Code = codes.CODE_ERR_UNKNOWN
//					res.Msg = err.Error()
//					c.JSON(http.StatusOK, res)
//					return
//				}
//
//				channel, _ := c.Get("APP_ID")
//				wg := model.WalletGenerated{
//					UserID:         req.UserID,
//					ChainCode:      v,
//					Wallet:         wal.Address,
//					EncryptPK:      wal.GetPk(),
//					EncryptVersion: wal.Epm,
//					CreateTime:     time.Now(),
//					Channel:        fmt.Sprintf("%v", channel),
//					CanPort:        false,
//					Status:         "00",
//					GroupID:        g.ID,
//					Nonce:          g.Nonce,
//				}
//
//				err = db.Model(&model.WalletGenerated{}).Save(&wg).Error
//				if err != nil {
//					mylog.Errorf("create wallet error %v", err)
//				} else {
//					wgs = append(wgs, wg)
//					resultList = append(resultList, &AuthGetBackWallet{
//						WalletAddr: wg.Wallet,
//						WalletId:   wg.ID,
//						GroupID:    wg.GroupID,
//						ChainCode:  wg.ChainCode,
//					})
//				}
//			}
//
//		}
//
//		walletKeys := make([]model.WalletKeys, 0)
//		for _, r := range resultList {
//			walletKey := common.MyIDStr()
//			time.Sleep(time.Millisecond)
//			walletKeys = append(walletKeys, model.WalletKeys{
//				WalletKey:  walletKey,
//				WalletId:   r.WalletId,
//				Channel:    req.Channel,
//				ExpireTime: req.ExpireTime,
//				UserId:     req.UserID,
//			})
//			r.WalletKey = walletKey
//			r.ExpireTime = req.ExpireTime
//		}
//		mylog.Info("重新登陆删除过期的walletKeys: ", req.UserID, req.Channel)
//		store.WalletKeyDelByUserIdAndChannel(req.UserID, req.Channel)
//		err = store.WalletKeySaveBatch(walletKeys)
//		if err != nil {
//			res.Code = codes.CODE_ERR_UNKNOWN
//			res.Msg = "walletkey save err"
//			c.JSON(http.StatusOK, res)
//			return
//		}
//		res.Code = codes.CODE_SUCCESS_200
//		res.Msg = "success"
//		res.Data = resultList
//
//		c.JSON(http.StatusOK, res)
//		return
//	}
func AuthCreateBatchTgWallet1(c *gin.Context) {
	var req AuthCreateBatchWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	//返回的钱包列表
	resultList := make([]*AuthGetBackWallet, 0)
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}
	// 已经校验过channel
	// 校验channel 必须是tg
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
	if len(req.ChainCodes) == 0 {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "chain list empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if len(req.UserID) == 0 || req.UserID == "" {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "user id is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	validChains := wallet.CheckAllCodes(req.ChainCodes)
	if len(validChains) == 0 {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "chain list all invalid"
		c.JSON(http.StatusOK, res)
		return
	}
	//如果没有验证码，则校验是否有钱包,有则表示用户不是新用户,不是注册后调用 返回空数组
	db := system.GetDb()
	// todo  临时使用 缺失安全校验
	//if req.Captcha == "" {
	//	var count int64
	//	db.Model(&model.WalletGenerated{}).Where("user_id = ? and status = ? ", req.UserID, "00").Count(&count)
	//	// 数据库
	//	if count > 0 {
	//		res.Code = codes.CODE_ERR_AUTH_FAIL
	//		res.Msg = "get error,no auth"
	//		c.JSON(http.StatusOK, res)
	//		return
	//	}
	//}
	//else {
	//	// 验证码校验 登陆调用
	//	isValid := system.VerifyCode(req.Account+req.Type, req.Captcha)
	//	if !isValid {
	//		res.Code = codes.CODE_ERR_VERIFY_FAIL
	//		res.Msg = "captcha error"
	//		c.JSON(http.StatusOK, res)
	//		return
	//	}
	//}

	//获取用户所有的钱包组
	var walletGroups []model.WalletGroup
	err := db.Model(&model.WalletGroup{}).Where("user_id = ?", req.UserID).Find(&walletGroups).Error

	if err != nil {
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	groupSize := len(walletGroups)
	//没有达到最少组数，创建新的组
	if groupSize < req.LeastGroups {
		needCreateGropesSize := req.LeastGroups - groupSize
		for _ = range needCreateGropesSize {
			strmneno, err := enc.NewKeyStories()
			if err != nil {
				res.Code = codes.CODE_ERR_UNKNOWN
				res.Msg = fmt.Sprintf("can not create wallet group : %s", err.Error())
				c.JSON(http.StatusOK, res)
				return
			}
			te := &model.WalletGroup{
				UserID:         req.UserID,
				CreateTime:     time.Now(),
				EncryptMem:     strmneno,
				EncryptVersion: fmt.Sprintf("AES:%d", 1),
				Nonce:          int(enc.Porter().GetNonce()),
			}
			db.Save(te)
			walletGroups = append(walletGroups, *te)
		}
	}
	//获取每一组的钱包地址
	for _, g := range walletGroups {
		var wgs []model.WalletGenerated
		db.Model(&model.WalletGenerated{}).
			Where("user_id = ? and group_id = ? and status = ? and chain_code IN ?", req.UserID, g.ID, "00", validChains).Find(&wgs)

		//校验每一个 chaincode 对应的钱包是否已经存在
		needCreates := make([]string, 0)
		for _, v := range validChains {
			exist := false
			for _, w := range wgs {
				if v == w.ChainCode {
					exist = true
					break
				}
			}
			if !exist {
				needCreates = append(needCreates, v)
			}
		}
		mylog.Info("need create: ", needCreates)

		if len(needCreates) == 0 {
			for _, w := range wgs {
				resultList = append(resultList, &AuthGetBackWallet{
					WalletAddr: w.Wallet,
					WalletId:   w.ID,
					GroupID:    w.GroupID,
					ChainCode:  w.ChainCode,
				})
			}
			continue
		}

		// 需要创建的chaincode钱包
		for _, v := range needCreates {
			wal, err := wallet.Generate(&g, wallet.ChainCode(v))
			if err != nil {
				res.Code = codes.CODE_ERR_UNKNOWN
				res.Msg = err.Error()
				c.JSON(http.StatusOK, res)
				return
			}

			channel, _ := c.Get("APP_ID")
			wg := model.WalletGenerated{
				UserID:         req.UserID,
				ChainCode:      v,
				Wallet:         wal.Address,
				EncryptPK:      wal.GetPk(),
				EncryptVersion: wal.Epm,
				CreateTime:     time.Now(),
				Channel:        fmt.Sprintf("%v", channel),
				CanPort:        false,
				Status:         "00",
				GroupID:        g.ID,
				Nonce:          g.Nonce,
			}

			err = db.Model(&model.WalletGenerated{}).Save(&wg).Error
			if err != nil {
				mylog.Errorf("create wallet error %v", err)
			} else {
				wgs = append(wgs, wg)
				resultList = append(resultList, &AuthGetBackWallet{
					WalletAddr: wg.Wallet,
					WalletId:   wg.ID,
					GroupID:    wg.GroupID,
					ChainCode:  wg.ChainCode,
				})
			}
		}

	}

	walletKeys := make([]model.WalletKeys, 0)
	for _, r := range resultList {
		walletKey := common.MyIDStr()
		time.Sleep(time.Millisecond)
		walletKeys = append(walletKeys, model.WalletKeys{
			WalletKey:  walletKey,
			WalletId:   r.WalletId,
			Channel:    req.Channel,
			ExpireTime: req.ExpireTime,
			UserId:     req.UserID,
		})
		r.WalletKey = walletKey
		r.ExpireTime = req.ExpireTime
	}
	mylog.Info("重新登陆删除过期的walletKeys: ", req.UserID, req.Channel)
	store.WalletKeyDelByUserIdAndChannel(req.UserID, req.Channel)
	err = store.WalletKeySaveBatch(walletKeys)
	if err != nil {
		res.Code = codes.CODE_ERR_UNKNOWN
		res.Msg = "walletkey save err"
		c.JSON(http.StatusOK, res)
		return
	}
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = resultList

	c.JSON(http.StatusOK, res)
	return
}
func Sig(c *gin.Context) {
	var req SigWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.Message) == 0 || (req.Type != "transaction" && req.Type != "sign") {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters"
		c.JSON(http.StatusOK, res)
		return
	}

	db := system.GetDb()
	var wg model.WalletGenerated
	db.Model(&model.WalletGenerated{}).Where("id = ? and status = ?", req.WalletID, "00").First(&wg)
	if wg.ID == 0 {
		res.Code = codes.CODES_ERR_OBJ_NOT_FOUND
		res.Msg = fmt.Sprintf("unable to find wallet object with %d", req.WalletID)
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Info("accept req: ", req.Message)

	chainConfig := config.GetRpcConfig(wg.ChainCode)
	txhash, sig, err := chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
	sigStr := ""

	if len(sig) > 0 {
		sigStr = base64.StdEncoding.EncodeToString(sig)
	}

	wl := &model.WalletLog{
		WalletID:  int64(req.WalletID),
		Wallet:    wg.Wallet,
		Data:      req.Message,
		Sig:       sigStr,
		ChainCode: wg.ChainCode,
		Operation: req.Type,
		OpTime:    time.Now(),
		TxHash:    txhash,
	}

	if err != nil {
		wl.Err = err.Error()
	}
	err1 := db.Model(&model.WalletLog{}).Save(wl).Error
	if err1 != nil {
		mylog.Error("save log error ", err)
	}

	if err != nil {
		res.Code = codes.CODES_ERR_TX
		res.Msg = err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = struct {
		Signature string `json:"signature"`
		Wallet    string `json:"wallet"`
		Tx        string `json:"tx"`
	}{
		Signature: sigStr,
		Wallet:    wg.Wallet,
		Tx:        txhash,
	}
	c.JSON(http.StatusOK, res)
}

func AuthSig(c *gin.Context) {
	var req common.AuthSigWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		body, _ := c.GetRawData()
		mylog.Info("AuthSig req: ", string(body))
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验参数TYPE 必须为 transaction 或 sign
	if req.Type != "transaction" && req.Type != "sign" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters: type error"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验channel是否为空
	if len(req.Channel) == 0 || req.Channel == "" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters: channel is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	walletId := uint64(0)
	//limitKey 校验
	limitFlag := req.LimitOrderParams.LimitOrderKey != ""
	if limitFlag {
		lk, errL := store.LimitKeyCheckAndGet(req.LimitOrderParams.LimitOrderKey)
		if errL != nil || lk.LimitKey == "" {
			res.Code = codes.CODE_ERR_BAT_PARAMS
			res.Msg = "bad request  : limit order key error"
			c.JSON(http.StatusOK, res)
			return
		}
		walletId = lk.WalletID

	} else {
		if len(req.Message) == 0 {
			res.Code = codes.CODE_ERR_BAT_PARAMS
			res.Msg = "bad request parameters: message is empty"
			c.JSON(http.StatusOK, res)
			return
		}
		wk, err2 := store.WalletKeyCheckAndGet(req.WalletKey)
		if err2 != nil || wk == nil {
			res.Code = codes.CODE_ERR_AUTH_FAIL
			res.Msg = err2.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		// 校验钱包key是否正确 并且 钱包key对应的用户id和请求的用户id一致
		if wk.WalletKey == "" || wk.UserId != req.UserId {
			mylog.Info("walletkey check fail")
			store.WalletKeyDelByUserIdAndChannel(req.UserId, req.Channel)
		}
		walletId = wk.WalletId
	}

	db := system.GetDb()
	var wg model.WalletGenerated
	db.Model(&model.WalletGenerated{}).Where("id = ? and status = ?", walletId, "00").First(&wg)
	if wg.ID == 0 {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = fmt.Sprintf("unable to find wallet object with %d", walletId)
		c.JSON(http.StatusOK, res)
		return
	}
	if wg.UserID != req.UserId {
		// 校验钱包key是否正确 并且 钱包key对应的用户id和请求的用户id一致
		if !limitFlag {
			mylog.Info("WalletGenerated  check fail")
			store.WalletKeyDelByUserIdAndChannel(req.UserId, req.Channel)
		}

		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "user id not match"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Info("accept req: ", req.Message)

	chainConfig := config.GetRpcConfig(wg.ChainCode)
	if !limitFlag {
		txhash, sig, err := chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
		sigStr := ""

		if len(sig) > 0 {
			sigStr = base64.StdEncoding.EncodeToString(sig)
		}

		wl := &model.WalletLog{
			WalletID:  int64(walletId),
			Wallet:    wg.Wallet,
			Data:      req.Message,
			Sig:       sigStr,
			ChainCode: wg.ChainCode,
			Operation: req.Type,
			OpTime:    time.Now(),
			TxHash:    txhash,
		}

		if err != nil {
			wl.Err = err.Error()
		}
		err1 := db.Model(&model.WalletLog{}).Save(wl).Error
		if err1 != nil {
			mylog.Error("save log error ", err)
		}

		if err != nil {
			res.Code = codes.CODES_ERR_TX
			res.Msg = err.Error()
			c.JSON(http.StatusOK, res)
			return
		}

		res.Code = codes.CODE_SUCCESS
		res.Msg = "success"
		res.Data = common.SignRes{
			Signature: sigStr,
			Wallet:    wg.Wallet,
			Tx:        txhash,
		}
		c.JSON(http.StatusOK, res)
	} else {
		okxReq := &req.LimitOrderParams
		swapDataMap := make(map[string]interface{})
		msg := req.Message
		to := req.To
		amount := req.Amount
		for i := 1; i <= 6; i++ {
			lk1, errL1 := store.LimitKeyCheckAndGet(req.LimitOrderParams.LimitOrderKey)
			if errL1 != nil || lk1.LimitKey == "" {
				swapDataMap["callDataErrNoLimitKey"+strconv.Itoa(i)] = errL1.Error()
				res.Code = codes.CODE_ERR_BAT_PARAMS
				res.Msg = "bad request  : limit order key error"
				res.Data = common.SignRes{
					Signature: "",
					Wallet:    "",
					Tx:        "",
					CallData:  swapDataMap,
				}
				c.JSON(http.StatusOK, res)
				return
			}

			//翻倍滑点
			if i > 1 {
				fromString, err2 := decimal.NewFromString(okxReq.Slippage)
				if err2 != nil {
					fromString = decimal.NewFromFloat(0.05).Mul(decimal.NewFromInt(int64(i)))
				}
				okxReq.Slippage = fromString.Add(decimal.NewFromFloat(0.05)).String()

			}

			swapDataMap, okxResponse, errL := swapData.GetSwapData(i, swapDataMap, okxReq)
			if errL != nil || okxResponse.Code != "0" || len(okxResponse.Data) == 0 {
				res.Code = codes.CODE_ERR_102
				res.Msg = "bad request okx res"
				res.Data = common.SignRes{
					Signature: "",
					Wallet:    wg.Wallet,
					Tx:        "",
					CallData:  swapDataMap,
				}
				c.JSON(http.StatusOK, res)
				return
			}
			OKXData := okxResponse.Data[0]
			msg1 := OKXData.Tx.Data
			msg = msg1
			if wg.ChainCode == "SOLANA" {
				// Base58 解码
				decoded, _ := base58.Decode(msg1)
				// Base64 编码
				msg = base64.StdEncoding.EncodeToString(decoded)
			}

			to = OKXData.Tx.To
			amount1 := OKXData.Tx.Value
			if amount1 == "" {
				amount1 = "0"
			}
			amount = new(big.Int)
			amount.SetString(amount1, 10)
			txhash, sig, err := chain.HandleMessage(chainConfig, msg, to, req.Type, amount, &req.Config, &wg)
			sigStr := ""
			if err != nil && (strings.Contains(err.Error(), "error: 0x1771") ||
				strings.Contains(err.Error(), "error: 6001") ||
				strings.Contains(err.Error(), "Error Message: slippage") ||
				strings.Contains(err.Error(), "Custom:6001") ||
				strings.Contains(err.Error(), "status:failed") ||
				strings.Contains(err.Error(), "status:unpub")) {
				swapDataMap["callDataErr"+strconv.Itoa(i)] = err.Error()
				swapDataMap["callDataErrTxHash"+strconv.Itoa(i)] = txhash
				continue
			}
			if len(sig) > 0 {
				sigStr = base64.StdEncoding.EncodeToString(sig)
			}

			wl := &model.WalletLog{
				WalletID:  int64(walletId),
				Wallet:    wg.Wallet,
				Data:      req.Message,
				Sig:       sigStr,
				ChainCode: wg.ChainCode,
				Operation: req.Type,
				OpTime:    time.Now(),
				TxHash:    txhash,
			}

			if err != nil {
				wl.Err = err.Error()
				swapDataMap["callDataErr"+strconv.Itoa(i)] = err.Error()
			}
			err1 := db.Model(&model.WalletLog{}).Save(wl).Error
			if err1 != nil {
				mylog.Error("save log error ", err)
			}

			if err != nil {
				res.Code = codes.CODES_ERR_TX
				res.Msg = err.Error()
				res.Data = common.SignRes{
					Signature: msg,
					Wallet:    wg.Wallet,
					Tx:        "",
					CallData:  swapDataMap,
				}
				c.JSON(http.StatusOK, res)
				return
			}
			mylog.Info("swap success,del limitkey ", req.LimitOrderParams.LimitOrderKey)
			store.LimitKeyDelByKey(req.LimitOrderParams.LimitOrderKey)
			res.Code = codes.CODE_SUCCESS
			res.Msg = "success"
			res.Data = common.SignRes{
				Signature: msg,
				Wallet:    wg.Wallet,
				Tx:        txhash,
				CallData:  swapDataMap,
			}
			c.JSON(http.StatusOK, res)
			return
		}

	}

}
func AuthCloseAllAta(c *gin.Context) {
	var req common.AuthSigWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		body, _ := c.GetRawData()
		mylog.Info("AuthSig req: ", string(body))
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验参数TYPE 必须为 transaction 或 sign
	if req.Type != "transaction" && req.Type != "sign" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters: type error"
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验channel是否为空
	if len(req.Channel) == 0 || req.Channel == "" {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters: channel is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	walletId := uint64(0)
	if len(req.Message) == 0 {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "bad request parameters: message is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	wk, err2 := store.WalletKeyCheckAndGet(req.WalletKey)
	if err2 != nil || wk == nil {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = err2.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验钱包key是否正确 并且 钱包key对应的用户id和请求的用户id一致
	if wk.WalletKey == "" || wk.UserId != req.UserId {
		mylog.Info("walletkey check fail")
		store.WalletKeyDelByUserIdAndChannel(req.UserId, req.Channel)
	}
	walletId = wk.WalletId

	db := system.GetDb()
	var wg model.WalletGenerated
	db.Model(&model.WalletGenerated{}).Where("id = ? and status = ?", walletId, "00").First(&wg)
	if wg.ID == 0 {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = fmt.Sprintf("unable to find wallet object with %d", walletId)
		c.JSON(http.StatusOK, res)
		return
	}
	if wg.UserID != req.UserId {
		// 校验钱包key是否正确 并且 钱包key对应的用户id和请求的用户id一致

		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "user id not match"
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Info("accept req: ", req.Message)

	chainConfig := config.GetRpcConfig(wg.ChainCode)

	feePayer := solana.MustPublicKeyFromBase58(wg.Wallet)
	instructions, err2 := getCloseAtaAccountsInstructionsTx(chainConfig, feePayer)
	if err2 != nil {
		res.Code = codes.CODE_ERR_102
		res.Msg = "无法获取账户信息"
		c.JSON(http.StatusOK, res)
		return
	}
	if len(instructions) <= 0 {
		res.Code = codes.CODE_SUCCESS_200
		res.Msg = "无待关闭账户"
		c.JSON(http.StatusOK, res)
		return
	}
	txHashs := make([]string, 0)
	// 批量处理
	// 20个一批
	count := len(instructions)
	for i, ins := range batchSlice(instructions, 20) {
		tx, err := solana.NewTransaction(
			ins,
			solana.Hash{},
			solana.TransactionPayer(feePayer),
		)
		toBase64 := tx.MustToBase64()
		req.Message = toBase64
		txhash, sig, err := chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)

		sigStr := ""
		txHashs = append(txHashs, txhash)
		if len(sig) > 0 {
			sigStr = base64.StdEncoding.EncodeToString(sig)
		}
		mylog.Infof("批量关闭Ata 第%d批,%d个账户,tx:%s,sig:%s,err:%+v", i+1, len(ins), txhash, sigStr, err)
		wl := &model.WalletLog{
			WalletID:  int64(walletId),
			Wallet:    wg.Wallet,
			Data:      req.Message,
			Sig:       sigStr,
			ChainCode: wg.ChainCode,
			Operation: req.Type,
			OpTime:    time.Now(),
			TxHash:    txhash,
		}

		if err != nil {
			wl.Err = err.Error()
		}
		err1 := db.Model(&model.WalletLog{}).Save(wl).Error
		if err1 != nil {
			mylog.Error("save log error ", err)
		}
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = fmt.Sprintf("success:%d,txHashs:%s", count, strings.Join(txHashs, ","))
	res.Data = common.SignRes{
		Signature: "",
		Wallet:    wg.Wallet,
		Tx:        "",
	}
	c.JSON(http.StatusOK, res)
}

func getCloseAtaAccountsInstructionsTx(t *config.ChainConfig, payer solana.PublicKey) ([]solana.Instruction, error) {
	ctx := context.Background()
	rpcUrlDefault := t.GetRpc()[0]
	c := rpc.New(rpcUrlDefault)
	// 获取账户所有代币账户
	accounts, err := c.GetTokenAccountsByOwner(
		ctx,
		payer,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Commitment: rpc.CommitmentFinalized,
		},
	)

	// todo solana.Token2022ProgramID 未处理
	if err != nil {
		mylog.Error("get token accounts error ", err)
		return nil, err
	}
	var instructions []solana.Instruction
	// 处理每个代币账户
	for _, account := range accounts.Value {
		var tokAcc token.Account

		data := account.Account.Data.GetBinary()
		dec := bin.NewBinDecoder(data)
		err1 := dec.Decode(&tokAcc)
		if err1 != nil {
			fmt.Println(err1)
		}

		if tokAcc.Amount > 0 {
			continue
		}
		mylog.Infof("tokenAccount: payer:%s,account:%s,mint:%s \n", payer.String(), account.Pubkey.String(), tokAcc.Mint.String())
		closeIx := token.NewCloseAccountInstruction(
			account.Pubkey,
			payer,
			payer,
			[]solana.PublicKey{},
		).Build()
		instructions = append(instructions, closeIx)
	}
	return instructions, nil
}

func List(c *gin.Context) {
	var req CreateWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request"
		c.JSON(http.StatusBadRequest, res)
		return
	}

	db := system.GetDb()
	var wg []model.WalletGenerated
	db.Model(&model.WalletGenerated{}).Where("user_id = ? and status = ?", req.UserID, "00").Find(&wg)

	type WalletList struct {
		ID         uint64    `json:"id"`
		Wallet     string    `json:"wallet"`
		Chain      string    `json:"chain"`
		CreateTime time.Time `json:"create_time"`
		Export     bool      `json:"export"`
		GroupID    uint64    `json:"group_id"`
	}

	retData := make([]WalletList, 0)
	for _, v := range wg {
		retData = append(retData, WalletList{
			ID:         uint64(v.ID),
			Wallet:     v.Wallet,
			Chain:      v.ChainCode,
			CreateTime: v.CreateTime,
			Export:     v.CanPort,
			GroupID:    v.GroupID,
		})
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = retData
	c.JSON(http.StatusOK, res)
}

func DelWalletKeys(c *gin.Context) {
	var req DelWalletKeysRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request"
		c.JSON(http.StatusOK, res)
		return
	}
	err := store.WalletKeyDelByUserIdAndChannel(req.UserId, req.Channel)
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

type DelWalletKeysRequest struct {
	UserId  string `json:"userId"`
	Channel string `json:"Channel"`
}
type AuthGetBackWallet struct {
	WalletAddr string `json:"walletAddr"`
	WalletId   uint64 `json:"walletId"`
	GroupID    uint64 `json:"groupId"`
	ChainCode  string `json:"chainCode"`
	WalletKey  string `json:"walletKey"`
	ExpireTime int64  `json:"expireTime"`
}

// 定义一个通用的分批函数
func batchSlice[T any](s []T, batchSize int) [][]T {
	if batchSize <= 0 {
		return nil // 如果批大小不合法，返回 nil
	}

	var batches [][]T
	for i := 0; i < len(s); i += batchSize {
		end := i + batchSize
		if end > len(s) {
			end = len(s) // 确保不超出切片范围
		}
		batches = append(batches, s[i:end]) // 添加子切片到结果切片
	}
	return batches
}
