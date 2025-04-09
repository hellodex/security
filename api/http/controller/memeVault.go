package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/chain"
	"github.com/hellodex/HelloSecurity/codes"
	"github.com/hellodex/HelloSecurity/config"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet"
	"github.com/hellodex/HelloSecurity/wallet/enc"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

type MemeVaultListReq struct {
	Uuid             string     `json:"uuid"`
	VaultType        int        `json:"vaultType"`
	UserType         string     `json:"userType"`
	CreateTime       *time.Time `  json:"createTime"`
	ExpireTime       *time.Time `  json:"expireTime"`
	ExpireTimeBefore *time.Time `  json:"expireTimeBefore"`
	CreateTimeBefore *time.Time ` json:"createTimeBefore"`
	Page             int        `json:"page"`
	PageSize         int        `json:"pageSize"`
}
type MemeVaultSupportListReq struct {
	ID               uint64          `json:"id"`
	UUID             string          `json:"uuid"`
	GroupId          uint64          `json:"groupId"`
	WalletID         int64           `json:"walletId"`
	Wallet           string          `json:"wallet"`
	FromWallet       string          `json:"fromWallet"`
	FromWalletID     int64           `json:"fromWalletID"`
	ChainCode        string          `json:"chainCode"`
	VaultType        int             `json:"vaultType"`
	Status           int             `json:"status"` // 0:成功 1:失败
	SupportAddress   string          `json:"supportAddress"`
	SupportAmount    decimal.Decimal `json:"supportAmount"`
	Channel          string          `json:"channel"`
	CreateTime       time.Time       `json:"createTime"`
	CreateTimeBefore time.Time       `json:"createTimeBefore"`
	UpdateTime       time.Time       `json:"updateTime"`
	Page             int             `json:"page"`
	PageSize         int             `json:"pageSize"`
}
type ClaimToMemeVaultReq struct {
	WalletID int64           `json:"walletID"`
	Channel  string          `json:"channel"`
	Config   common.OpConfig `json:"config"`
}

