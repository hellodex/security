package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"gorm.io/gorm"
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

func GetWalletByUserNo(db *gorm.DB, req *common.UserStructReq, validChains []string, channel any) ([]common.AuthGetBackWallet, error) {

	if req == nil || req.Uuid == "" || len(req.Uuid) <= 0 {
		return nil, fmt.Errorf("user uuid is empty")
	}
	//获取用户所有的钱包组
	resultList := make([]common.AuthGetBackWallet, 0)
	var walletGroups []model.WalletGroup
	err := db.Model(&model.WalletGroup{}).Where("user_id = ? and vault_type < 1", req.Uuid).Find(&walletGroups).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	groupSize := len(walletGroups)
	//没有达到最少组数，创建新的组
	if groupSize < req.LeastGroups {
		needCreateGropesSize := req.LeastGroups - groupSize
		for _ = range needCreateGropesSize {
			strmneno, err := enc.NewKeyStories()
			if err != nil {
				return nil, fmt.Errorf("can not create wallet group : %s", err.Error())
			}
			te := &model.WalletGroup{
				UserID:         req.Uuid,
				CreateTime:     time.Now(),
				EncryptMem:     strmneno,
				EncryptVersion: fmt.Sprintf("AES:%d", 1),
				Nonce:          int(enc.Porter().GetNonce()),
				VaultType:      0,
			}
			db.Save(te)
			walletGroups = append(walletGroups, *te)
		}
	}
	//获取每一组的钱包地址
	for _, g := range walletGroups {
		var wgs []model.WalletGenerated
		db.Model(&model.WalletGenerated{}).
			Where("user_id = ? and group_id = ? and status = ? and chain_code IN ?", req.Uuid, g.ID, "00", validChains).Find(&wgs)

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
				resultList = append(resultList, common.AuthGetBackWallet{
					WalletAddr: w.Wallet,
					WalletId:   w.ID,
					GroupID:    w.GroupID,
					ChainCode:  w.ChainCode,
					VaultType:  g.VaultType,
				})
			}
			continue
		}

		// 需要创建的chaincode钱包
		for _, v := range needCreates {
			wal, err := wallet.Generate(&g, wallet.ChainCode(v))
			if err != nil {
				mylog.Errorf("create wallet error %v", err)
				continue
			}

			wg := model.WalletGenerated{
				UserID:         req.Uuid,
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
			}
		}
		for _, wg := range wgs {
			resultList = append(resultList, common.AuthGetBackWallet{
				WalletAddr: wg.Wallet,
				WalletId:   wg.ID,
				GroupID:    wg.GroupID,
				ChainCode:  wg.ChainCode,
				VaultType:  g.VaultType,
			})
		}
	}
	vaults := GetMemeVaultWallet(db, req, channel)
	if vaults != nil && len(vaults) > 0 {
		resultList = append(resultList, vaults...)
	}
	walletKeys := make([]model.WalletKeys, 0)
	resultListRes := make([]common.AuthGetBackWallet, 0)
	for _, r := range resultList {
		walletKey := common.MyIDStr()
		time.Sleep(time.Millisecond)
		walletKeys = append(walletKeys, model.WalletKeys{
			WalletKey:  walletKey,
			WalletId:   r.WalletId,
			Channel:    req.Channel,
			ExpireTime: req.ExpireTime,
			UserId:     req.Uuid,
		})
		r.WalletKey = walletKey
		r.ExpireTime = req.ExpireTime
		resultListRes = append(resultListRes, r)
	}
	mylog.Info("重新登陆删除过期的walletKeys: ", req.Uuid, req.Channel)
	store.WalletKeyDelByUserIdAndChannel(req.Uuid, req.Channel)
	err = store.WalletKeySaveBatch(walletKeys)
	if err != nil {
		return nil, err
	}
	return resultListRes, nil
}

