package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/hellodex/HelloSecurity/swapData"
	"github.com/mr-tron/base58"
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
func main3() {
	okxres := `{"code":"0","data":[{"routerResult":{"chainId":"501","chainIndex":"501","contextSlot":0,"dexRouterList":[{"router":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v--6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN","routerPercent":"100","subRouterList":[{"dexProtocol":[{"dexName":"Meteora DLMM","percent":"100"}],"fromToken":{"decimal":"6","isHoneyPot":false,"taxRate":"0","tokenContractAddress":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","tokenSymbol":"USDC","tokenUnitPrice":"0.999796584051310263"},"toToken":{"decimal":"6","isHoneyPot":false,"taxRate":"0","tokenContractAddress":"6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN","tokenSymbol":"TRUMP","tokenUnitPrice":"8.894400919889379544"}}]}],"estimateGasFee":"232000","fromToken":{"decimal":"6","isHoneyPot":false,"taxRate":"0","tokenContractAddress":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","tokenSymbol":"USDC","tokenUnitPrice":"0.999796584051310263"},"fromTokenAmount":"100000","priceImpactPercentage":"0.01","quoteCompareList":[{"amountOut":"0.011124","dexLogo":"https://static.okx.com/cdn/web3/dex/logo/SolFi.png","dexName":"SolFi","tradeFee":"0.0000002873904"},{"amountOut":"0.011109","dexLogo":"https://static.okx.com/cdn/explorer/dex/logo/Raydium.png","dexName":"Raydium CPMM","tradeFee":"0.00000012818265"},{"amountOut":"0.011109","dexLogo":"https://static.okx.com/cdn/explorer/dex/logo/Raydium.png","dexName":"Raydium CL","tradeFee":"0.0000002873904"},{"amountOut":"0.011108","dexLogo":"https://static.okx.com/cdn/wallet/logo/dex_orcaswap.png","dexName":"Orca Whirlpools","tradeFee":"0.0000001730874"}],"swapMode":"exactIn","toToken":{"decimal":"6","isHoneyPot":false,"taxRate":"0","tokenContractAddress":"6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN","tokenSymbol":"TRUMP","tokenUnitPrice":"8.894400919889379544"},"toTokenAmount":"11130","tradeFee":"0.000815"},"tx":{"data":"2A2qbHxwdYqyeirixtYp6eugjM1wmgdr9Zm8hNfeQwLfHkd9VDM5YNfP9VPus66KApwntGqNRdEoTGm3jtq9h4W79jsF1i74Tn2q5EpeYoWcJ1fF2N2B5x2wVjZBnbET4JNStVinSR6n4KvXDh8dfM3ncFA9Pk5XNkUanHg4vKNwzvpqn3zVx3jiexSt8TihB2W6Su7sPpYrre7d4NtbNQkkLiHHuRgRgCFSE5Fg7SbUGEJrz52e9dogzvKHAuQuaKg8fsxHTDbctzqSYsawpWsTGMN8oYborrrLPQnaUxzZnzncR3qhoMcJDy5RE53DBDtw9gdS2CMvyHUYoyjczu7KvAqdnYVMBnksHUhWdEmuY2qrss21DAr4VwwkirMBFLuzGBGKDiLn6eyWYxJKeaCWnk3a2hpHZDu69zQifSD3B4iBbkPwJwke2KrbkXCB2JZFAgdEuqXH4LeU7NYE2xGJchBzh7vBfkdz4xTaGTayo8gqFLKQ2Yh7L9kF6hDeqgDNhGWaGjtC8n2SshtPBAqybvgdafE7VAgbXJJYGbTH4uhNEnVV7R2CwGntRCA8AH4fgir7Qq6BsqtFuG2n5XZwiB8E5YaBtGrug1JCDXLJXNfEjY4EaLutnea6QdeZ5puHFnLDesxBvDJB3aVw8TPfSygQGu9k1ptKesE2DRfnze9U8WwPUxy4nCxRRcF67qHAHFM1BriW5vQCUqWqhapypPUUjWG1VTPdZ1xLSQ12Zb1PXZkz7uXRhNQonG6PcuFX6sBqzvSSgaUBnUfnJ62anALWCrv9W9bKY3BZuBLvnd8rG3zTyBAm3AU8gPpx28cvTT4PAvkcm1JuDGDHi17W","from":"KERxu1WdAfziZbmRkZnpj7mUgyJrLGdYC7d1VMwPR25","gas":"33333","gasPrice":"50000000000000","maxPriorityFeePerGas":"","maxSpendAmount":"100000","minReceiveAmount":"11018","signatureData":["{\"jitoCalldata\":\"2bogaVmn2zQ5tRD135xkUrmmfLrQoMFAd9DFWr8ymo47iQFMA6czAg3HfFopi1SnVigiMENAeEzNj2dogPPETxijZihEtpeuVcmWznBEnqPqdLVtSSZozsG4mWNkJ1r379hTUhNJrK48adqnAtQhXHzQiivKrKxQButuk9ZniAZJaWekkWZMRs8syjHZwqGTsGJFnGWDAwFopE99CvgAcbozFMMd1TTqAUPHCfnkxbRchMpCoWQHA9NoExbi7EnsLYsE3JtH5ADs5uZ2Hx2SkLjnY5FEnxzGw7644QVd\"}"],"slippage":"0.01","to":"6m2CDdhRgxpH4WjvdzxAYbGxwdGUz5MziiL5jek2kBma","value":"0"}}],"msg":""}`
	var okxRes swapData.OkxResponse
	if err := json.Unmarshal([]byte(okxres), &okxRes); err != nil {
		fmt.Println(err)
	}
	if len(okxRes.Data) > 0 && len(okxRes.Data[0].Tx.SignatureData) > 0 {
		var m map[string]string
		if err := json.Unmarshal([]byte(okxRes.Data[0].Tx.SignatureData[0]), &m); err == nil {
			if s, exist := m["jitoCalldata"]; exist {
				fmt.Println(s)
			}
		}
	}
	spew.Dump(okxRes)

}
func main() {
	messageStr := `2bogaVmn2zQ5tRD135xkUrmmfLrQoMFAd9DFWr8ymo47iQFMA6czAg3HfFopi1SnVigiMENAeEzNj2dogPPETxijZihEtpeuVcmWznBEnqPqdLVtSSZozsG4mWNkJ1r379hTUhNJrK3dQmyueqBrsxihjGTAV2ESQqhRvrKnzQvitGNEQUDNBXi9updMFZQ2M1Cab7Fmu4uEX6N1YNJ2jRCXdZxN11TKsGpEj9tum2Yv6Zk3QKxNYK3qmX2B5wUci43w5kNDzZooSmQmwzUJJANkTGUL33vgciydQHAX` //decode, _ := base58.Decode(messageStr)
	decode, err2 := base58.Decode(messageStr)
	message, _ := base64.StdEncoding.DecodeString(messageStr)
	log.Printf("message: %v err: %v", message, err2)
	tx, err := solana.TransactionFromDecoder(bin.NewBorshDecoder(decode))
	//fmt.Printf("Transaction: %v\n", tx)
	if err != nil {
		log.Printf("TransactionFromDecoder error: %v", err)
	}
	spew.Dump(tx.Message.AccountKeys)
	spew.Dump(tx)
	spew.Dump(tx.Message.Instructions)
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
