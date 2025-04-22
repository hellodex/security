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
	"strings"
	"sync"
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
type MemeVaultVo struct {
	ID       uint64 `json:"id"`
	UUID     string `json:"uuid"`
	UserType string `json:"userType"`
	//GroupId      uint64             `gorm:"column:group_id" json:"groupId"`
	ChainIndex   string               `json:"chainIndex"`
	VaultType    int                  `json:"vaultType"`
	Status       int                  `json:"status"` // 0:正常/未失效  1 注销 2 冻结
	MaxAmount    decimal.Decimal      `json:"maxAmount"`
	MinAmount    decimal.Decimal      `json:"minAmount"`
	StartTime    string               `json:"startTime"`
	ExpireTime   string               `json:"expireTime"`
	CreateTime   string               `json:"createTime"`
	UpdateTime   string               `json:"updateTime"`
	VaultSupport []MemeVaultSupportVo `json:"vaultSupport"`
}
type MemeVaultSupportVo struct {
	ID             uint64          `json:"id"`
	UUID           string          `json:"uuid"`
	GroupId        uint64          `json:"groupId"`
	WalletID       uint64          `json:"walletId"`
	Wallet         string          `json:"wallet"`
	FromWallet     string          `json:"fromWallet"`
	FromWalletID   uint64          `json:"fromWalletID"`
	ChainCode      string          `json:"chainCode"`
	VaultType      int             `json:"vaultType"`
	Status         int             `json:"status"` // 0:成功 1:失败
	SupportAddress string          `json:"supportAddress"`
	SupportAmount  decimal.Decimal `json:"supportAmount"`
	Price          decimal.Decimal `json:"price"`
	Channel        string          `json:"channel"`
	Tx             string          `json:"tx"`
	CreateTime     string          `json:"createTime"`
	UpdateTime     string          `json:"updateTime"`
	Usd            decimal.Decimal `json:"usd"`
}

type MemeVaultUpdateReq struct {
	ID       uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID     string `gorm:"column:uuid" json:"uuid"`
	UserType string `gorm:"column:user_type" json:"userType"`
	//GroupId      uint64             `gorm:"column:group_id" json:"groupId"`
	ChainIndex string          `gorm:"column:chain_index" json:"chainIndex"`
	VaultType  int             `gorm:"column:vault_type" json:"vaultType"`
	Status     int             `gorm:"column:status" json:"status"` // 0:正常/未失效  1 注销 2 冻结
	MaxAmount  decimal.Decimal `gorm:"column:max_amount" json:"maxAmount"`
	MinAmount  decimal.Decimal `gorm:"column:min_amount" json:"minAmount"`
	StartTime  string          `gorm:"column:start_time" json:"startTime"`
	ExpireTime string          `gorm:"column:expire_time" json:"expireTime"`
	Admin      string          `gorm:"-" json:"admin"`
	TwoFACode  string          `gorm:"-" json:"twoFACode"`
}

func VaultSupportList(c *gin.Context) {
	var req MemeVaultSupportListReq
	res := common.Response{}
	before5Years := time.Now().AddDate(-5, 0, 0)
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid VaultSupportList:parameterFormatError:" + err.Error()
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
	// Get total count of matching records
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		totalCount = 100
		mylog.Error("VaultSupportList:queryCountError:" + err.Error())
	}
	err := query.Order("ID DESC").Limit(req.PageSize).Offset(offset).Find(&memes).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid VaultSupportList:queryError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	var memeVos []MemeVaultSupportVo
	memeVos = memeVaultSupportToVo(memes...)
	PaginatedResult := common.PaginatedResult[MemeVaultSupportVo]{
		Current: req.Page,
		Total:   int(totalCount),
		Size:    req.PageSize,
		Records: memeVos}
	res.Data = PaginatedResult
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)

}
func MemeVaultSupportListByUUID(c *gin.Context) {
	var req MemeVaultListReq
	res := common.Response{}
	if err := c.ShouldBindJSON(&req); err != nil {
		mylog.Errorf("MemeVaultSupport req format error:%s", err)
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
	resV := memeVaultToVo(vault)
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = resV[0]
	c.JSON(http.StatusOK, res)
	return
}
func MemeVaultUpdate(c *gin.Context) {
	var req MemeVaultUpdateReq
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

	reqStr, _ := json.Marshal(req)
	if ok, errV := Verify2fa(req.Admin, req.TwoFACode, "MemeVaultUpdate "+string(reqStr)); !ok {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("2fa verify failed,err:%v", errV)
		c.JSON(http.StatusOK, res)
		mylog.Infof("冲狗基金-修改TOTP验证未通过")
		return
	}

	err = updateMemeVault(db, req, inDb)

	if err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid request:updateError:" + err.Error()
		res.Data = inDb
		c.JSON(http.StatusOK, res)
		return
	}
	inDb = model.MemeVault{}
	_ = db.Model(&model.MemeVault{}).Where("id = ?", req.ID).Take(&model.MemeVault{}).Error
	resV := memeVaultToVo(inDb)
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	res.Data = resV[0]
	c.JSON(http.StatusOK, res)
}