func AuthSig(c *gin.Context) {
	var req common.AuthSigWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Infof("AuthSig &req Err  : %v", err)
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = fmt.Sprintf("Invalid request,c.ShouldBindJSON(&req),err:: %s", err.Error())
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
	var txhash string
	var sig []byte
	var err error
	memeVaultFlag := false
	if req.Config.VaultTip.Sign() > 0 {
		memeVaultFlag = true
	}
	userReceive := decimal.Zero
	// 是否需要确认交易状态
	req.Config.ShouldConfirm = true
	// 确认交易的超时时间
	req.Config.ConfirmTimeOut = 15 * time.Second

	if !limitFlag {
		if memeVaultFlag {
			txhash, sig, err = chain.JUPHandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
		} else {
			txhash, sig, err = chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
		}

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
			res.Data = common.SignRes{
				Signature: sigStr,
				Wallet:    wg.Wallet,
				Tx:        txhash,
			}
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
		// 限价交易
		isMemeVaultWalletTrade := IsMemeVaultWalletTrade(db, 0, &wg)

		limitOrderParam := &req.LimitOrderParams
		limitOrderParam.IsMemeVaultWalletTrade = isMemeVaultWalletTrade
		req.LimitOrderParams.IsMemeVaultWalletTrade = isMemeVaultWalletTrade
		if isMemeVaultWalletTrade && wg.ChainCode == "SOLANA" {
			//limitOrderParam.JitoTipLamports = req.Config.Tip
		}
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
				fromString, err2 := decimal.NewFromString(limitOrderParam.Slippage)
				if err2 != nil {
					fromString = decimal.NewFromFloat(0.05).Mul(decimal.NewFromInt(int64(i)))
				}
				limitOrderParam.Slippage = fromString.Add(decimal.NewFromFloat(0.05)).String()

			}

			swapDataMap, swapRes, errL := swapData.GetSwapDataWithOpts(i, swapDataMap, limitOrderParam)
			switch swapRes.Plat {
			case codes.Okx:
				okxResponse := swapRes.Data.(swapData.OkxResponse)
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
			case codes.Bsc0x:
				swapResEvm := swapRes.Data.(map[string]interface{})
				data, exData := swapResEvm["singData"]
				toF, exTo := swapResEvm["to"]
				amount1Interface, _ := swapResEvm["value"]
				gasInterface, exGas := swapResEvm["gas"]
				gasPriceInterface, exGasPrice := swapResEvm["gasPrice"]
				if !exData || !exTo || !exGas || !exGasPrice {
					res.Code = codes.CODE_ERR_102
					res.Msg = "bad request bscOx res"
					res.Data = common.SignRes{
						Signature: "",
						Wallet:    wg.Wallet,
						Tx:        "",
						CallData:  swapDataMap,
					}
					c.JSON(http.StatusOK, res)
					return
				}
				msg = data.(string)
				to = toF.(string)
				amount1 := amount1Interface.(string)
				if amount1 == "" {
					amount1 = "0"
				}
				amount = new(big.Int)
				amount.SetString(amount1, 10)
				gasStr := gasInterface.(string)
				if gasStr == "" {
					gasStr = "0"
				}
				gas, e := decimal.NewFromString(gasStr)
				if e != nil {
					gas = decimal.NewFromInt(0)
				}
				if gas.Sign() > 0 {
					req.Config.UnitLimit = gas.Mul(decimal.NewFromFloat(1.2)).BigInt()
				}
				gasPStr := gasPriceInterface.(string)
				if gasPStr == "" {
					gasPStr = "0"
				}
				gasP, e := decimal.NewFromString(gasPStr)
				if e != nil {
					gasP = decimal.NewFromInt(0)
				}
				if gas.Sign() > 0 {
					req.Config.UnitPrice = gasP.Mul(decimal.NewFromFloat(1.2)).BigInt()
				}
			case codes.Jup:
				swapResSol := swapRes.Data.(map[string]interface{})
				data, exData := swapResSol["singData"]
				if !exData || data == nil || len(data.(string)) == 0 {
					res.Code = codes.CODE_ERR_102
					res.Msg = "bad request jup res"
					res.Data = common.SignRes{
						Signature: "",
						Wallet:    wg.Wallet,
						Tx:        "",
						CallData:  swapDataMap,
					}
					c.JSON(http.StatusOK, res)
					return
				}
				msg = data.(string)
				to = req.To
				if jupvaultTip, ok := swapResSol["vaultTip"].(*big.Int); ok {
					req.Config.VaultTip = jupvaultTip
				}
				if userReceiveIn, ok := swapResSol["userReceive"].(decimal.Decimal); ok {
					userReceive = userReceiveIn
				}

			default:
				res.Code = codes.CODE_ERR_102
				res.Msg = "bad request swap res"
				res.Data = common.SignRes{
					Signature: "",
					Wallet:    wg.Wallet,
					Tx:        "",
					CallData:  swapDataMap,
				}
				c.JSON(http.StatusOK, res)
				return
			}
			if isMemeVaultWalletTrade {
				txhash, sig, err = chain.JUPHandleMessage(chainConfig, msg, to, req.Type, amount, &req.Config, &wg)
			} else {
				txhash, sig, err = chain.HandleMessage(chainConfig, msg, to, req.Type, amount, &req.Config, &wg)
			}
			//txhash, sig, err := chain.HandleMessage(chainConfig, msg, to, req.Type, amount, &req.Config, &wg, true)
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
				Signature:   msg,
				Wallet:      wg.Wallet,
				Tx:          txhash,
				CallData:    swapDataMap,
				UserReceive: userReceive,
			}
			c.JSON(http.StatusOK, res)
			return
		}
		mylog.Info("swap fail,retry later,del limitkey ", req.LimitOrderParams.LimitOrderKey)
		store.LimitKeyDelByKey(req.LimitOrderParams.LimitOrderKey)
		res.Code = codes.CODE_ERR_102
		res.Msg = "bad request swap res,retry later"
		res.Data = common.SignRes{
			Signature: "",
			Wallet:    wg.Wallet,
			Tx:        "",
			CallData:  swapDataMap,
		}
		c.JSON(http.StatusOK, res)
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
	instructions, err2 := getCloseAtaAccountsInstructionsTx(chainConfig, &req.Config, feePayer)
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
	lastTx := ""
	//分批
	//computeUnitPrice := uint64(16000000)
	//computeUnitLimit := uint32(202000) // 设置为 202,000 计算单位
	computeUnitPrice := uint64(100000)
	computeUnitLimit := uint32(202000) // 设置为 202,000 计算单位
	reqconf := &req.Config
	if reqconf != nil {
		if reqconf.UnitPrice != nil && reqconf.UnitPrice.Uint64() > 0 {
			computeUnitPrice = reqconf.UnitPrice.Uint64()
		}
		if reqconf.UnitLimit != nil && reqconf.UnitLimit.Uint64() > 0 {
			computeUnitLimit = uint32(reqconf.UnitLimit.Uint64())
		}
	}
	batchSlices := batchSlice(instructions, 20)
	req.Config.ShouldConfirm = false
	req.Config.ConfirmTimeOut = 5 * time.Second
	for i, ins := range batchSlices {
		ins = append(ins, computebudget.NewSetComputeUnitLimitInstruction(computeUnitLimit).Build())
		ins = append(ins, computebudget.NewSetComputeUnitPriceInstruction(computeUnitPrice).Build())
		tx, err := solana.NewTransaction(
			ins,
			solana.Hash{},
			solana.TransactionPayer(feePayer),
		)
		toBase64 := tx.MustToBase64()
		req.Message = toBase64
		//不需要确认状态
		txhash, sig, err := chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
		if len(batchSlices) > 1 {
			time.Sleep(time.Millisecond * 400)
		}
		lastTx = txhash
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
		Signature: lastTx,
		Wallet:    wg.Wallet,
		Tx:        "",
	}
	c.JSON(http.StatusOK, res)
}

func getCloseAtaAccountsInstructionsTx(t *config.ChainConfig, reqConfig *common.OpConfig, payer solana.PublicKey) ([]solana.Instruction, error) {
	ctx := context.Background()
	rpcUrlDefault := t.GetRpc()[0]

	if reqConfig != nil && reqConfig.Rpc != "" {
		rpcUrlDefault = reqConfig.Rpc
	}
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
