package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/chain"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type IdoVerifyReq struct {
	Tx        string `json:"tx"`
	ChainCode string `json:"chainCode"`
	Rpc       string `json:"rpc"`
	UUID      string `json:"uuid"`
}

func IdoVerify(c *gin.Context) {
	var req IdoVerifyReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Invalid request:Params error:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	tx := req.Tx
	// 限制请求频率
	if common.RateLimiterMoreThan("IdoVerify"+tx, 1, 1*time.Second) {
		res.Code = codes.CODE_ERR
		res.Msg = "操作太频繁,请稍后再试"
		c.JSON(http.StatusOK, res)
		return
	}
	db := system.GetDb()
	var idoLogs []model.IdoLog
	// 查询是否已经验证过
	err := db.Model(&model.IdoLog{}).Where("tx = ?", tx).Find(&idoLogs).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "query ido log error:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	// 已经验证过
	if len(idoLogs) > 0 {
		res.Code = codes.CODE_SUCCESS_200
		res.Msg = "success: already verify"
		res.Data = idoLogs[0]
		c.JSON(http.StatusOK, res)
		return
	}
	// 解析交易
	parser, err := chain.TxParser(req.Tx, req.ChainCode, req.Rpc)
	if err != nil {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "parse tx error:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	// 金额是否为0
	if parser.Amount.Sign() < 1 {
		res.Code = codes.CODE_ERR_INVALID
		res.Msg = "Amount is zero:"
		c.JSON(http.StatusOK, res)
		return
	}
	//接收方地址是否是ido地址 接收的token是报价币种
	if _, ok := common.IdoAddrMap[parser.To]; ok && common.QuoteContains(req.ChainCode, parser.Contract) {
		amountI := parser.Amount
		amount := decimal.NewFromBigInt(amountI, 0)
		amount = amount.Div(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(parser.Decimals)))).Round(8)
		log := &model.IdoLog{
			Tx:         req.Tx,
			ChainCode:  req.ChainCode,
			Wallet:     parser.From,
			IdoWallet:  parser.To,
			Token:      parser.Symbol,
			Price:      parser.Price,
			Amount:     amount,
			Block:      parser.Block,
			BlockTime:  parser.BlockTime,
			CreateTime: time.Now(),
		}
		err := db.Create(log).Error
		if err != nil {
			res.Code = codes.CODE_ERR_INVALID
			res.Msg = "create ido log error:" + err.Error()
			res.Data = log
			c.JSON(http.StatusOK, res)
			return
		}
		// 验证成功
		res.Code = codes.CODE_SUCCESS_200
		res.Msg = "success"
		res.Data = log
		c.JSON(http.StatusOK, res)
		return
	}

	res.Code = codes.CODE_ERR_INVALID
	res.Msg = "failed: not a valid ido transfer"
	c.JSON(http.StatusOK, res)
	return
}
