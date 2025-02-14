package main

import (
	"crypto/hmac"
	md5 "crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/duke-git/lancet/v2/random"
	"github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	TestgenerateTGToken()
}
func TestgenerateTGToken() {
	for range 10 {
		s, _ := generateToken(model.TgLogin{TgUserId: "123456"})
		fmt.Println(fmt.Sprintf("%s", s))
	}

}
func TestMd5() {
	hash := md5.Sum([]byte("hellodex"))
	fmt.Println(fmt.Sprintf("%x", hash))
}
func TestRandomString() {
	for range 10 {
		log.Printf(common.RandomStr(8))
	}
}
func TestSnowflakeId() {
	for range 1000 {
		log.Printf(common.GenerateSnowflakeId())
	}

}

var PWD_KEY = "???"

func TestHMAC() {
	hmac := hmac.New(sha256.New, []byte(PWD_KEY))
	hmac.Write([]byte("123456"))
	password := hex.EncodeToString(hmac.Sum(nil))
	fmt.Println(fmt.Sprintf("%s", password))
}

var db *gorm.DB

func testExecuteSql() {
	testDb()
	acc, err := UserInfoGetByAccountId("123456", 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(fmt.Sprintf("id: %+v", acc))
}
func testDb() {
	// 获取配置
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.Llongfile), // io writer
		logger.Config{
			SlowThreshold:             10 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.Error,          // Log level
			IgnoreRecordNotFoundError: true,                  // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,                  // Disable color
		},
	)
	// 构造 MySQL DSN（数据源名称）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		"root", "0", "0", 0, "0")

	// 打开 MySQL 连接
	database, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN 数据源名称
		DefaultStringSize:         256,   // 默认字符串长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的版本不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的版本不支持
		SkipInitializeWithVersion: false, // 根据版本自动配置
	}), &gorm.Config{
		Logger: newLogger, // 使用自定义的 GORM 日志记录器
	})

	// 错误处理
	if err != nil {
		log.Fatal(err)
	}
	db = database

}
func UserInfoGetByAccountId(accountId string, accountType int) (*model.AuthAccount, error) {
	aa := &model.AuthAccount{}
	err := db.Model(&model.AuthAccount{}).Where("account_id = ? and account_type =?", accountId, accountType).Take(&aa).Error
	if err != nil {
		log.Printf("UserInfoGetByAccountId error: %v", err)
		return nil, err
	}
	return aa, nil
}

func generateToken(login model.TgLogin) (string, error) {
	//用时间戳和用户id进行base64编码
	randInt := random.RandInt(0, 1000000)
	s := strconv.FormatInt(int64(randInt), 10)
	formatInt := strconv.FormatInt(time.Now().Unix(), 10)
	token := cryptor.Base64StdEncode(login.TgUserId + formatInt + s)
	return token, nil
}
