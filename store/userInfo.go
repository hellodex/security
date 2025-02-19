package store

import (
	"github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
)

func UserInfoGetByAccountId(accountId string, accountType int) ([]model.AuthAccount, error) {
	var aa []model.AuthAccount
	err := db.Model(&model.AuthAccount{}).
		Where("account_id = ? and account_type =?", accountId, accountType).
		Find(&aa).Error
	if err != nil {
		log.Error("UserInfoGetByAccountId error: ", err)
		return nil, err
	}
	return aa, nil
}
func UserInfoGetByUUIDAndAccountTypeAndStatus(uuid string, accountType int) ([]model.AuthAccount, error) {
	var aa []model.AuthAccount
	err := db.Model(&model.AuthAccount{}).
		Where("user_uuid = ? AND account_type = ? AND status = 0", uuid, accountType).
		Find(&aa).Error
	if err != nil {
		log.Error("UserInfoGetByAccountId error: ", err)
		return nil, err
	}
	return aa, nil
}
func AuthAccountSave(u *model.AuthAccount) error {
	err := db.Save(u).Error
	if err != nil {
		log.Error("AuthAccountSave error: ", err)
		return err
	}
	return nil
}
func AuthAccountCancel(u *model.AuthAccount) error {
	err := db.Model(&model.AuthAccount{}).Where("user_uuid = ? AND account_id = ? and account_type = ?", u.UserUUID, u.AccountID, u.AccountType).
		Updates(map[string]interface{}{"status": 1}).Error
	return err
}
func UserInfoGetByInvitationCode(InvitationCode string) ([]model.UserInfo, error) {
	var aa []model.UserInfo
	err := db.Model(&model.UserInfo{}).
		Where("invitation_code = ?  ", InvitationCode).
		Find(&aa).Error
	if err != nil {
		log.Error("UserInfoGetByAccountId error: ", err)
		return nil, err
	}
	return aa, nil
}
func UserInfoSave(u *model.UserInfo) error {
	err := db.Save(u).Error
	if err != nil {
		log.Error("UserInfoSave error: ", err)
		return err
	}
	return nil
}
