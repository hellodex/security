package controller

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/codes"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/system"
	"github.com/hellodex/HelloSecurity/wallet"
	"github.com/hellodex/HelloSecurity/wallet/enc"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"net/http"
	"runtime"
	"time"
)

type MemeVaultListReq struct {
	Uuid             string     `json:"uuid"`
	VaultType        int        `json:"vault_type"`
	UserType         string     `json:"user_type"`
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
	GroupId          uint64          `json:"group_id"`
	WalletID         int64           `json:"wallet_id"`
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
	req.VaultType = 1
	req.Status = 1
	req.CreateTime = time.Now()
	req.UpdateTime = time.Now()
	db := system.GetDb()
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

func GetMemeVault(db *gorm.DB, req *common.UserStructReq, channel any) []common.AuthGetBackWallet {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)      // 分配缓冲区
			n := runtime.Stack(buf, false) // false 表示只当前 goroutine
			mylog.Errorf("GetMemeVault Recovered: %v Stack trace: \n %s ", r, buf[:n])
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
				mylog.Errorf("GetMemeVault 保存钱包组失败 req:%+v, Vault:%d,err:%+v", req, 1, err)
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
		mylog.Info("GetMemeVault need create: ", needCreates)
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
				mylog.Errorf("GetMemeVault create wallet error %v", err)
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
				mylog.Errorf("GetMemeVault create wallet error %v", err)
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