func VaultSupportList(c *gin.Context) {
	var req MemeVaultSupportListReq
	res := common.Response{}
	before5Years := time.Now().AddDate(-5, 0, 0)
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:parameterFormatError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	mylog.Infof("VaultSupportList req: %+v", req)
	db := system.GetDb()
	query := db.Model(&model.MemeVaultSupport{})
	if len(req.UUID) > 0 {
		query = query.Where("uuid = ?", req.UUID)
	}
	if req.VaultType > 0 {
		query = query.Where("vault_type = ?", req.VaultType)
	}
	if req.WalletID > 0 {
		query = query.Where("wallet_id = ?", req.WalletID)
	}
	if len(req.Wallet) > 0 {
		query = query.Where("wallet = ?", req.Wallet)
	}
	if len(req.FromWallet) > 0 {
		query = query.Where("from_wallet = ?", req.FromWallet)
	}
	if req.FromWalletID > 0 {
		query = query.Where("from_wallet_id = ?", req.FromWalletID)
	}
	if len(req.ChainCode) > 0 {
		query = query.Where("chain_code = ?", req.ChainCode)
	}
	if len(req.SupportAddress) > 0 {
		query = query.Where("support_token = ?", req.SupportAddress)
	}
	//大于传递的创建时间
	if req.CreateTime.After(before5Years) {
		query = query.Where("create_time >= ?", req.CreateTime)
	}
	//小于传递的创建时间
	if req.CreateTimeBefore.After(before5Years) {
		query = query.Where("create_time <= ?", req.CreateTimeBefore)
	}
	// 计算分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20 // 默认每页20条
	}
	offset := (req.Page - 1) * req.PageSize
	var memes []model.MemeVaultSupport
	err := query.Order("ID DESC").Limit(req.PageSize).Offset(offset).Find(&memes).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:queryError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	PaginatedResult := common.PaginatedResult[model.MemeVaultSupport]{
		Page:     req.Page,
		PageSize: req.PageSize,
		Data:     memes}
	res.Data = PaginatedResult
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)

}
func MemeVaultSupportListByUUID(c *gin.Context) {
	var req MemeVaultListReq
	res := common.Response{}
	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Error("MemeVaultSupport req format error:%s", err)
		res.Code = codes.CODE_ERR_REQFORMAT
		res.Msg = "Invalid request" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("MemeVaultSupportListByUUID req: %+v", req)
	if len(req.Uuid) == 0 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:uuid is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	if req.VaultType < 0 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:VaultType is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	db := system.GetDb()
	var vault model.MemeVault
	err := db.Model(&model.MemeVault{}).Where("uuid = ? and vault_type =?  order by id",
		req.Uuid, req.VaultType).Take(&vault).Error
	if err != nil && vault.ID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:vault is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	var vaultSp []model.MemeVaultSupport
	err = db.Model(&model.MemeVaultSupport{}).Where("uuid = ? and vault_type =?  order by id desc limit 30",
		req.Uuid, req.VaultType).Find(&vaultSp).Error
	if err == nil && len(vaultSp) > 0 {
		vault.VaultSupport = vaultSp
	}
	if len(vaultSp) < 1 {
		vault.VaultSupport = []model.MemeVaultSupport{}
	}
	res.Code = codes.CODE_ERR
	res.Msg = "success"
	res.Data = vault
	c.JSON(http.StatusOK, res)
	return
}
func MemeVaultUpdate(c *gin.Context) {
	var req model.MemeVault
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:parameterFormatError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("MemeVaultUpdate req: %+v", req)
	db := system.GetDb()
	if req.ID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:id is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	inDb := model.MemeVault{}
	err := db.Model(&model.MemeVault{}).Where("id = ?", req.ID).Take(&inDb).Error
	if err != nil && inDb.ID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:id not exist"
		c.JSON(http.StatusOK, res)
		return
	}
	up := db.Model(&model.MemeVault{}).Where("id = ?", req.ID)
	updates := map[string]interface{}{}

	if len(req.ChainIndex) > 1 && req.ChainIndex != inDb.ChainIndex {
		updates["chain_index"] = req.ChainIndex
	}
	if len(req.UserType) > 1 && req.UserType != inDb.UserType {
		updates["user_type"] = req.UserType
	}
	if req.MinAmount.GreaterThanOrEqual(decimal.NewFromFloat(1.0)) &&
		!req.MinAmount.Equal(inDb.MinAmount) {
		updates["min_amount"] = req.MinAmount
	}
	if req.MaxAmount.GreaterThanOrEqual(req.MinAmount) &&
		req.MaxAmount.GreaterThanOrEqual(decimal.NewFromFloat(1.0)) &&
		!req.MaxAmount.Equal(inDb.MaxAmount) {
		updates["max_amount"] = req.MinAmount
	}
	if req.Status > 0 && req.Status != inDb.Status {
		updates["status"] = req.Status
	}
	if req.StartTime.After(time.Now()) && !req.StartTime.Equal(inDb.StartTime) {
		updates["start_time"] = req.StartTime
	}
	if req.ExpireTime.After(time.Now()) && !req.ExpireTime.Equal(inDb.ExpireTime) {
		updates["expire_time"] = req.ExpireTime
	}
	if len(updates) == 0 {
		res.Code = codes.CODE_SUCCESS
		res.Msg = "Invalid request:id not exist"
		res.Data = inDb
		c.JSON(http.StatusOK, res)
		return
	}
	updates["update_time"] = time.Now()
	err = up.Updates(updates).Error
	if err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:id not exist"
		res.Data = inDb
		c.JSON(http.StatusOK, res)
		return
	}
	inDb = model.MemeVault{}
	_ = db.Model(&model.MemeVault{}).Where("id = ?", req.ID).Take(&model.MemeVault{}).Error
	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = inDb
	c.JSON(http.StatusOK, res)
}
func MemeVaultAdd(c *gin.Context) {
	var req model.MemeVault
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:parameterFormatError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("MemeVaultAdd req: %+v", req)
	if len(req.UUID) < 1 {
		res.Msg = "Invalid GetMemeVaultList:uuidEmptyError:"
	}
	if len(req.UserType) < 1 {
		res.Msg = "Invalid GetMemeVaultList:UserTypeEmptyError:"
	}
	if len(res.Msg) > 0 {
		res.Code = codes.CODE_ERR
		c.JSON(http.StatusOK, res)
		return
	}
	if len(req.ChainIndex) < 1 {
		req.ChainIndex = "11111111111111111111"
	}
	if req.MinAmount.LessThan(decimal.NewFromFloat(1.0)) {
		req.MinAmount = decimal.NewFromFloat(1.0)
	}
	if req.MaxAmount.LessThanOrEqual(req.MinAmount) {
		req.MaxAmount = decimal.Max(req.MinAmount, decimal.NewFromFloat(5.0))
	}
	if req.StartTime.Before(time.Now().AddDate(0, 0, -1)) {
		req.StartTime = time.Now()
	}
	if req.ExpireTime.Before(time.Now().AddDate(0, 0, -1)) {
		req.ExpireTime = time.Now().AddDate(0, 0, 1)
	}
	if req.VaultType < 1 {
		req.VaultType = 1
	}

	req.Status = 1
	req.CreateTime = time.Now()
	req.UpdateTime = time.Now()
	db := system.GetDb()
	var indb []model.MemeVault
	_ = db.Model(&model.MemeVault{}).Where("uuid = ? and vault_type = ?", req.UUID, req.VaultType).Find(&indb).Error
	if len(indb) > 0 {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid MemeVaultAdd:uuid-vault  ExistError:"
		c.JSON(http.StatusOK, res)
		return
	}
	err := db.Create(&req).Error
	if err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:createError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	var memeV model.MemeVault
	db.Model(&model.MemeVault{}).Where("id = ?", req.ID).Take(&memeV)
	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = memeV
	c.JSON(http.StatusOK, res)
}

