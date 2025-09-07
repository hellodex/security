package controller

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"gorm.io/gorm"

	"github.com/hellodex/HelloSecurity/store"
	"github.com/hellodex/HelloSecurity/swapData"
	"github.com/mr-tron/base58"
	"github.com/shopspring/decimal"

	"github.com/hellodex/HelloSecurity/api/common"
	chain "github.com/hellodex/HelloSecurity/chain"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
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

	//mylog.Info("accept req: ", req.Message)

	chainConfig := config.GetRpcConfig(wg.ChainCode)
	var txhash string
	var sig []byte
	var err error
	memeVaultFlag := false
	if req.Config.VaultTip != nil && req.Config.VaultTip.Sign() > 0 {
		memeVaultFlag = true
	}
	userReceive := decimal.Zero
	// 是否需要确认交易状态
	req.Config.ShouldConfirm = true
	// 确认交易的超时时间
	req.Config.ConfirmTimeOut = 30 * time.Second

	if !limitFlag {

		if memeVaultFlag {
			if req.UserId == "1846030691993784320" {
				//txhash, sig, err = chain.HandleMessageTest(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
				//txhash, sig, err = chain.MemeVaultHandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
			} else {
				//txhash, sig, err = chain.MemeVaultHandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
			}
			//txhash, sig, err = chain.MemeVaultHandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)

		} else {

			if req.UserId == "1846030691993784320" {
				//txhash, sig, err = chain.HandleMessageTest(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
				txhash, sig, err = chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
			} else {
				txhash, sig, err = chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
			}
			//txhash, sig, err = chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)

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
			mylog.Error("返回api时错误", err)
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
		limitOrderSlippageShouldeAdd := true
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
			if i > 1 && limitOrderSlippageShouldeAdd {
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
				req.Config.VaultTip = okxResponse.VaultTip
				userReceive = okxResponse.UserReceive
				if len(okxResponse.JitoCallData) > 0 {
					req.Config.JitoCalldata = okxResponse.JitoCallData
				}
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
				// 冲狗基金交易50%归属基金钱包
				//txhash, sig, err = chain.MemeVaultHandleMessage(chainConfig, msg, to, req.Type, amount, &req.Config, &wg)
			} else {
				txhash, sig, err = chain.HandleMessage(chainConfig, msg, to, req.Type, amount, &req.Config, &wg)
			}
			//txhash, sig, err := chain.HandleMessage(chainConfig, msg, to, req.Type, amount, &req.Config, &wg, true)
			sigStr := ""
			if err != nil && (strings.Contains(err.Error(), "error: 0x1771") ||
				strings.Contains(err.Error(), "error: 6001") ||
				strings.Contains(err.Error(), "Error Message: slippage") ||
				strings.Contains(err.Error(), "Custom:6001") ||
				//strings.Contains(err.Error(), "status:failed") ||
				strings.Contains(err.Error(), "status:unpub")) {
				swapDataMap["callDataErr"+strconv.Itoa(i)] = err.Error()
				swapDataMap["callDataErrTxHash"+strconv.Itoa(i)] = txhash

				if strings.Contains(err.Error(), "status:unpub") {
					limitOrderSlippageShouldeAdd = false
					if i >= 3 {
						break
					}
				} else {
					limitOrderSlippageShouldeAdd = true
				}
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

	//mylog.Info("accept req: ", req.Message)

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
	//computeUnitPrice := uint64(100000)
	//computeUnitLimit := uint32(202000) // 设置为 202,000 计算单位
	reqconf := &req.Config
	if reqconf != nil {
		//if reqconf.UnitPrice != nil && reqconf.UnitPrice.Uint64() > 0 {
		//	computeUnitPrice = reqconf.UnitPrice.Uint64()
		//}
		//if reqconf.UnitLimit != nil && reqconf.UnitLimit.Uint64() > 0 {
		//	computeUnitLimit = uint32(reqconf.UnitLimit.Uint64())
		//}
	}
	batchSlices := batchSlice(instructions, 20)
	req.Config.ShouldConfirm = false
	req.Config.ConfirmTimeOut = 5 * time.Second
	for i, ins := range batchSlices {
		//ins = append(ins, computebudget.NewSetComputeUnitLimitInstruction(computeUnitLimit).Build())
		//ins = append(ins, computebudget.NewSetComputeUnitPriceInstruction(computeUnitPrice).Build())
		tx, err := solana.NewTransaction(
			ins,
			solana.Hash{},
			solana.TransactionPayer(feePayer),
		)
		toBase64 := tx.MustToBase64()
		req.Message = toBase64
		//不需要确认状态
		req.Config.Type = "AuthForceCloseAll"
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
			_ = db.Model(&model.WalletLog{}).Save(wl).Error
			if strings.Contains(err.Error(), "Insufficient funds for fee") {
				res.Code = codes.CODE_ERR
				msg := "请先预留足够SOL支付Gas费"

				res.Msg = msg
				res.Data = common.SignRes{
					Signature: lastTx,
					Wallet:    wg.Wallet,
					Tx:        "",
				}
				c.JSON(http.StatusOK, res)
				return
			}
			break
		} else {
			err1 := db.Model(&model.WalletLog{}).Save(wl).Error
			if err1 != nil {
				mylog.Error("save log error ", err)
			}
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

func AuthForceCloseAll(c *gin.Context) {
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

	chainConfig := config.GetRpcConfig(wg.ChainCode)

	feePayer := solana.MustPublicKeyFromBase58(wg.Wallet)

	instructions, err2 := getCloseAtaAccountsInstructionsByAtas(chainConfig, feePayer, req.TokenList)
	if err2 != nil {
		res.Code = codes.CODE_ERR_102
		res.Msg = "无法获取账户信息"
		c.JSON(http.StatusOK, res)
		return
	}
	if len(instructions) <= 0 {
		res.Code = codes.CODE_SUCCESS_200
		res.Msg = "无可关闭账户"
		c.JSON(http.StatusOK, res)
		return
	}
	txHashs := make([]string, 0)
	// 批量处理
	// 10个一批
	count := len(instructions)
	lastTx := ""
	reqconf := &req.Config
	if reqconf != nil {

	}
	//烧币和关闭是两个指令所以批量需要x2
	batchSlices := batchSlice(instructions, 8*2)
	req.Config.ShouldConfirm = false
	req.Config.ConfirmTimeOut = 5 * time.Second
	for i, ins := range batchSlices {
		tx, err := solana.NewTransaction(
			ins,
			solana.Hash{},
			solana.TransactionPayer(feePayer),
		)
		toBase64 := tx.MustToBase64()
		req.Message = toBase64
		//不需要确认状态
		req.Config.Type = "AuthForceCloseAll"
		//mylog.Infof("AuthForceCloseAll传递参数 %v: ", req.Config.Type)
		txhash, sig, err := chain.HandleMessage(chainConfig, req.Message, req.To, req.Type, req.Amount, &req.Config, &wg)
		if err != nil {
			mylog.Infof("批量关闭Ata失败第%d批,%d个账户,tx:%s,err:%+v", i+1, len(ins), txhash, err)
			continue
		}

		if len(batchSlices) > 1 {
			time.Sleep(time.Millisecond * 400)
		}
		lastTx = txhash
		sigStr := ""
		txHashs = append(txHashs, txhash)
		if len(sig) > 0 {
			sigStr = base64.StdEncoding.EncodeToString(sig)
		}
		mylog.Infof("批量关闭Ata成功第%d批,%d个账户,tx:%s,sig:%s,err:%+v", i+1, len(ins), txhash, sigStr)
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
			_ = db.Model(&model.WalletLog{}).Save(wl).Error
			if strings.Contains(err.Error(), "Insufficient funds for fee") {
				res.Code = codes.CODE_ERR
				msg := "请先预留足够SOL支付Gas费"

				res.Msg = msg
				res.Data = common.SignRes{
					Signature: lastTx,
					Wallet:    wg.Wallet,
					Tx:        "",
				}
				c.JSON(http.StatusOK, res)
				return
			}
			break
		} else {
			err1 := db.Model(&model.WalletLog{}).Save(wl).Error
			if err1 != nil {
				mylog.Error("save log error ", err)
			}
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

// getCloseAtaAccountsInstructionsTx 生成关闭所有余额为0的Associated Token Account（ATA）的指令
// 参数：
// - t: 链配置，包含RPC端点等信息
// - reqConfig: 操作配置，可覆盖默认RPC端点
// - payer: 支付账户，用于支付交易费用和接收退款
// 返回：
// - 指令列表和可能的错误
func getCloseAtaAccountsInstructionsTx(t *config.ChainConfig, reqConfig *common.OpConfig, payer solana.PublicKey) ([]solana.Instruction, error) {
	ctx := context.Background()

	// 获取RPC端点
	rpcUrlDefault := t.GetRpc()[0]
	if reqConfig != nil && reqConfig.Rpc != "" {
		rpcUrlDefault = reqConfig.Rpc
	}
	client := rpc.New(rpcUrlDefault)

	// 查询TokenProgramID的代币账户
	accounts, err := client.GetTokenAccountsByOwner(
		ctx,
		payer,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Commitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		mylog.Errorf("Failed to get token accounts: %v", err)
		return nil, fmt.Errorf("get token accounts error: %w", err)
	}

	// 查询Token2022ProgramID的代币账户
	accounts2022, err := client.GetTokenAccountsByOwner(
		ctx,
		payer,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.Token2022ProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Commitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		mylog.Errorf("Failed to get token2022 accounts: %v", err)
		return nil, fmt.Errorf("get token2022 accounts error: %w", err)
	}

	// 合并TokenProgram和Token2022Program的账户列表
	allAccounts := append(accounts.Value, accounts2022.Value...)
	var instructions []solana.Instruction
	seenAccounts := make(map[string]bool) // 用于去重账户

	// 处理每个代币账户
	for _, account := range allAccounts {
		accountPubkey := account.Pubkey.String()

		// 跳过重复的账户
		if seenAccounts[accountPubkey] {
			mylog.Warnf("Duplicate account detected: %s", accountPubkey)
			continue
		}
		seenAccounts[accountPubkey] = true

		// 验证账户是否属于Token程序
		if account.Account.Owner != solana.TokenProgramID && account.Account.Owner != solana.Token2022ProgramID {
			mylog.Warnf("Skipping non-TokenProgram account: %s, owner: %s", accountPubkey, account.Account.Owner.String())
			continue
		}

		// 暂时跳过Token-2022账户（因数据结构可能不同）
		//if account.Account.Owner == solana.Token2022ProgramID {
		//	mylog.Warnf("Skipping Token-2022 account: %s (not fully supported)", accountPubkey)
		//	continue
		//}

		// 检查数据是否为空
		data := account.Account.Data.GetBinary()
		if len(data) == 0 {
			mylog.Warnf("Empty data for account: %s", accountPubkey)
			continue
		}

		// 解码账户数据
		var tokAcc token.Account
		dec := bin.NewBinDecoder(data)
		if err := dec.Decode(&tokAcc); err != nil {
			mylog.Errorf("Failed to decode token account %s (data length: %d): %v", accountPubkey, len(data), err)
			continue
		}

		// 跳过余额大于0的账户
		if tokAcc.Amount > 0 {
			mylog.Infof("Skipping account with non-zero balance: %s, amount=%d", accountPubkey, tokAcc.Amount)
			continue
		}

		// 记录账户信息
		mylog.Infof("Processing token account: payer=%s, account=%s, mint=%s, amount=%d",
			payer.String(), accountPubkey, tokAcc.Mint.String(), tokAcc.Amount)

		// 生成CloseAccount指令
		var closeIx solana.Instruction

		if account.Account.Owner == solana.Token2022ProgramID {
			// Token2022账户需要手动构造指令
			closeIx = &solana.GenericInstruction{
				ProgID: solana.Token2022ProgramID,
				AccountValues: []*solana.AccountMeta{
					{PublicKey: account.Pubkey, IsSigner: false, IsWritable: true},
					{PublicKey: payer, IsSigner: false, IsWritable: true},
					{PublicKey: payer, IsSigner: true, IsWritable: false},
				},
				DataBytes: []byte{9}, // CloseAccount指令索引
			}
			mylog.Infof("Generated Token2022 CloseAccount instruction for account: %s", accountPubkey)
		} else {
			// 普通Token账户使用标准方法
			closeIx = token.NewCloseAccountInstruction(
				account.Pubkey,
				payer,
				payer,
				[]solana.PublicKey{},
			).Build()
			mylog.Infof("Generated CloseAccount instruction for account: %s", accountPubkey)
		}

		instructions = append(instructions, closeIx)
	}

	// 如果生成了指令，模拟交易以验证
	//if len(instructions) > 0 {
	//	if err := simulateInstructions(client, instructions, payer); err != nil {
	//		mylog.Errorf("Transaction simulation failed: %v", err)
	//		return nil, fmt.Errorf("transaction simulation failed: %w", err)
	//	}
	//	mylog.Infof("Transaction simulation passed for %d instructions", len(instructions))
	//} else {
	//	mylog.Info("No accounts to close")
	//}

	return instructions, nil
}

func getCloseAtaAccountsInstructionsByAtas(t *config.ChainConfig, payer solana.PublicKey, tokenList []common.CloseTokenAccountInfo) ([]solana.Instruction, error) {
	var instructions []solana.Instruction

	// 获取RPC客户端来查询账户信息
	ctx := context.Background()

	rpcUrl := t.GetRpc()[0]
	client := rpc.New(rpcUrl)

	for _, tokenAccount := range tokenList {

		accountPubkey, err := solana.PublicKeyFromBase58(tokenAccount.Account)
		if err != nil {
			mylog.Errorf("Invalid account address: %s, error: %v", tokenAccount.Account, err)
			return nil, fmt.Errorf("invalid account address %s: %w", tokenAccount.Account, err)
		}

		// 安全解析mint地址
		mintPubkey, err := solana.PublicKeyFromBase58(tokenAccount.Mint)
		if err != nil {
			mylog.Errorf("Invalid mint address: %s, error: %v", tokenAccount.Mint, err)
			return nil, fmt.Errorf("invalid mint address %s: %w", tokenAccount.Mint, err)
		}

		// 查询账户信息以确定是否为Token-2022
		accountInfo, err := client.GetAccountInfo(ctx, accountPubkey)
		if err != nil {
			mylog.Errorf("Failed to get account info for %s: %v", tokenAccount.Account, err)
			return nil, fmt.Errorf("failed to get account info for %s: %w", tokenAccount.Account, err)
		}

		if accountInfo == nil || accountInfo.Value == nil {
			mylog.Warnf("Account %s not found or empty", tokenAccount.Account)
			continue
		}

		// 判断是否为Token-2022账户
		isToken2022 := accountInfo.Value.Owner == solana.Token2022ProgramID

		if tokenAccount.Amount > 0 {
			var burnIx solana.Instruction

			if isToken2022 {
				// Token-2022需要手动构造Burn指令
				amountBytes := make([]byte, 8)
				binary.LittleEndian.PutUint64(amountBytes, tokenAccount.Amount)
				burnIx = &solana.GenericInstruction{
					ProgID: solana.Token2022ProgramID,
					AccountValues: []*solana.AccountMeta{
						{PublicKey: accountPubkey, IsSigner: false, IsWritable: true},
						{PublicKey: mintPubkey, IsSigner: false, IsWritable: true},
						{PublicKey: payer, IsSigner: true, IsWritable: false},
					},
					DataBytes: append([]byte{8}, amountBytes...), // 8是Burn指令索引
				}
				mylog.Infof("Generated Token2022 Burn instruction for account: %s, amount: %d", accountPubkey, tokenAccount.Amount)
			} else {
				// 标准Token使用现有方法
				burnIx = token.NewBurnInstruction(
					tokenAccount.Amount,
					accountPubkey,
					mintPubkey,
					payer,
					[]solana.PublicKey{},
				).Build()
			}

			instructions = append(instructions, burnIx)
		}

		var closeIx solana.Instruction

		if isToken2022 {
			// Token2022账户需要手动构造指令
			closeIx = &solana.GenericInstruction{
				ProgID: solana.Token2022ProgramID,
				AccountValues: []*solana.AccountMeta{
					{PublicKey: accountPubkey, IsSigner: false, IsWritable: true},
					{PublicKey: payer, IsSigner: false, IsWritable: true},
					{PublicKey: payer, IsSigner: true, IsWritable: false},
				},
				DataBytes: []byte{9}, // CloseAccount指令索引
			}
			mylog.Infof("Generated Token2022 CloseAccount instruction for account: %s", accountPubkey)
		} else {
			// 普通Token账户使用标准方法
			closeIx = token.NewCloseAccountInstruction(
				accountPubkey,
				payer,
				payer,
				[]solana.PublicKey{},
			).Build()
		}

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
