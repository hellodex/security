package store

import (
	"github.com/hellodex/HelloSecurity/api/common"
	mylog "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
)

func IdoLogList(info model.IdoLog) ([]model.IdoLog, error) {
	query := db.Model(&model.IdoLog{})
	if len(info.ChainCode) > 0 {
		query = query.Where("chain_code = ?", info.ChainCode)
	}
	if len(info.Wallet) > 0 {
		query = query.Where("wallet = ?", info.Wallet)
	}
	if len(info.Token) > 0 {
		query = query.Where("token = ?", info.Token)
	}
	if info.Amount.Sign() > 0 {
		query = query.Where("amount >=   ?", info.Amount)
	}
	if len(info.Tx) > 0 {
		query = query.Where("tx = ?", info.Tx)
	}
	var infos []model.IdoLog
	err := query.Order("id desc").Limit(1000).Find(&infos).Error
	if err != nil {
		return nil, err
	}
	return infos, nil
}
func IdoLogAdd(info *model.IdoLog) (*model.IdoLog, error) {
	err := db.Model(&model.IdoLog{}).Create(info).Error
	if err != nil {
		return nil, err
	}
	return info, nil
}

func IdoLogPage(info model.IdoLog, page int, pageSize int) (*common.PaginatedResult[model.IdoLog], error) {
	query := db.Model(&model.IdoLog{})
	if len(info.ChainCode) > 0 {
		query = query.Where("chain_code = ?", info.ChainCode)
	}
	if len(info.Wallet) > 0 {
		query = query.Where("wallet = ?", info.Wallet)
	}
	if len(info.Token) > 0 {
		query = query.Where("token = ?", info.Token)
	}
	if info.Amount.Sign() > 0 {
		query = query.Where("amount >=   ?", info.Amount)
	}
	if len(info.Tx) > 0 {
		query = query.Where("tx = ?", info.Tx)
	}
	// 计算分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20 // 默认每页20条
	}
	offset := (page - 1) * pageSize
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		totalCount = 100
		mylog.Error("VaultSupportList:queryCountError:" + err.Error())
	}
	var infos []model.IdoLog
	err := query.Order("id desc").Limit(pageSize).Offset(offset).Find(&infos).Error
	if err != nil {
		return nil, err
	}
	PaginatedResult := common.PaginatedResult[model.IdoLog]{
		Current: page,
		Total:   int(totalCount),
		Size:    pageSize,
		Records: infos}
	return &PaginatedResult, nil
}