func MemeVaultList(c *gin.Context) {
	var req MemeVaultListReq
	res := common.Response{}
	before5Years := time.Now().AddDate(-5, 0, 0)
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:parameterFormatError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	mylog.Infof("MemeVaultList req: %+v", req)

	db := system.GetDb()
	query := db.Model(&model.MemeVault{})
	if len(req.Uuid) > 0 {
		query = query.Where("uuid = ?", req.Uuid)
	}
	if req.VaultType > 0 {
		query = query.Where("vault_type = ?", req.VaultType)
	}
	if len(req.UserType) > 0 {
		query = query.Where("user_type = ?", req.UserType)
	}
	// 大于传递的过期时间
	if req.ExpireTime != nil && req.ExpireTime.After(before5Years) {
		query = query.Where("expire_time >= ?", req.ExpireTime)
	}
	//大于传递的创建时间
	if req.CreateTime != nil && req.CreateTime.After(before5Years) {
		query = query.Where("create_time >= ?", req.CreateTime)
	}
	//小于传递的过期时间
	if req.ExpireTimeBefore != nil && req.ExpireTimeBefore.After(before5Years) {
		query = query.Where("expire_time <= ?", req.ExpireTimeBefore)
	}
	//小于传递的创建时间
	if req.CreateTimeBefore != nil && req.CreateTimeBefore.After(before5Years) {
		query = query.Where("create_time <= ?", req.CreateTimeBefore)
	}
	// 计算分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20 // 默认每页20条
	}
	offset := (req.Page - 1) * req.PageSize
	var memes []model.MemeVault
	err := query.Order("ID").Limit(req.PageSize).Offset(offset).Find(&memes).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:queryError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	PaginatedResult := common.PaginatedResult[model.MemeVault]{
		Page:     req.Page,
		PageSize: req.PageSize,
		Data:     memes}
	res.Data = PaginatedResult
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
}
func ClaimToMemeVault(c *gin.Context) {
	var req ClaimToMemeVaultReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid ClaimToMemeVault:parameterFormatError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	mylog.Info("ClaimToMemeVault req: ", req)
	if req.WalletID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "bad request parameters:  WalletID is empty"
		c.JSON(http.StatusOK, res)
		return
	}
	db := system.GetDb()
	// 查询用户基金钱包
	var tWg model.WalletGenerated
	err := db.Model(&model.WalletGenerated{}).Where("id = ? and status = ?", req.WalletID, "00").First(&tWg).Error
	if err != nil || tWg.ID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "bad request : ToWallet is nil " + strconv.FormatInt(req.WalletID, 10)
		c.JSON(http.StatusOK, res)
		return
	}
	chainCode := tWg.ChainCode
	// 查询hello的基金钱包
	var fWg model.WalletGenerated
	err = db.Model(&model.WalletGenerated{}).Where("group_id = ? and chain_code=? status = ?", config.GetConfig().MemeVaultFrom, chainCode, "00").First(&fWg).Error
	if err != nil || fWg.ID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = "bad request : fromWallet is nil " + strconv.FormatInt(int64(config.GetConfig().MemeVaultFrom), 10)
		c.JSON(http.StatusOK, res)
		return
	}
	// 查询用户的基金钱包组
	var group model.WalletGroup
	err = db.Model(&model.WalletGroup{}).Where("id = ? ", tWg.GroupID).First(&group).Error
	if err != nil || group.ID < 1 || group.VaultType != 1 {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("bad request : ToWallet is not meme vault group %+v", group)
		c.JSON(http.StatusOK, res)
		return
	}
	//查询用户的基金钱包配置
	var vault model.MemeVault
	err = db.Model(&model.MemeVault{}).Where("uuid = ? and vault_type =?   order by id",
		tWg.UserID, group.VaultType).Take(&vault).Error
	if err != nil || vault.ID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("bad request :MemeVault is nil")
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验过期时间
	if vault.ExpireTime.Before(time.Now()) {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("MemeVault is expired")
		c.JSON(http.StatusOK, res)
		return
	}
	tokenDecimals := 18
	if chainCode == "SOLANA" {
		tokenDecimals = 9
	}
	priceStr := wallet.QuotePrice(chainCode, "So11111111111111111111111111111111111111112")
	if priceStr == "" {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("get price error")
		c.JSON(http.StatusOK, res)
		return
	}
	price, err := decimal.NewFromString(priceStr)

	rand.Seed(time.Now().UnixNano())
	// 生成0到1之间的随机小数
	randomFloat := rand.Float64()

	// 将这个随机小数缩放到1到5之间
	amountUsd := decimal.NewFromFloat(1 + randomFloat*(5-1))
	amountD := amountUsd.Div(price).Round(int32(tokenDecimals))
	amount := amountD.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(tokenDecimals))))
	chainConfig := config.GetRpcConfig(fWg.ChainCode)

	txhash, err := chain.HandleTransfer(chainConfig, tWg.Wallet, "", amount.BigInt(), &fWg, &req.Config)
	if err != nil {
		mylog.Error("transfer error:", req, err)
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("unknown error %s", err.Error())
		c.JSON(http.StatusOK, res)
		return
	}
	token := ""
	memeV := &model.MemeVaultSupport{
		UUID:           tWg.UserID,
		GroupId:        tWg.GroupID,
		WalletID:       tWg.ID,
		Wallet:         tWg.Wallet,
		FromWallet:     fWg.Wallet,
		FromWalletID:   fWg.ID,
		ChainCode:      fWg.ChainCode,
		VaultType:      group.VaultType,
		Status:         201,
		SupportAddress: token,
		SupportAmount:  amountD,
		Price:          price,
		Channel:        req.Channel,
		Tx:             txhash,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}
	err = db.Model(&model.MemeVaultSupport{}).Save(memeV).Error
	if err != nil {
		mylog.Error("save log error ", err)
	}
	reqdata, _ := json.Marshal(req)

	wl := &model.WalletLog{
		WalletID:  int64(fWg.ID),
		Wallet:    tWg.Wallet,
		Data:      string(reqdata),
		ChainCode: fWg.ChainCode,
		Operation: "claimToMemeVault",
		OpTime:    time.Now(),
		TxHash:    txhash,
	}

	err = db.Model(&model.WalletLog{}).Save(wl).Error
	if err != nil {
		mylog.Error("save log error ", err)
	}

	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = struct {
		Wallet string `json:"wallet"`
		Tx     string `json:"tx"`
	}{
		Wallet: fWg.Wallet,
		Tx:     txhash,
	}
	c.JSON(http.StatusOK, res)
}
func GetMemeVaultWallet(db *gorm.DB, req *common.UserStructReq, channel any) []common.AuthGetBackWallet {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)      // 分配缓冲区
			n := runtime.Stack(buf, false) // false 表示只当前 goroutine
			mylog.Errorf("GetMemeVaultWallet Recovered: %v Stack trace: \n %s ", r, buf[:n])
		}
	}()

	walletGroupsValidChains := make(map[uint64]string)
	//获取用户所有的基金钱包组
	resultList := make([]common.AuthGetBackWallet, 0)
	var walletGroups []model.WalletGroup
	err := db.Model(&model.WalletGroup{}).Where("user_id = ? and vault_type >0", req.Uuid).Find(&walletGroups).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	// 冲狗基金标识
	flagMeMeVault := false

	for _, group := range walletGroups {
		if group.VaultType == 1 {
			flagMeMeVault = true
		}
		// 查询钱包组对应基金的chainCode
		var memeV model.MemeVault
		err := db.Model(&model.MemeVault{}).Where("uuid = ? and vault_type = ?", req.Uuid, group.VaultType).Take(&memeV).Error
		if err != nil {
			walletGroupsValidChains[group.ID] = wallet.GetAllCodesByIndex()
		}
		if memeV.ID > 0 {
			if memeV.GroupId == 0 {
				//映射钱包组和基金关系
				err = db.Model(&model.MemeVault{}).Where("id = ?  ", memeV.ID).
					Updates(map[string]interface{}{
						"group_id": group.ID}).Error
			}
			walletGroupsValidChains[group.ID] = memeV.ChainIndex
		}
	}
	//用户默认的冲狗基金
	if !flagMeMeVault {
		// 钱包组未创建
		strmneno, err := enc.NewKeyStories()
		if err == nil {
			group := &model.WalletGroup{
				UserID:         req.Uuid,
				CreateTime:     time.Now(),
				EncryptMem:     strmneno,
				EncryptVersion: fmt.Sprintf("AES:%d", 1),
				Nonce:          int(enc.Porter().GetNonce()),
				VaultType:      1,
			}
			// 创建钱包组
			err = db.Save(group).Error
			if err != nil {
				mylog.Errorf("GetMemeVaultWallet 保存钱包组失败 req:%+v, Vault:%d,err:%+v", req, 1, err)
				return nil
			}
			walletGroups = append(walletGroups, *group)
			var memeV model.MemeVault
			err := db.Model(&model.MemeVault{}).Where("uuid = ? and vault_type = ?", req.Uuid, group.VaultType).Take(&memeV).Error
			if err != nil || memeV.ID < 1 {
				walletGroupsValidChains[group.ID] = wallet.GetAllCodesByIndex()
			}
			if memeV.ID > 0 {
				if memeV.GroupId == 0 {
					//映射钱包组和基金关系
					err = db.Model(&model.MemeVault{}).Where("id = ?  ", memeV.ID).
						Updates(map[string]interface{}{
							"group_id": group.ID}).Error
				}
				walletGroupsValidChains[group.ID] = memeV.ChainIndex
			}
		}
	}

	// 获取钱包 一个钱包组实例 对应一组钱包钱包实例
	for _, g := range walletGroups {
		chainIndexs, ex := walletGroupsValidChains[g.ID]
		if !ex {
			chainIndexs = "111111111111111111111"
		}
		//拿取对应链
		validChains := wallet.CheckAllCodesByIndex(chainIndexs)
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
		mylog.Info("GetMemeVaultWallet need create: ", needCreates)
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
		// 需要创建的对应chaincode钱包
		for _, v := range needCreates {
			wal, err := wallet.Generate(&g, wallet.ChainCode(v))
			if err != nil {
				mylog.Errorf("GetMemeVaultWallet create wallet error %v", err)
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
				mylog.Errorf("GetMemeVaultWallet create wallet error %v", err)
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
	return resultList
}

func CheckMemeVaultWalletTransfer(db *gorm.DB, req TokenTransferReq, fromWallet model.WalletGenerated) bool {
	to := req.To
	// 校验 冲狗基金 其他账户不允许转账给冲狗基金
	var vWg model.WalletGenerated
	err := db.Model(&model.WalletGenerated{}).Where("wallet = ? ", to).First(&vWg).Error
	if err == nil && vWg.ID > 0 {
		gId := vWg.GroupID
		var group model.WalletGroup
		err = db.Model(&model.WalletGroup{}).Where("id = ? ", gId).First(&group).Error
		if err == nil && group.ID > 0 && group.VaultType == 1 {
			return false
		}
	}
	return true
}
func IsMemeVaultWalletTrade(db *gorm.DB, walletId int64, toWallet *model.WalletGenerated) bool {
	if toWallet.GroupID > 0 {
		var group model.WalletGroup
		err := db.Model(&model.WalletGroup{}).Where("id = ? ", toWallet.GroupID).First(&group).Error
		if err == nil && group.ID > 0 && group.VaultType == 1 {
			return true
		}
	} else if walletId > 0 {
		var vWg model.WalletGenerated
		err := db.Model(&model.WalletGenerated{}).Where("id = ? ", walletId).First(&vWg).Error
		if err == nil && vWg.ID > 0 {
			gId := vWg.GroupID
			var group model.WalletGroup
			err = db.Model(&model.WalletGroup{}).Where("id = ? ", gId).First(&group).Error
			if err == nil && group.ID > 0 && group.VaultType == 1 {
				return true
			}
		}

	}

	return false
}