func updateMemeVault(db *gorm.DB, req MemeVaultUpdateReq, inDb model.MemeVault) error {
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
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	if len(req.StartTime) > 0 && req.StartTime != "0" {
		timeInt, err := strconv.ParseInt(req.StartTime, 10, 64)
		if err == nil {
			t := time.UnixMilli(timeInt)
			startTime = t
		}
	}
	if len(req.ExpireTime) > 0 && req.ExpireTime != "0" {
		timeInt, err := strconv.ParseInt(req.ExpireTime, 10, 64)
		if err == nil {
			t := time.UnixMilli(timeInt)
			endTime = t
		}
	}
	if startTime.After(time.Now().Add(-1*time.Hour)) && !startTime.Equal(inDb.StartTime) {
		updates["start_time"] = startTime
	}
	if endTime.After(time.Now().Add(1*time.Hour)) && !endTime.Equal(inDb.ExpireTime) {
		updates["expire_time"] = endTime
	}
	if len(updates) == 0 {
		//res.Code = codes.CODE_SUCCESS
		//res.Msg = "Invalid request:id not exist"
		//res.Records = inDb
		//c.JSON(http.StatusOK, res)
		return nil
	}
	updates["update_time"] = time.Now()
	err := up.Updates(updates).Error
	return err
}
func MemeVaultAdd(c *gin.Context) {
	var req MemeVaultUpdateReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:parameterFormatError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}

	mylog.Infof("MemeVaultAdd req: %+v", req)
	reqStr, _ := json.Marshal(req)
	if ok, errV := Verify2fa(req.Admin, req.TwoFACode, "MemeVaultAdd "+string(reqStr)); !ok {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("2fa verify failed,err:%v", errV)
		mylog.Infof("冲狗基金-添加TOTP验证未通过")
		c.JSON(http.StatusOK, res)
		return
	}
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
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	if len(req.StartTime) > 0 && req.StartTime != "0" {
		timeInt, err := strconv.ParseInt(req.StartTime, 10, 64)
		if err == nil {
			startTime = time.UnixMilli(timeInt)

		}
	}
	if len(req.ExpireTime) > 0 && req.ExpireTime != "0" {
		timeInt, err := strconv.ParseInt(req.ExpireTime, 10, 64)
		if err == nil {
			endTime = time.UnixMilli(timeInt)
		}
	}

	var indb []model.MemeVault
	db := system.GetDb()
	_ = db.Model(&model.MemeVault{}).Where("uuid = ? and vault_type = ?", req.UUID, req.VaultType).Find(&indb).Error
	if len(indb) > 0 {
		// 已有记录 则保存过期时间最长
		if endTime.After(time.Now()) &&
			!endTime.Equal(indb[0].ExpireTime) &&
			endTime.Before(indb[0].ExpireTime) {
			req.ExpireTime = strconv.FormatInt(indb[0].ExpireTime.UnixMilli(), 10)
		}
		err := updateMemeVault(db, req, indb[0])
		if err != nil {
			res.Code = codes.CODE_ERR
			res.Msg = "已存在MemeVault,更新失败:" + err.Error()
			c.JSON(http.StatusOK, res)
			return
		}
		res.Code = codes.CODE_ERR
		res.Msg = "已存在MemeVault,更新成功"
		c.JSON(http.StatusOK, res)
		return
	}
	if startTime.Before(time.Now()) {
		startTime = time.Now()
	}
	if endTime.Before(time.Now().Add(1 * time.Hour)) {
		endTime = time.Now().AddDate(0, 0, 1)
	}
	if req.VaultType < 1 {
		req.VaultType = 1
	}

	req.Status = 1
	meme := &model.MemeVault{
		UUID:       req.UUID,
		VaultType:  req.VaultType,
		UserType:   req.UserType,
		ChainIndex: req.ChainIndex,
		MinAmount:  req.MinAmount,
		MaxAmount:  req.MaxAmount,
		StartTime:  startTime,
		ExpireTime: endTime,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
		Status:     0,
	}

	err := db.Create(meme).Error
	if err != nil {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:createError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	var memeV model.MemeVault
	db.Model(&model.MemeVault{}).Where("id = ?", meme.ID).Take(&memeV)
	resV := memeVaultToVo(memeV)
	res.Code = codes.CODE_SUCCESS
	res.Msg = "success"
	res.Data = resV[0]
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
	var totalCount int64
	if err := query.Debug().Count(&totalCount).Error; err != nil {
		totalCount = 0
		mylog.Error("MemeVaultList:queryCountError:" + err.Error())
	}
	err := query.Debug().Order("ID").Limit(req.PageSize).Offset(offset).Find(&memes).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR
		res.Msg = "Invalid GetMemeVaultList:queryError:" + err.Error()
		c.JSON(http.StatusOK, res)
		return
	}
	var memeVos1 []MemeVaultVo
	memeVos1 = memeVaultToVo(memes...)
	PaginatedResult := common.PaginatedResult[MemeVaultVo]{
		Current: req.Page,
		Size:    req.PageSize,
		Total:   int(totalCount),
		Records: memeVos1}
	res.Data = PaginatedResult
	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "success"
	c.JSON(http.StatusOK, res)
}

