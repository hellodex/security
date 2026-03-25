package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	coreconfig "github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/store"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet/enc"

	log "github.com/ethereum/go-ethereum/log"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func AuthClaimPumpCashback(c *gin.Context) {
	var req common.AuthSigWalletRequest
	res := common.Response{}
	res.Timestamp = time.Now().Unix()

	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = fmt.Sprintf("Invalid request: %s", err.Error())
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.Channel) == 0 {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "channel is empty"
		c.JSON(http.StatusOK, res)
		return
	}

	if len(req.Message) == 0 {
		res.Code = codes.CODE_ERR_BAT_PARAMS
		res.Msg = "message is empty"
		c.JSON(http.StatusOK, res)
		return
	}

	wk, err2 := store.WalletKeyCheckAndGet(req.WalletKey)
	if err2 != nil || wk == nil {
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "登录信息已失效，请重新登录"
		c.JSON(http.StatusOK, res)
		return
	}

	if wk.WalletKey == "" || wk.UserId != req.UserId {
		store.WalletKeyDelByUserIdAndChannel(req.UserId, req.Channel)
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "walletkey check fail"
		c.JSON(http.StatusOK, res)
		return
	}
	walletId := wk.WalletId

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
		store.WalletKeyDelByUserIdAndChannel(req.UserId, req.Channel)
		res.Code = codes.CODE_ERR_AUTH_FAIL
		res.Msg = "user id not match"
		c.JSON(http.StatusOK, res)
		return
	}

	// Dynamic RPC Config handled later

	txhash := ""
	sigStr := ""
	var err error

	// 1. Decode base64 transaction
	messageBytes, err := base64.StdEncoding.DecodeString(req.Message)
	if err != nil {
		res.Code = codes.CODES_ERR_TX
		res.Msg = "Invalid base64 message"
		c.JSON(http.StatusOK, res)
		return
	}

	// 2. Parse transaction bytes
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(messageBytes))
	if err != nil {
		res.Code = codes.CODES_ERR_TX
		res.Msg = "Failed to parse transaction: " + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	// 3. Sign transaction message
	msgBytes, _ := tx.Message.MarshalBinary()
	sigBytes, err := enc.Porter().SigSol(&wg, msgBytes)
	if err != nil {
		res.Code = codes.CODES_ERR_TX
		res.Msg = "Failed to sign transaction: " + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	// 4. Attach signature
	tx.Signatures = []solana.Signature{solana.Signature(sigBytes)}
	sigStr = base64.StdEncoding.EncodeToString(sigBytes)

	// 5. Build dynamic RPC client
	rpcUrl := req.Config.Rpc
	if rpcUrl == "" {
		baseConfig := coreconfig.GetRpcConfig(wg.ChainCode)
		if len(baseConfig.GetRpc()) > 0 {
			rpcUrl = baseConfig.GetRpc()[0]
		}
	}

	if rpcUrl == "" {
		res.Code = codes.CODES_ERR_TX
		res.Msg = "No RPC URL configured"
		c.JSON(http.StatusOK, res)
		return
	}

	rpcClients := make([]*rpc.Client, 0)
	for _, u := range strings.Split(rpcUrl, ",") {
		if strings.TrimSpace(u) != "" {
			rpcClients = append(rpcClients, rpc.New(strings.TrimSpace(u)))
		}
	}

	// 6. Broadcast transaction directly without simulation (SkipPreflight: true)
	if len(rpcClients) > 0 {
		opts := rpc.TransactionOpts{SkipPreflight: true}
		sig, errTx := rpcClients[0].SendTransactionWithOpts(context.Background(), tx, opts)
		if errTx != nil {
			err = errTx
		} else {
			txhash = sig.String()
		}
	} else {
		err = fmt.Errorf("empty valid RPC clients list")
	}

	// 7. Store wallet operational logs
	wl := &model.WalletLog{
		WalletID:  int64(walletId),
		Wallet:    wg.Wallet,
		Data:      req.Message,
		Sig:       sigStr,
		ChainCode: wg.ChainCode,
		Operation: "claimPumpCashback",
		OpTime:    time.Now(),
		TxHash:    txhash,
	}
	if err != nil {
		wl.Err = err.Error()
	}
	err1 := db.Model(&model.WalletLog{}).Save(wl).Error
	if err1 != nil {
		log.Error("save log error ", err1)
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
}
