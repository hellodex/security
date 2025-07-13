package controller

import (
	"errors"
	evmcommon "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

type AirdropQueryReq struct {
	WalletAddress string `json:"walletAddress"`
	Amount        uint64 `json:"amount"`
	Type          int    `json:"type"`
	SortByAmount  bool   `json:"sortByAmount"`
	Page          int    `json:"page"`
	PageSize      int    `json:"pageSize"`
}

func AirdropQuery(c *gin.Context) {
	var req AirdropQueryReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request body" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验钱包地址是否有效
	isEvm := evmcommon.IsHexAddress(req.WalletAddress)
	if isEvm {
		req.WalletAddress = strings.ToLower(req.WalletAddress)
	} else {
		//也不是solana地址 钱包地址无效
		if !isValidSolanaAddress(req.WalletAddress) {
			res.Code = codes.CODE_ERR
			res.Msg = "钱包地址无效"
			c.JSON(http.StatusOK, res)
			return
		}
	}
	db := system.GetDb()
	var airdrops []model.AirDrop
	err := db.Model(&model.AirDrop{}).Where("wallet_address = ?", req.WalletAddress).Find(&airdrops).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR
		res.Msg = "查询失败"
		c.JSON(http.StatusOK, res)
		return
	}
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = airdrops
	c.JSON(http.StatusOK, res)
	return

}
func AirdropPage(c *gin.Context) {
	var req AirdropQueryReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request body" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	// 计算分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 500 // 默认每页500条
	}
	offset := (req.Page - 1) * req.PageSize
	// 校验钱包地址是否有效
	isEvm := evmcommon.IsHexAddress(req.WalletAddress)
	if isEvm {
		req.WalletAddress = strings.ToLower(req.WalletAddress)
	} else {
		//也不是solana地址 钱包地址无效
		if !isValidSolanaAddress(req.WalletAddress) {
			req.WalletAddress = ""
		}
	}
	db := system.GetDb()

	query := db.Model(&model.AirDrop{})
	if len(req.WalletAddress) > 0 {
		query = query.Where("wallet_address =?", req.WalletAddress)
	}
	if req.Type > 0 {
		query = query.Where("type =?  ", req.Type)
	}
	var airdrops []model.AirDrop
	var totalCount int64
	if err := query.Debug().Count(&totalCount).Error; err != nil {
		totalCount = 0
		mylog.Error("MemeVaultList:queryCountError:" + err.Error())
	}
	err := query.Debug().Order("ID").Limit(req.PageSize).Offset(offset).Find(&airdrops).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR
		res.Msg = "查询失败"
		c.JSON(http.StatusOK, res)
		return
	}
	PaginatedResult := common.PaginatedResult[model.AirDrop]{
		Current: req.Page,
		Size:    req.PageSize,
		Total:   int(totalCount),
		Records: airdrops}
	res.Data = PaginatedResult
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
	return

}
func isValidSolanaAddress(address string) bool {
	// 尝试将 Base58 字符串解码为公钥
	_, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return false
	}
	// 解码成功，地址有效
	return true
}