var ClaimLook = sync.Mutex{}

func ClaimToMemeVault(c *gin.Context) {
	var req ClaimToMemeVaultReq
	res := common.Response{}
	res.Timestamp = time.Now().Unix()
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Code = codes.CODE_ERR
		mylog.Error("Invalid ClaimToMemeVault:parameterFormatError:" + err.Error())
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	walletIdStr := strconv.FormatInt(req.WalletID, 10)
	lock := common.GetLock("ClaimToMemeVault+" + walletIdStr)
	if lock.Lock.TryLock() {
		defer lock.Lock.Unlock()
	} else {
		res.Code = codes.CODE_ERR
		res.Msg = "操作太频繁,请稍后再试"
		c.JSON(http.StatusOK, res)
		return
	}
	if common.RateLimiterMoreThan("ClaimToMemeVault"+walletIdStr, 1, 3*time.Second) {
		res.Code = codes.CODE_ERR
		res.Msg = "操作太频繁,请稍后再试"
		c.JSON(http.StatusOK, res)
		return
	}
	mylog.Info("ClaimToMemeVault req: ", req)
	if req.WalletID < 1 {
		res.Code = codes.CODE_ERR
		mylog.Error("ClaimToMemeVault bad request parameters:  WalletID is empty")
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	db := system.GetDb()
	// 查询用户基金钱包
	var tWg model.WalletGenerated
	err := db.Model(&model.WalletGenerated{}).Where("id = ? and status = ?", req.WalletID, "00").First(&tWg).Error
	if err != nil || tWg.ID < 1 {
		res.Code = codes.CODE_ERR
		mylog.Error("bad request : ToWallet is nil " + strconv.FormatInt(req.WalletID, 10))
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	chainCode := tWg.ChainCode
	if chainCode != "SOLANA" {
		res.Code = codes.CODE_ERR
		mylog.Error("bad request : ToWallet is not solana " + chainCode)
		res.Msg = "请切换至SOL链钱包领取，其他链很快开放"
		c.JSON(http.StatusOK, res)
		return
	}
	// 查询hello的基金钱包
	var fWg model.WalletGenerated
	err = db.Model(&model.WalletGenerated{}).Where("group_id = ? and chain_code=? and status = ?", config.GetConfig().MemeVaultFrom, chainCode, "00").First(&fWg).Error
	if err != nil || fWg.ID < 1 {
		res.Code = codes.CODE_ERR
		mylog.Error("bad request : fromWallet is nil " + strconv.FormatInt(int64(config.GetConfig().MemeVaultFrom), 10))
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	// 查询用户的基金钱包组
	var group model.WalletGroup
	err = db.Model(&model.WalletGroup{}).Where("id = ? ", tWg.GroupID).First(&group).Error
	if err != nil || group.ID < 1 || group.VaultType != 1 {
		res.Code = codes.CODE_ERR
		mylog.Error(fmt.Sprintf("bad request : ToWallet is not meme vault group %+v", group))
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	//查询用户的基金钱包配置
	var vault model.MemeVault
	err = db.Model(&model.MemeVault{}).Where("uuid = ? and vault_type =?   order by id",
		tWg.UserID, group.VaultType).Take(&vault).Error
	if err != nil || vault.ID < 1 {
		res.Code = codes.CODE_ERR
		res.Msg = fmt.Sprintf("暂无冲狗基金资格，请参与IDO或去中文TG群")
		mylog.Errorf(fmt.Sprintf("ClaimToMemeVault bad request :MemeVault is nil"))
		c.JSON(http.StatusOK, res)
		return
	}
	// 校验过期时间
	if vault.ExpireTime.Before(time.Now()) {
		res.Code = codes.CODE_ERR

		mylog.Error("ClaimToMemeVault MemeVault is expired")
		res.Msg = "领取失败,冲狗基金已过期"
		c.JSON(http.StatusOK, res)
		return
	}
	var vaultSp []model.MemeVaultSupport
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	err = db.Model(&model.MemeVaultSupport{}).Where("uuid = ? and vault_type =? and create_time >? and status >=200 and status <205  ",
		tWg.UserID, group.VaultType, startOfDay).Find(&vaultSp).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		res.Code = codes.CODE_ERR
		mylog.Error(fmt.Sprintf("ClaimToMemeVault bad request :MemeVaultSupport is nil"))
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	if len(vaultSp) > 0 {
		res.Code = codes.CODE_ERR
		res.Msg = "领取失败,今日已领取过"
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
		mylog.Error("ClaimToMemeVault get price error")
		res.Msg = "领取失败,请联系客服"
		c.JSON(http.StatusOK, res)
		return
	}
	price, err := decimal.NewFromString(priceStr)

	rand.Seed(time.Now().UnixNano())
	// 生成0到1之间的随机小数
	randomFloat := rand.Float64()

	// 将这个随机小数缩放到1到5之间
	amountUsd := vault.MinAmount.Add(decimal.NewFromFloat(randomFloat).Mul(vault.MaxAmount.Sub(vault.MinAmount)))
	amountD := amountUsd.Div(price).Round(int32(tokenDecimals))
	amount := amountD.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(tokenDecimals))))
	chainConfig := config.GetRpcConfig(fWg.ChainCode)

	maxRetries := 3
	txHash := ""
	req.Config.ShouldConfirm = true
	req.Config.ConfirmTimeOut = 20
	mylog.Infof("ClaimToMemeVault  req :%+v,[%s-%s],price:%s,random:%s,amount:%s->%s ", req, vault.MinAmount.String(), vault.MaxAmount.String(), priceStr, amountUsd.String(), amountD.String(), amount.String())
	retries := 0
	for range maxRetries {
		retries++
		txHash, err = chain.HandleTransfer(chainConfig, tWg.Wallet, "", amount.BigInt(), &fWg, &req.Config)
		if err != nil {
			mylog.Errorf("ClaimToMemeVault transfer  error: %s,%+v,%v", txHash, req, err)
		}
		if txHash != "" && !strings.HasPrefix(txHash, "11111111111111") && err == nil {

			break
		}
	}
	mylog.Infof("ClaimToMemeVault [%d]  req :%+v, amount:%s ,err:%v", retries, req, amount.String(), err)
	reqdata, _ := json.Marshal(req)

	wl := &model.WalletLog{
		WalletID:  int64(fWg.ID),
		Wallet:    tWg.Wallet,
		Data:      string(reqdata),
		ChainCode: fWg.ChainCode,
		Operation: "claimToMemeVault",
		OpTime:    time.Now(),
		TxHash:    txHash,
	}

	_ = db.Model(&model.WalletLog{}).Save(wl).Error

	if err != nil && !strings.HasPrefix(txHash, "11111111111111") {
		res.Code = codes.CODE_ERR
		mylog.Error("ClaimToMemeVault transfer error:", req, err)
		res.Msg = "领取失败,请联系客服"
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
		Status:         200,
		SupportAddress: token,
		SupportAmount:  amountD,
		Price:          price,
		Channel:        req.Channel,
		Tx:             txHash,
		Usd:            amountUsd,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}
	mylog.Infof("ClaimToMemeVault MemeVaultSupport %s ", memeV.String())
	err = db.Model(&model.MemeVaultSupport{}).Save(memeV).Error
	mylog.Infof("ClaimToMemeVault MemeVaultSupport %s ", memeV.String())
	if err != nil {
		mylog.Error("save log error ", err)
	}

	res.Code = codes.CODE_SUCCESS_200
	res.Msg = "已领取 " + amountD.Round(3).String() + "U等值的SOL，请稍后在钱包查询"
	go func() {
		//tasks.HandleTx(*memeV)
	}()
	res.Data = struct {
		Wallet string `json:"wallet"`
		Tx     string `json:"tx"`
	}{
		Wallet: fWg.Wallet,
		Tx:     txHash,
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
			//if memeV.GroupId == 0 {
			//	//映射钱包组和基金关系
			//	err = db.Model(&model.MemeVault{}).Where("id = ?  ", memeV.ID).
			//		Updates(map[string]interface{}{
			//			"group_id": group.ID}).Error
			//}
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
				//if memeV.GroupId == 0 {
				//	//映射钱包组和基金关系
				//	err = db.Model(&model.MemeVault{}).Where("id = ?  ", memeV.ID).
				//		Updates(map[string]interface{}{
				//			"group_id": group.ID}).Error
				//}
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
	if fromWallet.GroupID > 0 {
		var group model.WalletGroup
		err = db.Model(&model.WalletGroup{}).Where("id = ? ", fromWallet.GroupID).First(&group).Error
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
func memeVaultSupportToVo(supports ...model.MemeVaultSupport) []MemeVaultSupportVo {
	if len(supports) < 1 {
		return []MemeVaultSupportVo{}
	}
	mvo := make([]MemeVaultSupportVo, 0)
	for _, support := range supports {
		creatTime := ""
		updateTime := ""

		if !support.CreateTime.Before(time.Now().AddDate(-10, 0, 0)) {
			creatTime = support.CreateTime.Format("2006-01-02 15:04:05")
		}
		if !support.UpdateTime.Before(time.Now().AddDate(-10, 0, 0)) {
			updateTime = support.UpdateTime.Format("2006-01-02 15:04:05")
		}
		mvo = append(mvo, MemeVaultSupportVo{
			ID:             support.ID,
			UUID:           support.UUID,
			GroupId:        support.GroupId,
			WalletID:       support.WalletID,
			Wallet:         support.Wallet,
			FromWallet:     support.FromWallet,
			FromWalletID:   support.FromWalletID,
			ChainCode:      support.ChainCode,
			VaultType:      support.VaultType,
			Status:         support.Status,
			SupportAddress: support.SupportAddress,
			SupportAmount:  support.SupportAmount,
			Price:          support.Price,
			Channel:        support.Channel,
			Tx:             support.Tx,
			Usd:            support.Usd,
			CreateTime:     creatTime,
			UpdateTime:     updateTime,
		})

	}
	return mvo
}
func memeVaultToVo(supports ...model.MemeVault) []MemeVaultVo {
	if len(supports) < 1 {
		return []MemeVaultVo{}
	}
	mv := make([]MemeVaultVo, 0)
	for _, support := range supports {
		startTime := ""
		expireTime := ""
		createTime := ""
		updateTime := ""

		if !support.StartTime.Before(time.Now().AddDate(-10, 0, 0)) {
			startTime = support.StartTime.Format("2006-01-02 15:04:05")
		}
		if !support.ExpireTime.Before(time.Now().AddDate(-10, 0, 0)) {
			expireTime = support.ExpireTime.Format("2006-01-02 15:04:05")
		}
		if !support.CreateTime.Before(time.Now().AddDate(-10, 0, 0)) {
			createTime = support.CreateTime.Format("2006-01-02 15:04:05")
		}
		if !support.UpdateTime.Before(time.Now().AddDate(-10, 0, 0)) {
			updateTime = support.UpdateTime.Format("2006-01-02 15:04:05")
		}
		mvo := make([]MemeVaultSupportVo, 0)
		if len(support.VaultSupport) > 0 {
			mvo = memeVaultSupportToVo(support.VaultSupport...)
		} else {
			mvo = []MemeVaultSupportVo{}
		}

		mv = append(mv, MemeVaultVo{
			ID:           support.ID,
			UUID:         support.UUID,
			UserType:     support.UserType,
			ChainIndex:   support.ChainIndex,
			VaultType:    support.VaultType,
			Status:       support.Status,
			MaxAmount:    support.MaxAmount,
			MinAmount:    support.MinAmount,
			StartTime:    startTime,
			ExpireTime:   expireTime,
			CreateTime:   createTime,
			UpdateTime:   updateTime,
			VaultSupport: mvo,
		})

	}
	return mv
}
