package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"log"
)

// import (
//
//	"crypto/hmac"
//	md5 "crypto/md5"
//	"crypto/sha256"
//	"encoding/hex"
//	"fmt"
//	"github.com/duke-git/lancet/v2/cryptor"
//	"github.com/duke-git/lancet/v2/random"
//	"github.com/hellodex/HelloSecurity/api/common"
//	"github.com/hellodex/HelloSecurity/model"
//	"gorm.io/driver/mysql"
//	"gorm.io/gorm"
//	"gorm.io/gorm/logger"
//	"log"
//	"os"
//	"strconv"
//	"time"
//
// )
//
//	func main() {
//		TestTG2WEB()
//	}
//
//	func TestTG2WEB() {
//		for range 1000 {
//			encode := cryptor.Base64StdEncode(fmt.Sprintf("%s%s%d", common.RandomStr(30), "144444", time.Now().UnixNano()))
//			fmt.Println(fmt.Sprintf("%s", encode))
//		}
//	}
//
//	func tsetLock() {
//		lock := common.GetLockWithTTL("testLock", time.Second*3)
//		lock = common.GetLockWithTTL("testLock", time.Second*3)
//		time.Sleep(time.Second * 3)
//		lock = common.GetLockWithTTL("testLock", time.Second*3)
//		lock.Lock.Lock()
//		lock.Lock.Unlock()
//	}
//
//	func TestgenerateTGToken() {
//		for range 10 {
//			s, _ := generateToken(model.TgLogin{UUID: "123456"})
//			fmt.Println(fmt.Sprintf("%s", s))
//		}
//
// }
//
//	func TestMd5() {
//		hash := md5.Sum([]byte("hellodex"))
//		fmt.Println(fmt.Sprintf("%x", hash))
//	}
//
//	func TestRandomString() {
//		for range 10 {
//			log.Printf(common.RandomStr(8))
//		}
//	}
//
//	func TestSnowflakeId() {
//		for range 1000 {
//			log.Printf(common.GenerateSnowflakeId())
//		}
//
// }
//
// var PWD_KEY = "???"
//
//	func TestHMAC() {
//		hmac := hmac.New(sha256.New, []byte(PWD_KEY))
//		hmac.Write([]byte("123456"))
//		password := hex.EncodeToString(hmac.Sum(nil))
//		fmt.Println(fmt.Sprintf("%s", password))
//	}
//
// var db *gorm.DB
//
//	func testExecuteSql() {
//		testDb()
//		acc, err := UserInfoGetByAccountId("123456", 2)
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(fmt.Sprintf("id: %+v", acc))
//	}
//
//	func testDb() {
//		// 获取配置
//		newLogger := logger.New(
//			log.New(os.Stdout, "\r\n", log.Llongfile), // io writer
//			logger.Config{
//				SlowThreshold:             10 * time.Millisecond, // Slow SQL threshold
//				LogLevel:                  logger.Error,          // Log level
//				IgnoreRecordNotFoundError: true,                  // Ignore ErrRecordNotFound error for logger
//				Colorful:                  true,                  // Disable color
//			},
//		)
//		// 构造 MySQL DSN（数据源名称）
//		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
//			"root", "0", "0", 0, "0")
//
//		// 打开 MySQL 连接
//		database, err := gorm.Open(mysql.New(mysql.Config{
//			DSN:                       dsn,   // DSN 数据源名称
//			DefaultStringSize:         256,   // 默认字符串长度
//			DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的版本不支持
//			DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式
//			DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的版本不支持
//			SkipInitializeWithVersion: false, // 根据版本自动配置
//		}), &gorm.Config{
//			Logger: newLogger, // 使用自定义的 GORM 日志记录器
//		})
//
//		// 错误处理
//		if err != nil {
//			log.Fatal(err)
//		}
//		db = database
//
// }
//
//	func UserInfoGetByAccountId(accountId string, accountType int) (*model.AuthAccount, error) {
//		aa := &model.AuthAccount{}
//		err := db.Model(&model.AuthAccount{}).Where("account_id = ? and account_type =?", accountId, accountType).Take(&aa).Error
//		if err != nil {
//			log.Printf("UserInfoGetByAccountId error: %v", err)
//			return nil, err
//		}
//		return aa, nil
//	}
//
//	func generateToken(login model.TgLogin) (string, error) {
//		//用时间戳和用户id进行base64编码
//		randInt := random.RandInt(0, 1000000)
//		s := strconv.FormatInt(int64(randInt), 10)
//		formatInt := strconv.FormatInt(time.Now().Unix(), 10)
//		token := cryptor.Base64StdEncode(login.UUID + formatInt + s)
//		return token, nil
//
// //	}
func main() {
	messageStr := "AbhcR6grmy8wUqM5IjMeXjUssEV+H8j8WBaFiZEsALabE5CJpg8hdYPq2w81OMyPXB7V7VCtT9xYLJNadWL5mwaAAQAGCgSro++mDpb3oto4vYrVHFFRJZWD/9xE02xAvbUqE9/uTUTu8qyEgtpXR5hp/dPP7uVvedu9HkUpe0mqDV454J6VH3itHCFt19c62vRQbTIukrI/yyUkIXF79oI5wPCdLx/9Jd7qecUx4qmefZVHc1DzkBF8eqvtZeBttoJg7n5p7R95XKvcwpcIhCGU4nKdFS0BZYoEl5jd66Go9cbGbW8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAbd9uHXZaGT2cvhRs7reawctIXtX1s3kTqM9YV+/wCpVZFW8aJcbRNPKvfmCpoNNH7HkVZjZGLV0a0m8TU0j2mMlyWPTiSJ8bs9ECkUjg2DC1oTmdr/EIQEjnvY2+n4WQMGRm/lIRcy/+ytunLDm+e8jOW7xfcSayxDmzpAAAAA+sk0JeYRFXvrE+QS6/5AUSKfpu0Rj4C77JairZKhv0UJCQAFAliMAgAJAAkDQEIPAAAAAAAFAgABaQMAAAAEq6Pvpg6W96LaOL2K1RxRUSWVg//cRNNsQL21KhPf7g0AAAAAAAAAMTc0ODkzOTI5MTgyMfAdHwAAAAAApQAAAAAAAAAG3fbh12Whk9nL4UbO63msHLSF7V9bN5E6jPWFfv8AqQYEARgAFgEBBQIAAQwCAAAArfsFAAAAAAAGAQEBEQgGAAIABAUGAQEHIAABAhgEAwcHBwYGCAUVAAECBhIZDw4MDRcRExALFAoZT0XI/vcoNHbKrfsFAAAAAABYoeQDAAAAAPmp2gMAAAAAAQAAAK37BQAAAAAAAQAAAAEAAAABAAAABAEAAABkgJaYgAAAAAAAvpYBAAAAAAAGAwEAAAEJAjYpWVg/JDSlDXbU2A9mF1AJkEtbbIFu0iZydEe56XEmCzAmJygpKissLS4vAMRI+LuCwsD1DXxJ6Vq20Ph+U32vHeJRXh6AKgQLlh0OAAVRRFR7Ww=="
	//decode, _ := base58.Decode(messageStr)
	//toString := base64.StdEncoding.EncodeToString(decode)
	message, _ := base64.StdEncoding.DecodeString(messageStr)

	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(message))
	//fmt.Printf("Transaction: %v\n", tx)
	if err != nil {
		log.Printf("TransactionFromDecoder error: %v", err)
	}
	unitPriceIndex := InstructionIndexGetAndAppendTo(tx, "ComputeBudget111111111111111111111111111111", 3)
	unitLimitIndex := InstructionIndexGetAndAppendTo(tx, "ComputeBudget111111111111111111111111111111", 2)
	if unitPriceIndex > -1 {
		p := tx.Message.Instructions[unitPriceIndex]
		fmt.Println("Instruction  ", binary.LittleEndian.Uint32(p.Data[1:9]))
	}
	if unitLimitIndex > -1 {
		p := tx.Message.Instructions[unitLimitIndex]
		fmt.Println("Instruction  ", binary.LittleEndian.Uint32(p.Data[1:5]))
	}

	//for i, instruction := range tx.Message.Instructions {
	//	programID, err := tx.ResolveProgramIDIndex(instruction.ProgramIDIndex)
	//	if err == nil {
	//		programIDStr := programID.String()
	//		fmt.Printf("Instruction %d: %s\n", i, programIDStr)
	//
	//		data := instruction.Data[1:5]
	//		u := binary.LittleEndian.Uint32(data)
	//		if instruction.Data[0] == 3 {
	//			data = instruction.Data[1:9]
	//			u1 := binary.LittleEndian.Uint64(data)
	//			fmt.Printf("Instruction %d: %d\n", i, u1)
	//		} else {
	//			fmt.Printf("Instruction %d: %d\n", i, u)
	//		}
	//
	//	}
	//
	//}
	//conf := &hc.OpConfig{
	//	UnitLimit: new(big.Int).SetInt64(21),
	//	UnitPrice: new(big.Int).SetInt64(999),
	//}
	//SimulateTransaction(rpc.New("https://mainnet.helius-rpc.com/?api-key=50465b2c-93d8-4d53-8987-a9ccd7962504"), tx)
}
func InstructionIndexGetAndAppendTo(tx *solana.Transaction, queryProgramID string, discriminator byte) int16 {

	program := solana.MustPublicKeyFromBase58(queryProgramID)
	for i, instruction := range tx.Message.Instructions {
		programID, err := tx.ResolveProgramIDIndex(instruction.ProgramIDIndex)
		if err == nil && programID.Equals(program) {
			if instruction.Data[0] == discriminator {
				return int16(i)
			}
		}
	}
	return int16(-1)
}
