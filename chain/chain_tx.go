package chain

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/shopspring/decimal"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	compute_budget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	hc "github.com/hellodex/HelloSecurity/api/common"
	"github.com/hellodex/HelloSecurity/config"
	"github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/wallet"
	"github.com/hellodex/HelloSecurity/wallet/enc"
	"github.com/mr-tron/base58"
)

var mylog = log.GetLogger()

const maxRetries = 30

var ZERO = big.NewInt(0)

const fixedTestAddr = "KERxu1WdAfziZbmRkZnpj7mUgyJrLGdYC7d1VMwPR25"

var transferFnSignature = []byte("transfer(address,uint256)")

const erc20ABI = `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

// 通过jito发送，
// func HandleMessage(t *config.ChainConfig, messageStr string, to string, typecode string,
//
//	value *big.Int,
//	conf *hc.OpConfig,
//	wg *model.WalletGenerated,
//
//	) (txhash string, sig []byte, err error) {
//		mylog.Info("调用HandleMessage")
//		if len(t.GetRpc()) == 0 {
//			return txhash, sig, errors.New("rpc_config")
//		}
//		rpcUrlDefault := t.GetRpc()[0]
//		if len(conf.Rpc) > 0 {
//			rpcUrlDefault = conf.Rpc
//		}
//		mylog.Infof("RPC for transaction current used: %s", rpcUrlDefault)
//
//		if wg.ChainCode == "SOLANA" {
//			message, _ := base64.StdEncoding.DecodeString(messageStr)
//			if typecode == "sign" {
//				sig, err = enc.Porter().SigSol(wg, message)
//				if err != nil {
//					mylog.Error("type=", typecode, err)
//					return txhash, sig, err
//				}
//				return txhash, sig, err
//			}
//
//			casttype, err := parseCallType(conf.Type)
//			if err != nil {
//				casttype = CallTypeGeneral
//			}
//			// 使用多个rpc节点确认交易
//			rpcList := make([]*rpc.Client, 0)
//			splitUrl := strings.Split(rpcUrlDefault, ",")
//			mapUrl := make(map[string]bool)
//			for _, s := range splitUrl {
//				_, exi := mapUrl[s]
//				if len(s) > 0 && !exi {
//					rpcList = append(rpcList, rpc.New(s))
//					mapUrl[s] = true
//				}
//			}
//
//			tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(message))
//			if err != nil {
//				mylog.Error("TransactionFromDecoder error: ", message, " err:", err)
//				return txhash, sig, err
//			}
//
//			// if wg.Wallet == fixedTestAddr {
//			// 	casttype = CallTypeJito
//			// }
//
//			var tipAdd string
//			var sepdr = solana.MustPublicKeyFromBase58(wg.Wallet)
//			if casttype == CallTypeJito {
//				//tipAdd, err = getTipAccounts()
//				//mylog.Infof("[jito]fetch account response %v, %v", tipAdd, err)
//				//if err != nil {
//				//	return txhash, sig, err
//				//}
//
//				mylog.Infof("[jito] request %v", conf)
//				if casttype == CallTypeJito {
//					tipAcc, err := solana.PublicKeyFromBase58("3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT")
//					if err != nil {
//						mylog.Errorf("[jito]unparsed data %s %v", tipAdd, err)
//					} else if conf.Tip.Cmp(ZERO) == 1 {
//						var numSigs = tx.Message.Header.NumRequiredSignatures
//						var numRSig = tx.Message.Header.NumReadonlySignedAccounts
//						var numRUSig = tx.Message.Header.NumReadonlyUnsignedAccounts
//						mylog.Infof("[jito] tx header summary %d %d %d", numSigs, numRSig, numRUSig)
//						programIDIndex := uint16(0)
//						foundSystem := false
//						for i, acc := range tx.Message.AccountKeys {
//							if acc.Equals(system.ProgramID) {
//								programIDIndex = uint16(i)
//								foundSystem = true
//								break
//							}
//						}
//						if !foundSystem {
//							mylog.Info("[jito]reset system program id")
//							tx.Message.AccountKeys = append(tx.Message.AccountKeys, system.ProgramID)
//							programIDIndex = uint16(len(tx.Message.AccountKeys) - 1)
//						}
//
//						writableStartIndex := int(tx.Message.Header.NumRequiredSignatures)
//						// writableEndIndex := len(tx.Message.AccountKeys) - int(tx.Message.Header.NumReadonlyUnsignedAccounts)
//
//						// tx.Message.AccountKeys = append(tx.Message.AccountKeys, tipAcc)
//						preBoxes := append([]solana.PublicKey{}, tx.Message.AccountKeys[:writableStartIndex]...)
//						postBoxes := append([]solana.PublicKey{}, tx.Message.AccountKeys[writableStartIndex:]...)
//						tx.Message.AccountKeys = append(
//							append(preBoxes, tipAcc),
//							postBoxes...,
//						)
//
//						mylog.Infof("[jito] program index %d, %d", programIDIndex, writableStartIndex)
//
//						transferInstruction := system.NewTransferInstruction(
//							conf.Tip.Uint64(),
//							sepdr,
//							tipAcc,
//						)
//						data := transferInstruction.Build()
//						dData, _ := data.Data()
//						if programIDIndex >= uint16(writableStartIndex) {
//							programIDIndex += uint16(1)
//						}
//
//						compiledTransferInstruction := solana.CompiledInstruction{
//							ProgramIDIndex: programIDIndex,
//							Accounts: []uint16{
//								0,
//								uint16(writableStartIndex),
//							},
//							Data: dData,
//						}
//						tx.Message.Instructions = append(tx.Message.Instructions, compiledTransferInstruction)
//
//						updateInstructionIndexes(tx, writableStartIndex)
//					}
//				}
//			}
//
//			timeStart := time.Now().UnixMilli()
//			hashResult, err := GetLatestBlockhashFromMultipleClients(rpcList,rpc.CommitmentFinalized)
//			timeEnd := time.Now().UnixMilli() - timeStart
//			mylog.Infof("EX HandleMessage getblock %dms", timeEnd)
//			if err != nil {
//				mylog.Error("Get RecentBlockhash error: ", err)
//				return txhash, sig, err
//			}
//			mylog.Infof("Get RecentBlockhash：%s,Block: %d ", hashResult.Value.Blockhash, hashResult.Value.LastValidBlockHeight)
//			tx.Message.RecentBlockhash = hashResult.Value.Blockhash
//
//			msgBytes, _ := tx.Message.MarshalBinary()
//			sig, err = enc.Porter().SigSol(wg, msgBytes)
//			if err != nil {
//				mylog.Error("SigSol error wg: ", wg.Wallet, " err:", err)
//				return txhash, sig, err
//			}
//
//			mylog.Infof("EX Signed result sig %s %dms", base64.StdEncoding.EncodeToString(sig), time.Now().UnixMilli()-timeEnd)
//			timeEnd = time.Now().UnixMilli() - timeEnd
//			tx.Signatures = []solana.Signature{solana.Signature(sig)}
//
//			//txhash, err := rpcList.SendTransaction(context.Background(), tx)
//			//txhash, status, err := SendAndConfirmTransaction(rpcList[0], tx, casttype, conf.ShouldConfirm, conf.ConfirmTimeOut)
//			txhash, status, err := SendAndConfirmTransactionWithClients(rpcList, tx, casttype, conf.ShouldConfirm, conf.ConfirmTimeOut)
//			mylog.Infof("EX Txhash %s, status:%s, %dms", txhash, status, time.Now().UnixMilli()-timeEnd)
//
//			if status == "finalized" || status == "confirmed" || status == "processed" {
//				mylog.Info("rpc确认状态成功201 :", status)
//				mylog.Info("err:", err)
//				//mylog.Info(err.Error())
//				return txhash, sig, err
//			}
//
//			if err != nil {
//				mylog.Info("rpc确认状态成功208 :", status)
//				return txhash, sig, fmt.Errorf(err.Error()+" status:%s", status)
//			} else {
//				mylog.Info("rpc确认状态成功210 :", status)
//				return txhash, sig, fmt.Errorf("status:%s", status)
//			}
//		} else { // for all evm
//			message, err := hexutil.Decode(messageStr)
//			if err != nil {
//				return txhash, sig, err
//			}
//			if typecode == "sign" {
//				sig, err = enc.Porter().SigEth(wg, message)
//				if err != nil {
//					return txhash, sig, err
//				}
//				return txhash, sig, err
//			}
//			client, _ := ethclient.Dial(rpcUrlDefault)
//
//			nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(wg.Wallet))
//			if err != nil {
//				return txhash, sig, err
//			}
//
//			var gasPrice *big.Int
//			if conf != nil && conf.UnitPrice != nil && conf.UnitPrice.Uint64() > 0 {
//				gasPrice = conf.UnitPrice
//			} else {
//				gasPrice, err = client.SuggestGasPrice(context.Background())
//				if err != nil {
//					return txhash, sig, err
//				}
//			}
//
//			value := value
//			gasLimit := uint64(500000)
//			if conf != nil && conf.UnitLimit != nil && conf.UnitLimit.Uint64() > 0 {
//				gasLimit = conf.UnitLimit.Uint64()
//			}
//			tx := types.NewTransaction(nonce, common.HexToAddress(to), value, gasLimit, gasPrice, message)
//
//			// 查询链 ID
//			chainID, err := client.NetworkID(context.Background())
//			if err != nil {
//				return txhash, sig, err
//			}
//
//			// 对交易进行签名
//			signedTx, err := enc.Porter().SigEvmTx(wg, tx, chainID)
//			if err != nil {
//				return txhash, sig, err
//			}
//
//			// 发送已签名的交易
//			err = client.SendTransaction(context.Background(), signedTx)
//
//			return signedTx.Hash().Hex(), sig, err
//		}
//	}
func HandleMessage(t *config.ChainConfig, messageStr string, to string, typecode string,
	value *big.Int,
	conf *hc.OpConfig,
	wg *model.WalletGenerated,
) (txhash string, sig []byte, err error) {
	mylog.Info("调用HandleMessage")
	// 检查链配置是否包含 RPC 端点，如果没有则返回错误。
	if len(t.GetRpc()) == 0 {
		return txhash, sig, errors.New("rpc_config")
	}

	// 默认使用链配置中的第一个 RPC 端点。
	rpcUrlDefault := t.GetRpc()[0]
	// 如果操作配置中指定了 RPC URL，则优先使用它。
	if len(conf.Rpc) > 0 {
		rpcUrlDefault = conf.Rpc
	}
	// 记录当前使用的 RPC 端点。
	mylog.Infof("RPC for transaction current used: %s", rpcUrlDefault)

	// 检查是否为 Solana 链。
	if wg.ChainCode == "SOLANA" {
		// 解码 Base64 编码的消息字符串。
		message, _ := base64.StdEncoding.DecodeString(messageStr)

		// 如果操作类型为 "sign"，仅对消息进行签名。
		if typecode == "sign" {
			// 调用签名方法 SigSol 对消息进行签名。
			sig, err = enc.Porter().SigSol(wg, message)
			if err != nil {
				// 签名失败，记录错误并返回。
				mylog.Error("type=", typecode, err)
				return txhash, sig, err
			}
			// 签名成功，返回签名结果（txhash 为空）。
			return txhash, sig, err
		}

		// 解析操作配置中的交易类型（如 Jito 或 General）。
		casttype, err := parseCallType(conf.Type)
		if err != nil {
			// 解析失败，默认使用通用交易类型。
			casttype = CallTypeGeneral
		}

		// 初始化 RPC 客户端列表，用于与多个 RPC 节点交互以确认交易。
		rpcList := make([]*rpc.Client, 0)
		// 将 RPC URL 按逗号分割，可能包含多个端点。
		splitUrl := strings.Split(rpcUrlDefault, ",")
		// 使用 map 去重，防止重复添加相同的 RPC 端点。
		mapUrl := make(map[string]bool)
		rpcUrls := make([]string, 0)
		for _, s := range splitUrl {
			_, exi := mapUrl[s]
			// 仅添加非空且未重复的 RPC 端点。
			if len(s) > 0 && !exi {
				rpcUrls = append(rpcUrls, s)
				rpcList = append(rpcList, rpc.New(s))
				mapUrl[s] = true
			}
		}

		// 从解码的消息中解析 Solana 交易。
		tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(message))
		if err != nil {
			// 解析交易失败，记录错误并返回。
			mylog.Error("TransactionFromDecoder error: ", message, " err:", err)
			return txhash, sig, err
		}

		//var tipAdd string
		//// 将钱包地址转换为 Solana 公钥。
		//
		//if casttype == CallTypeJito {
		//	mylog.Infof("[jito] request %v", conf)
		//	if err != nil {
		//
		//		mylog.Errorf("[jito]unparsed data %s %v", tipAdd, err)
		//	} else if conf.Tip.Cmp(ZERO) == 1 {
		//		// 设置jito费用
		//		mylog.Infof("jito小费 %s", conf.Tip.String())
		//		_, _ = SimulateTransaction(rpcList[0], tx, conf)
		//		AddInstruction(tx, "3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT", conf.Tip, wg.Wallet)
		//		//设置优先费
		//		tx.Message.Instructions = appendUnitPrice(conf, tx)
		//	}
		//}

		// 记录获取最新区块哈希的开始时间。
		timeStart := time.Now().UnixMilli()
		//  并发获取最新区块哈希的功能。
		mylog.Infof("rpcs:%v", strings.Join(rpcUrls, ","))
		hashResult, err := GetLatestBlockhashFromMultipleClients(rpcList, rpc.CommitmentFinalized)
		// 计算耗时并记录。
		timeEnd := time.Now().UnixMilli() - timeStart
		mylog.Infof("EX HandleMessage getblock %dms", timeEnd)
		if err != nil {
			// 获取区块哈希失败，记录错误并返回。
			mylog.Error("Get RecentBlockhash error: ", err)
			return txhash, sig, err
		}
		// 记录获取的区块哈希和有效区块高度。
		mylog.Infof("Get RecentBlockhash：%s,Block: %d ", hashResult.Value.Blockhash, hashResult.Value.LastValidBlockHeight)

		// 将最新区块哈希设置到交易中。
		tx.Message.RecentBlockhash = hashResult.Value.Blockhash

		// 序列化交易消息以进行签名。
		msgBytes, _ := tx.Message.MarshalBinary()
		// 对交易消息进行签名。
		sig, err = enc.Porter().SigSol(wg, msgBytes)
		if err != nil {
			// 签名失败，记录错误并返回。
			mylog.Error("SigSol error wg: ", wg.Wallet, " err:", err)
			return txhash, sig, err
		}
		// 记录签名结果和耗时。
		mylog.Infof("EX Signed result sig %s %dms", base64.StdEncoding.EncodeToString(sig), time.Now().UnixMilli()-timeEnd)

		// 更新耗时。
		timeEnd = time.Now().UnixMilli() - timeEnd
		// 将签名添加到交易的签名列表中。
		tx.Signatures = []solana.Signature{solana.Signature(sig)}

		// 使用多个 RPC 客户端发送并确认交易。
		txhash, status, err := SendAndConfirmTransactionWithClients(rpcList, tx, casttype, conf.ShouldConfirm, conf.ConfirmTimeOut)
		// 记录交易哈希、状态和耗时。
		mylog.Infof("EX Txhash %s, status:%s, %dms", txhash, status, time.Now().UnixMilli()-timeEnd)

		// 检查交易状态是否为已确认或已最终化。
		if status == "finalized" || status == "confirmed" || status == "processed" {
			mylog.Info("rpc确认状态成功201 :", status)
			mylog.Info("err:", err)
			//mylog.Info(err.Error())
			return txhash, sig, err
		}

		if err != nil {
			mylog.Info("rpc确认状态成功208 :", status)
			return txhash, sig, fmt.Errorf(err.Error()+" status:%s", status)
		} else {
			mylog.Info("rpc确认状态成功210 :", status)
			return txhash, sig, fmt.Errorf("status:%s", status)
		}
	} else { // for all evm
		start := time.Now()
		message, err := hexutil.Decode(messageStr)
		if err != nil {
			return txhash, sig, err
		}
		if typecode == "sign" {
			sig, err = enc.Porter().SigEth(wg, message)
			if err != nil {
				return txhash, sig, err
			}
			return txhash, sig, err
		}
		client, _ := ethclient.Dial(rpcUrlDefault)

		nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(wg.Wallet))
		if err != nil {
			return txhash, sig, err
		}

		var gasPrice *big.Int
		if conf != nil && conf.UnitPrice != nil && conf.UnitPrice.Uint64() > 0 {
			gasPrice = conf.UnitPrice
		} else {
			gasPrice, err = client.SuggestGasPrice(context.Background())
			if err != nil {
				return txhash, sig, err
			}
		}

		value := value
		gasLimit := uint64(500000)
		if conf != nil && conf.UnitLimit != nil && conf.UnitLimit.Uint64() > 0 {
			gasLimit = conf.UnitLimit.Uint64()
		}
		tx := types.NewTransaction(nonce, common.HexToAddress(to), value, gasLimit, gasPrice, message)

		// 查询链 ID
		chainID, err := client.NetworkID(context.Background())
		if err != nil {
			return txhash, sig, err
		}

		// 对交易进行签名
		signedTx, err := enc.Porter().SigEvmTx(wg, tx, chainID)
		if err != nil {
			return txhash, sig, err
		}
		elapsed := time.Since(start)
		fmt.Printf("发送tx之前 耗时: %d ms\n", elapsed.Milliseconds())
		start = time.Now()
		err = client.SendTransaction(context.Background(), signedTx)
		elapsed = time.Since(start)
		fmt.Printf("SendTransaction 耗时: %d ms\n", elapsed.Milliseconds())

		return signedTx.Hash().Hex(), sig, err
	}
}

func ProgramIndexGetAndAppendToAccountKeys(tx *solana.Transaction, programID string) uint16 {

	program := solana.MustPublicKeyFromBase58(programID)
	programIndex := uint16(0)
	foundComputeBudget := false
	for i, acc := range tx.Message.AccountKeys {
		if acc.Equals(program) {
			programIndex = uint16(i)
			foundComputeBudget = true
			break
		}
	}
	// 如果未找到，添加到账户列表
	if !foundComputeBudget {
		tx.Message.AccountKeys = append(tx.Message.AccountKeys, program)
		programIndex = uint16(len(tx.Message.AccountKeys) - 1)
	}
	return programIndex
}

// InstructionIndexGetAndAppendTo ComputeUnitLimit:2 ComputeUnitPrice:3
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
func HandleMessageTest(t *config.ChainConfig, messageStr string, to string, typecode string,
	value *big.Int,
	conf *hc.OpConfig,
	wg *model.WalletGenerated,
) (txhash string, sig []byte, err error) {
	mylog.Info("HandleMessageTest")
	// 检查链配置是否包含 RPC 端点，如果没有则返回错误。
	if len(t.GetRpc()) == 0 {
		return txhash, sig, errors.New("rpc_config")
	}

	// 默认使用链配置中的第一个 RPC 端点。
	rpcUrlDefault := t.GetRpc()[0]
	// 如果操作配置中指定了 RPC URL，则优先使用它。
	if len(conf.Rpc) > 0 {
		rpcUrlDefault = conf.Rpc
	}
	// 记录当前使用的 RPC 端点。
	mylog.Infof("RPC for transaction current used: %s", rpcUrlDefault)

	// 检查是否为 Solana 链。
	if wg.ChainCode == "SOLANA" {
		// 解码 Base64 编码的消息字符串。
		message, _ := base64.StdEncoding.DecodeString(messageStr)

		// 如果操作类型为 "sign"，仅对消息进行签名。
		if typecode == "sign" {
			// 调用签名方法 SigSol 对消息进行签名。
			sig, err = enc.Porter().SigSol(wg, message)
			if err != nil {
				// 签名失败，记录错误并返回。
				mylog.Error("type=", typecode, err)
				return txhash, sig, err
			}
			// 签名成功，返回签名结果（txhash 为空）。
			return txhash, sig, err
		}

		// 解析操作配置中的交易类型（如 Jito 或 General）。
		casttype, err := parseCallType(conf.Type)
		if err != nil {
			// 解析失败，默认使用通用交易类型。
			casttype = CallTypeGeneral
		}

		// 初始化 RPC 客户端列表，用于与多个 RPC 节点交互以确认交易。
		rpcList := make([]*rpc.Client, 0)
		// 将 RPC URL 按逗号分割，可能包含多个端点。
		splitUrl := strings.Split(rpcUrlDefault, ",")
		// 使用 map 去重，防止重复添加相同的 RPC 端点。
		rpcUrls := make([]string, 0)
		mapUrl := make(map[string]bool)
		for _, s := range splitUrl {
			_, exi := mapUrl[s]
			// 仅添加非空且未重复的 RPC 端点。
			if len(s) > 0 && !exi {
				rpcList = append(rpcList, rpc.New(s))
				mapUrl[s] = true
			}
		}

		// 从解码的消息中解析 Solana 交易。
		tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(message))
		if err != nil {
			// 解析交易失败，记录错误并返回。
			mylog.Error("TransactionFromDecoder error: ", message, " err:", err)
			return txhash, sig, err
		}

		var tipAdd string
		// 将钱包地址转换为 Solana 公钥。
		//var sepdr = solana.MustPublicKeyFromBase58(wg.Wallet)

		if casttype == CallTypeJito {
			mylog.Infof("[jito] request %v", conf)
			if err != nil {

				mylog.Errorf("[jito]unparsed data %s %v", tipAdd, err)
			} else if conf.Tip.Cmp(ZERO) == 1 {
				// 设置jito费用
				AddInstruction(tx, "3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT", conf.Tip, wg.Wallet)
			}
		}
		//设置优先费
		_, _ = SimulateTransaction(rpcList[0], tx, conf)
		tx.Message.Instructions = appendUnitPrice(conf, tx)
		// 记录获取最新区块哈希的开始时间。
		timeStart := time.Now().UnixMilli()
		// 新增并发获取最新区块哈希的功能。
		mylog.Infof("rpcs:%v", strings.Join(rpcUrls, ","))
		hashResult, err := GetLatestBlockhashFromMultipleClients(rpcList, rpc.CommitmentFinalized)
		// 计算耗时并记录。
		timeEnd := time.Now().UnixMilli() - timeStart
		mylog.Infof("EX HandleMessage getblock %dms", timeEnd)
		if err != nil {
			// 获取区块哈希失败，记录错误并返回。
			mylog.Error("Get RecentBlockhash error: ", err)
			return txhash, sig, err
		}
		// 记录获取的区块哈希和有效区块高度。
		mylog.Infof("Get RecentBlockhash：%s,Block: %d ", hashResult.Value.Blockhash, hashResult.Value.LastValidBlockHeight)

		// 将最新区块哈希设置到交易中。
		tx.Message.RecentBlockhash = hashResult.Value.Blockhash

		// 序列化交易消息以进行签名。
		msgBytes, _ := tx.Message.MarshalBinary()
		// 对交易消息进行签名。
		sig, err = enc.Porter().SigSol(wg, msgBytes)
		if err != nil {
			// 签名失败，记录错误并返回。
			mylog.Error("SigSol error wg: ", wg.Wallet, " err:", err)
			return txhash, sig, err
		}
		// 记录签名结果和耗时。
		mylog.Infof("EX Signed result sig %s %dms", base64.StdEncoding.EncodeToString(sig), time.Now().UnixMilli()-timeEnd)

		// 更新耗时。
		timeEnd = time.Now().UnixMilli() - timeEnd
		// 将签名添加到交易的签名列表中。
		tx.Signatures = []solana.Signature{solana.Signature(sig)}

		// 使用多个 RPC 客户端发送并确认交易。
		txhash, status, err := SendAndConfirmTransactionWithClients(rpcList, tx, casttype, conf.ShouldConfirm, conf.ConfirmTimeOut)
		// 记录交易哈希、状态和耗时。
		mylog.Infof("EX Txhash %s, status:%s, %dms", txhash, status, time.Now().UnixMilli()-timeEnd)

		// 检查交易状态是否为已确认或已最终化。
		if status == "finalized" || status == "confirmed" || status == "processed" {
			mylog.Info("rpc确认状态成功201 :", status)
			mylog.Info("err:", err)
			//mylog.Info(err.Error())
			return txhash, sig, err
		}

		if err != nil {
			mylog.Info("rpc确认状态成功208 :", status)
			return txhash, sig, fmt.Errorf(err.Error()+" status:%s", status)
		} else {
			mylog.Info("rpc确认状态成功210 :", status)
			return txhash, sig, fmt.Errorf("status:%s", status)
		}
	} else { // for all evm
		message, err := hexutil.Decode(messageStr)
		if err != nil {
			return txhash, sig, err
		}
		if typecode == "sign" {
			sig, err = enc.Porter().SigEth(wg, message)
			if err != nil {
				return txhash, sig, err
			}
			return txhash, sig, err
		}
		client, _ := ethclient.Dial(rpcUrlDefault)

		nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(wg.Wallet))
		if err != nil {
			return txhash, sig, err
		}

		var gasPrice *big.Int
		if conf != nil && conf.UnitPrice != nil && conf.UnitPrice.Uint64() > 0 {
			gasPrice = conf.UnitPrice
		} else {
			gasPrice, err = client.SuggestGasPrice(context.Background())
			if err != nil {
				return txhash, sig, err
			}
		}

		value := value
		gasLimit := uint64(500000)
		if conf != nil && conf.UnitLimit != nil && conf.UnitLimit.Uint64() > 0 {
			gasLimit = conf.UnitLimit.Uint64()
		}
		tx := types.NewTransaction(nonce, common.HexToAddress(to), value, gasLimit, gasPrice, message)

		// 查询链 ID
		chainID, err := client.NetworkID(context.Background())
		if err != nil {
			return txhash, sig, err
		}

		// 对交易进行签名
		signedTx, err := enc.Porter().SigEvmTx(wg, tx, chainID)
		if err != nil {
			return txhash, sig, err
		}

		// 发送已签名的交易
		err = client.SendTransaction(context.Background(), signedTx)

		return signedTx.Hash().Hex(), sig, err
	}
}

func AddInstruction(tx *solana.Transaction, address string, tip *big.Int, wallet string) {
	//mylog.Info("调用AddInstruction")

	tipAcc, err := solana.PublicKeyFromBase58(address)
	var sepdr = solana.MustPublicKeyFromBase58(wallet)
	if tip == nil || tip.Cmp(ZERO) < 1 {
		err = fmt.Errorf("tip is nil")
	}
	if err != nil {
		// 解析 Tip 账户地址失败，记录错误。
		mylog.Errorf("[jito]unparsed data %s %v", address, err)
	} else {

		var numSigs = tx.Message.Header.NumRequiredSignatures
		var numRSig = tx.Message.Header.NumReadonlySignedAccounts
		var numRUSig = tx.Message.Header.NumReadonlyUnsignedAccounts
		mylog.Infof("[jito] tx header summary %d %d %d", numSigs, numRSig, numRUSig)

		// 查找系统程序 ID 的索引。
		programIDIndex := uint16(0)
		foundSystem := false
		for i, acc := range tx.Message.AccountKeys {
			if acc.Equals(system.ProgramID) {
				programIDIndex = uint16(i)
				foundSystem = true
				break
			}
		}
		// 如果未找到系统程序 ID，则添加并更新索引。
		if !foundSystem {
			mylog.Info("[jito]reset system program id")
			tx.Message.AccountKeys = append(tx.Message.AccountKeys, system.ProgramID)
			programIDIndex = uint16(len(tx.Message.AccountKeys) - 1)
		}

		// 计算可写账户的起始索引。
		writableStartIndex := int(tx.Message.Header.NumRequiredSignatures)

		// 将 Tip 账户插入到账户列表中，保持可写和只读账户的顺序。
		preBoxes := append([]solana.PublicKey{}, tx.Message.AccountKeys[:writableStartIndex]...)
		postBoxes := append([]solana.PublicKey{}, tx.Message.AccountKeys[writableStartIndex:]...)
		tx.Message.AccountKeys = append(append(preBoxes, tipAcc), postBoxes...)

		// 记录程序索引和可写账户起始索引。
		mylog.Infof("[jito] program index %d, %d", programIDIndex, writableStartIndex)

		// 创建系统转账指令，用于支付 Tip 金额。
		transferInstruction := system.NewTransferInstruction(
			tip.Uint64(),
			sepdr,
			tipAcc,
		)
		// 构建指令数据。
		data := transferInstruction.Build()
		dData, _ := data.Data()

		// 如果系统程序索引在可写账户之后，需调整索引。
		if programIDIndex >= uint16(writableStartIndex) {
			programIDIndex += uint16(1)
		}

		// 编译转账指令，包含程序 ID 索引、账户索引和数据。
		compiledTransferInstruction := solana.CompiledInstruction{
			ProgramIDIndex: programIDIndex,
			Accounts:       []uint16{0, uint16(writableStartIndex)},
			Data:           dData,
		}
		// 将转账指令添加到交易的指令列表中。
		tx.Message.Instructions = append(tx.Message.Instructions, compiledTransferInstruction)

		// 更新交易中所有指令的账户索引，以适应新增的 Tip 账户。
		updateInstructionIndexes(tx, writableStartIndex)
	}
}

// 冲狗基金交易50%归属基金钱包
func MemeVaultHandleMessage(t *config.ChainConfig, messageStr string, to string, typecode string,
	value *big.Int,
	conf *hc.OpConfig,
	wg *model.WalletGenerated,
) (txhash string, sig []byte, err error) {
	mylog.Info("调用MemeVaultHandleMessage")
	if len(t.GetRpc()) == 0 {
		return txhash, sig, errors.New("rpc_config")
	}
	rpcUrlDefault := t.GetRpc()[0]
	if len(conf.Rpc) > 0 {
		rpcUrlDefault = conf.Rpc
	}
	mylog.Infof("RPC for transaction current used: %s", rpcUrlDefault)

	if wg.ChainCode == "SOLANA" {
		message, _ := base64.StdEncoding.DecodeString(messageStr)
		if typecode == "sign" {
			sig, err = enc.Porter().SigSol(wg, message)
			if err != nil {
				mylog.Error("type=", typecode, err)
				return txhash, sig, err
			}
			return txhash, sig, err
		}

		casttype, err := parseCallType(conf.Type)
		if err != nil {
			casttype = CallTypeGeneral
		}

		c := make([]*rpc.Client, 0)
		splitUrl := strings.Split(rpcUrlDefault, ",")
		mapUrl := make(map[string]bool)
		for _, s := range splitUrl {
			_, exi := mapUrl[s]
			if len(s) > 0 && !exi {
				c = append(c, rpc.New(s))
				mapUrl[s] = true
			}
		}

		tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(message))
		if err != nil {
			mylog.Error("TransactionFromDecoder error: ", message, " err:", err)
			return txhash, sig, err
		}

		if casttype == CallTypeJito {
			// 设置jito费用
			AddInstruction(tx, "3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT", conf.Tip, wg.Wallet)
			AddInstruction(tx, "62aKuUCZMmDiVdW6GnHn3rzHveakd2kizUPHBJiQhENk", conf.VaultTip, wg.Wallet)
		}
		// SimulateTransaction
		_, _ = SimulateTransaction(c[1], tx, conf)
		//设置优先费
		tx.Message.Instructions = appendUnitPrice(conf, tx)
		timeStart := time.Now().UnixMilli()
		hashResult, err := c[1].GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
		timeEnd := time.Now().UnixMilli() - timeStart
		mylog.Infof("EX MemeVaultHandleMessage getblock %dms", timeEnd)
		if err != nil {
			mylog.Error("Get RecentBlockhash error: ", err)
			return txhash, sig, err
		}
		mylog.Infof("Get RecentBlockhash：%s,Block: %d ", hashResult.Value.Blockhash, hashResult.Value.LastValidBlockHeight)
		tx.Message.RecentBlockhash = hashResult.Value.Blockhash

		msgBytes, _ := tx.Message.MarshalBinary()
		sig, err = enc.Porter().SigSol(wg, msgBytes)
		if err != nil {
			mylog.Error("SigSol error wg: ", wg.Wallet, " err:", err)
			return txhash, sig, err
		}

		mylog.Infof("EX Signed result sig %s %dms", base64.StdEncoding.EncodeToString(sig), time.Now().UnixMilli()-timeEnd)
		timeEnd = time.Now().UnixMilli() - timeEnd
		tx.Signatures = []solana.Signature{solana.Signature(sig)}

		//txhash, err := c.SendTransaction(context.Background(), tx)
		//txhash, status, err := SendAndConfirmTransaction(c, tx, casttype, conf.ShouldConfirm, conf.ConfirmTimeOut)
		txhash, status, err := SendAndConfirmTransactionWithClients(c, tx, casttype, conf.ShouldConfirm, conf.ConfirmTimeOut)
		mylog.Infof("EX Txhash %s, status:%s, %dms", txhash, status, time.Now().UnixMilli()-timeEnd)

		if status == "finalized" || status == "confirmed" || status == "processed" {
			mylog.Info("rpc确认状态成功201 :", status)
			mylog.Info("err:", err)
			//mylog.Info(err.Error())
			return txhash, sig, err
		}

		if err != nil {
			mylog.Info("rpc确认状态成功208 :", status)
			return txhash, sig, fmt.Errorf(err.Error()+" status:%s", status)
		} else {
			mylog.Info("rpc确认状态成功210 :", status)
			return txhash, sig, fmt.Errorf("status:%s", status)
		}
	} else { // for all evm
		message, err := hexutil.Decode(messageStr)
		if err != nil {
			return txhash, sig, err
		}
		if typecode == "sign" {
			sig, err = enc.Porter().SigEth(wg, message)
			if err != nil {
				return txhash, sig, err
			}
			return txhash, sig, err
		}
		client, _ := ethclient.Dial(rpcUrlDefault)

		nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(wg.Wallet))
		if err != nil {
			return txhash, sig, err
		}

		var gasPrice *big.Int
		if conf != nil && conf.UnitPrice != nil && conf.UnitPrice.Uint64() > 0 {
			gasPrice = conf.UnitPrice
		} else {
			gasPrice, err = client.SuggestGasPrice(context.Background())
			if err != nil {
				return txhash, sig, err
			}
		}

		value := value
		gasLimit := uint64(500000)
		if conf != nil && conf.UnitLimit != nil && conf.UnitLimit.Uint64() > 0 {
			gasLimit = conf.UnitLimit.Uint64()
		}
		tx := types.NewTransaction(nonce, common.HexToAddress(to), value, gasLimit, gasPrice, message)

		// 查询链 ID
		chainID, err := client.NetworkID(context.Background())
		if err != nil {
			return txhash, sig, err
		}

		// 对交易进行签名
		signedTx, err := enc.Porter().SigEvmTx(wg, tx, chainID)
		if err != nil {
			return txhash, sig, err
		}

		// 发送已签名的交易
		err = client.SendTransaction(context.Background(), signedTx)

		return signedTx.Hash().Hex(), sig, err
	}
}

func HandleTransfer(t *config.ChainConfig, to, mint string, amount *big.Int, wg *model.WalletGenerated, reqconf *hc.OpConfig) (txhash string, err error) {
	if len(t.GetRpc()) == 0 {
		return txhash, errors.New("rpc_config")
	}

	rpcUrlDefault := t.GetRpc()[0]
	if len(reqconf.Rpc) > 0 {
		rpcUrlDefault = reqconf.Rpc
	}
	mylog.Infof("RPC for transfer current used: %s", rpcUrlDefault)

	if wg.ChainCode == "SOLANA" {
		client := rpc.New(rpcUrlDefault)
		fromAddr := solana.MustPublicKeyFromBase58(wg.Wallet)
		toAddr := solana.MustPublicKeyFromBase58(to)
		if mint == "" || mint == "SOL" {
			transaction := solana.Transaction{
				Message: solana.Message{
					Header: solana.MessageHeader{
						NumRequiredSignatures:       1,
						NumReadonlyUnsignedAccounts: 0,
						NumReadonlySignedAccounts:   0,
					},
					RecentBlockhash: solana.Hash{},
				},
			}

			same2same := 0
			transaction.Message.AccountKeys = append(transaction.Message.AccountKeys, fromAddr)
			if fromAddr != toAddr {
				transaction.Message.AccountKeys = append(transaction.Message.AccountKeys, toAddr)
				same2same = 1
			}
			transaction.Message.AccountKeys = append(transaction.Message.AccountKeys, solana.MustPublicKeyFromBase58("11111111111111111111111111111111"))

			transferInstruction := system.NewTransferInstruction(
				amount.Uint64(),
				fromAddr,
				toAddr,
			)
			data := transferInstruction.Build()
			dData, _ := data.Data()

			compiledTransferInstruction := solana.CompiledInstruction{
				ProgramIDIndex: uint16(2),
				Accounts: []uint16{
					0,
					uint16(same2same),
				},
				Data: dData,
			}
			transaction.Message.Instructions = append(transaction.Message.Instructions, compiledTransferInstruction)

			outHash, err := client.GetLatestBlockhash(context.Background(), "")
			if err != nil {
				mylog.Error("Get RecentBlockhash error: ", err)
				return txhash, err
			}
			mylog.Infof("Get RecentBlockhash：%s,Block: %d ", outHash.Value.Blockhash, outHash.Value.LastValidBlockHeight)
			transaction.Message.RecentBlockhash = outHash.Value.Blockhash

			messageHash, _ := transaction.Message.MarshalBinary()
			sig, err := enc.Porter().SigSol(wg, messageHash)
			if err != nil {
				return txhash, err
			}
			transaction.Signatures = []solana.Signature{solana.Signature(sig)}

			txbytes, _ := transaction.MarshalBinary()
			mylog.Info(base64.StdEncoding.EncodeToString(txbytes))

			txhash, err := client.SendTransaction(context.Background(), &transaction)
			if err != nil {
				if reqconf.ShouldConfirm {
					s, err3 := waitForSOLANATransactionConfirmation(client, txhash, 500, 30)
					return s, err3
				}
			}
			return txhash.String(), err
		} else {
			fromAccount, _, _ := solana.FindAssociatedTokenAddress(fromAddr, solana.MustPublicKeyFromBase58(mint))
			toAccount, _, _ := solana.FindAssociatedTokenAddress(toAddr, solana.MustPublicKeyFromBase58(mint))

			transaction := solana.Transaction{
				Message: solana.Message{
					Header: solana.MessageHeader{
						NumRequiredSignatures:       0,
						NumReadonlyUnsignedAccounts: 0,
						NumReadonlySignedAccounts:   0,
					},
					RecentBlockhash: solana.Hash{},
				},
			}

			transaction.Message.AccountKeys = append(transaction.Message.AccountKeys,
				fromAddr,
				fromAccount,
				toAccount,
				toAddr,
				solana.MustPublicKeyFromBase58(mint),
				solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
				solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"),
				solana.MustPublicKeyFromBase58("ComputeBudget111111111111111111111111111111"),
			)

			computeUnitPrice := uint64(16000000)
			computeUnitLimit := uint32(202000) // 设置为 202,000 计算单位
			if reqconf != nil {
				if reqconf.UnitPrice != nil && reqconf.UnitPrice.Uint64() > 0 {
					computeUnitPrice = reqconf.UnitPrice.Uint64()
				}
				if reqconf.UnitLimit != nil && reqconf.UnitLimit.Uint64() > 0 {
					computeUnitLimit = uint32(reqconf.UnitLimit.Uint64())
				}
			}
			setComputeUnitPriceIx := compute_budget.SetComputeUnitPrice{computeUnitPrice}
			cuData, _ := setComputeUnitPriceIx.Build().Data()
			compiledSetComputeUnitPriceIx := solana.CompiledInstruction{
				ProgramIDIndex: 7,
				Accounts:       []uint16{},
				Data:           cuData,
			}

			setComputeUnitLimitIx := compute_budget.SetComputeUnitLimit{computeUnitLimit}
			clData, _ := setComputeUnitLimitIx.Build().Data()
			compiledSetComputeUnitLimitIx := solana.CompiledInstruction{
				ProgramIDIndex: 7,
				Accounts:       []uint16{},
				Data:           clData,
			}

			transaction.Message.Instructions = append(transaction.Message.Instructions, compiledSetComputeUnitPriceIx, compiledSetComputeUnitLimitIx)

			toAccountInfo, _ := client.GetAccountInfo(context.Background(), toAccount)

			if toAccountInfo != nil {
				ownaddr := toAccountInfo.Value.Owner.String()
				mylog.Info(ownaddr)
			}

			if toAccountInfo == nil {
				transaction.Message.AccountKeys = append(
					transaction.Message.AccountKeys,
					solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"),
				)
				createATAInstruction := associatedtokenaccount.NewCreateInstruction(
					transaction.Message.AccountKeys[0],
					toAddr,
					solana.MustPublicKeyFromBase58(mint),
				)
				data := createATAInstruction.Build()
				dData, _ := data.Data()

				compiledCreateAccountInstruction := solana.CompiledInstruction{
					ProgramIDIndex: uint16(8),
					Accounts: []uint16{
						0,
						2,
						3,
						4,
						5,
						6,
					},
					Data: dData,
				}
				transaction.Message.Instructions = append(transaction.Message.Instructions, compiledCreateAccountInstruction)
			}

			transferInstruction := token.NewTransferInstruction(
				amount.Uint64(),
				fromAccount,
				toAccount,
				fromAddr,
				nil,
			)
			data := transferInstruction.Build()
			dData, _ := data.Data()
			compiledTransferInstruction := solana.CompiledInstruction{
				ProgramIDIndex: uint16(6),
				Accounts: []uint16{
					1,
					2,
					0,
				},
				Data: dData,
			}
			transaction.Message.Instructions = append(transaction.Message.Instructions, compiledTransferInstruction)

			transaction.Message.Header.NumRequiredSignatures = 1
			transaction.Message.Header.NumReadonlyUnsignedAccounts = 0
			transaction.Message.Header.NumReadonlySignedAccounts = 0

			acs := make([]string, 0)
			for _, v := range transaction.Message.AccountKeys {
				acs = append(acs, v.String())
			}
			mylog.Info(acs)

			retryWithSameHash := false
			var outHash solana.Hash
			var sig []byte

			for retries := 0; retries < maxRetries; retries++ {
				if !retryWithSameHash {
					outHashResponse, err := client.GetLatestBlockhash(context.Background(), "")
					if err != nil {
						mylog.Errorf("Failed to get latest blockhash: %v", err)
						continue
					}
					outHash = outHashResponse.Value.Blockhash
					transaction.Message.RecentBlockhash = outHash

					messageHash, _ := transaction.Message.MarshalBinary()
					sig, err = enc.Porter().SigSol(wg, messageHash)
					if err != nil {
						return txhash, err
					}
					transaction.Signatures = []solana.Signature{solana.Signature(sig)}
				}

				txhash, err := client.SendTransaction(context.Background(), &transaction)

				if err == nil {
					txbytes, _ := transaction.MarshalBinary()
					base64tx := base64.StdEncoding.EncodeToString(txbytes)
					mylog.Infof("txhash: %s, transaction data: %s, recentBlockHash: %s", txhash.String(), base64tx, outHash.String())
					if err != nil {
						if reqconf.ShouldConfirm {
							s, err3 := waitForSOLANATransactionConfirmation(client, txhash, 500, 10)
							return s, err3
						}
					}
					return txhash.String(), err
				}

				if strings.Contains(err.Error(), "Blockhash not found") {
					mylog.Info("Blockhash not found, retrying with same blockhash and signature...")
					retryWithSameHash = true
				} else {
					// 其他错误，重置 retryWithSameHash 并重新获取 blockhash 和签名
					mylog.Errorf("Send transaction failed: %v", err)
					retryWithSameHash = false
				}

				if retries == maxRetries-1 {
					mylog.Errorf("Transaction send failed after %d attempts: %v", 5, err)
					return "", err
				}
				time.Sleep(500 * time.Millisecond)
			}
			return "", err
		}
	} else {
		supp, evm := wallet.IsSupp(wallet.ChainCode(wg.ChainCode))
		if !supp {
			return txhash, errors.New("unsupport chain")
		}
		if !evm {
			return txhash, errors.New("unsupport chain")
		}

		toAddress := common.HexToAddress(to)
		tokenAddress := common.HexToAddress(mint)

		client, _ := ethclient.Dial(rpcUrlDefault)
		if tokenAddress == (common.Address{}) {
			tx, err := sendETH(client, wg, toAddress, amount, reqconf)
			if err != nil {
				mylog.Errorf("Failed to send ETH: %v", err)
				return "", err
			}
			return tx.Hash().Hex(), nil
		} else {
			tx, err := sendERC20(client, wg, toAddress, tokenAddress, amount, reqconf)
			if err != nil {
				mylog.Errorf("Failed to send ERC20 token: %v", err)
				return "", err
			}
			return tx.Hash().Hex(), nil
		}
	}
}

// ComputeUnitLimit:2 ComputeUnitPrice:3
func appendUnitPrice(conf *hc.OpConfig, tx *solana.Transaction) []solana.CompiledInstruction {

	computeBudgetProgramIndex := ProgramIndexGetAndAppendToAccountKeys(tx, "ComputeBudget111111111111111111111111111111")
	unitPriceIndex := InstructionIndexGetAndAppendTo(tx, "ComputeBudget111111111111111111111111111111", 3)
	unitLimitIndex := InstructionIndexGetAndAppendTo(tx, "ComputeBudget111111111111111111111111111111", 2)
	okxUnitLimit := uint32(300000)
	if conf.SimulateSuccess && conf.UnitLimit.Sign() > 0 {
		okxUnitLimit = uint32(conf.UnitLimit.Uint64())
	} else if unitLimitIndex > -1 {
		ins := tx.Message.Instructions[unitLimitIndex]
		okxUnitLimit = binary.LittleEndian.Uint32(ins.Data[1:5])
		conf.UnitLimit = big.NewInt(0)
	} else {
		//todo 优先费会因为okxUnitLimit默认值太高导致price太低
	}

	// 构造 SetComputeUnitPrice 指令数据
	// 如果操作配置中指定了UnitPrice，则使用它。
	lamports := uint64(0)
	if conf.PriorityFee != nil && conf.PriorityFee.Sign() > 0 {
		newPrice := decimal.NewFromBigInt(conf.PriorityFee, 0).Sub(decimal.NewFromInt(5000)).Div(decimal.NewFromInt(int64(okxUnitLimit)))
		lamports = newPrice.Mul(decimal.NewFromInt(1000000)).BigInt().Uint64()
		log.Infof("newPrice:%v,lamports:%v", newPrice.String(), lamports)

		//microLamports = decimal.NewFromUint64(conf.UnitPrice.Uint64()).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(6))).BigInt().Uint64()
	}
	//if microLamports == 0 {
	//	// 可选：通过 RPC 获取推荐优先费
	//	prioritizationFees, err := rpcList[0].GetRecentPrioritizationFees(context.Background(), []solana.PublicKey{})
	//	if err != nil || len(prioritizationFees) == 0 {
	//
	//		microLamports = 4000000 // 默认值
	//	} else {
	//		microLamports = prioritizationFees[0].PrioritizationFee
	//
	//	}
	//}
	if unitPriceIndex > -1 && lamports <= 0 {
		ins := tx.Message.Instructions[unitPriceIndex]
		log.Info("UnitPrice no update old data:", binary.LittleEndian.Uint32(ins.Data[1:9]))
	}
	if lamports > 0 {
		log.Info("重新设置solana price", lamports)
		computeUnitPriceData := make([]byte, 9)
		computeUnitPriceData[0] = 3 // Instruction index for SetComputeUnitPrice
		binary.LittleEndian.PutUint64(computeUnitPriceData[1:], lamports)
		// 手动构造 CompiledInstruction
		unitPriceInstruction := solana.CompiledInstruction{
			ProgramIDIndex: computeBudgetProgramIndex,
			Accounts:       []uint16{}, // SetComputeUnitPrice 不需要账户
			Data:           computeUnitPriceData,
		}
		if unitPriceIndex < 0 {

			tx.Message.Instructions = append(
				//[]solana.CompiledInstruction{compiledComputeUnitPrice, compiledComputeUnitLimit},
				[]solana.CompiledInstruction{unitPriceInstruction},
				tx.Message.Instructions...,
			)
			log.Info("UnitPrice append new data:", lamports)
		} else {
			ins := tx.Message.Instructions[unitPriceIndex]
			log.Info("UnitPrice update old data:", binary.LittleEndian.Uint32(ins.Data[1:9]))
			tx.Message.Instructions[unitPriceIndex] = unitPriceInstruction

		}
		temp := tx.Message.Instructions[unitPriceIndex]
		log.Info("UnitPrice curr data:", binary.LittleEndian.Uint32(temp.Data[1:9]))
	}
	unitLimitIndex = InstructionIndexGetAndAppendTo(tx, "ComputeBudget111111111111111111111111111111", 2)

	// 2. 添加 SetComputeUnitLimit 指令
	computeUnitLimit := uint32(0) // 默认计算单元限制：200,000
	if conf.UnitLimit != nil && conf.UnitLimit.Sign() >= 0 && conf.SimulateSuccess {
		log.Info("获取conf Limit", conf.UnitLimit)
		computeUnitLimit = uint32(conf.UnitLimit.Uint64())
	}

	if unitLimitIndex > -1 && computeUnitLimit <= 0 {
		ins := tx.Message.Instructions[unitLimitIndex]
		log.Info("UnitLimit no update old data:", binary.LittleEndian.Uint32(ins.Data[1:5]))
	}
	//
	computeUnitLimit = 0
	if computeUnitLimit > 0 {
		log.Info("重新设置solana Limit", computeUnitLimit)
		computeUnitLimitData := make([]byte, 5)
		computeUnitLimitData[0] = 2 // Instruction index for SetComputeUnitLimit
		binary.LittleEndian.PutUint32(computeUnitLimitData[1:], computeUnitLimit)
		compiledComputeUnitLimit := solana.CompiledInstruction{
			ProgramIDIndex: computeBudgetProgramIndex,
			Accounts:       []uint16{}, // SetComputeUnitLimit 不需要账户
			Data:           computeUnitLimitData,
		}
		// 将指令插入到交易指令列表开头（顺序：CU Price -> CU Limit -> 其他指令）
		if unitLimitIndex < 0 {
			tx.Message.Instructions = append(
				//[]solana.CompiledInstruction{compiledComputeUnitPrice, compiledComputeUnitLimit},
				[]solana.CompiledInstruction{compiledComputeUnitLimit},
				tx.Message.Instructions...,
			)
			log.Info("UnitLimit append new data:", lamports)
		} else {
			temp := tx.Message.Instructions[unitLimitIndex]
			log.Info("UnitLimit update old data:", binary.LittleEndian.Uint32(temp.Data[1:5]))
			tx.Message.Instructions[unitLimitIndex] = compiledComputeUnitLimit
		}
		temp := tx.Message.Instructions[unitLimitIndex]
		log.Info("UnitLimit curr data:", binary.LittleEndian.Uint32(temp.Data[1:5]))
	}

	return tx.Message.Instructions
}
func SimulateTransaction(rpc1 *rpc.Client, tx *solana.Transaction, conf *hc.OpConfig) (*rpc.SimulateTransactionResponse, error) {
	fmt.Println("SimulateTransaction cnf:price: ", conf.UnitPrice, ",limit: ", conf.UnitLimit)
	hashResult, err := rpc1.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}
	tx.Message.RecentBlockhash = hashResult.Value.Blockhash
	sim, errSim := rpc1.SimulateTransaction(context.Background(), tx)

	if errSim == nil && sim != nil && sim.Value != nil && sim.Value.Err == nil {
		fmt.Println("SimulateTransaction limit :", *sim.Value.UnitsConsumed)
		conf.UnitLimit = decimal.NewFromBigInt(new(big.Int).SetUint64(*sim.Value.UnitsConsumed), 0).Mul(decimal.NewFromFloat(1.1)).BigInt()
		conf.SimulateSuccess = true
	} else {
		var strErr error
		if errSim != nil {
			strErr = errSim
		}
		var txErr interface{}
		if errSim == nil && sim != nil && sim.Value != nil {
			txErr = sim.Value.Err
		}
		var errLog string
		if errSim == nil && sim != nil && sim.Value != nil && sim.Value.Err != nil && len(sim.Value.Logs) > 3 {
			logs := sim.Value.Logs
			lastThree := logs[len(logs)-3:]
			errLog = strings.Join(lastThree, "\n")
		}
		fmt.Println("SimulateTransaction err :", strErr, ",txErr:", txErr, ",errLog:", errLog)

	}
	return sim, err
}

// /转账确认tx 状态
func waitForSOLANATransactionConfirmation(client *rpc.Client, txhash solana.Signature, milliseconds int, maxRetries int) (string, error) {
	var errInChain interface{}
	var err2 error
	var status *rpc.SignatureStatusesResult
	scheduler := gocron.NewScheduler(time.Local)
	retries := 0
	scheduler.Every(milliseconds).Millisecond().SingletonMode().LimitRunsTo(maxRetries).Tag("waitForTransferTx").Do(func() {
		retries++
		startTime := time.Now()
		resp, err := client.GetSignatureStatuses(context.Background(), true, txhash)
		err2 = err
		if err == nil && resp != nil && len(resp.Value) != 0 && resp.Value[0] != nil && resp.Value[0].ConfirmationStatus != "processed" {
			err2 = nil
			errInChain = resp.Value[0].Err
			status = resp.Value[0]
			mylog.Infof("waitForTx Transfer retries:[%d] %s (elapsed: %d ms) ,Error status:%v ", retries, txhash, time.Since(startTime).Milliseconds(), errInChain)

			_ = scheduler.RemoveByTag("waitForTransferTx")
			scheduler.Clear()
			scheduler.StopBlockingChan()
		} else {
			if resp != nil && len(resp.Value) != 0 {
				err2 = nil
				status = resp.Value[0]
			}

			mylog.Infof("waitForTx Transfer retries:[%d] %s (elapsed: %d ms) ,status unavailable yet status:%+v  ", retries, txhash, time.Since(startTime).Milliseconds(), resp)
		}
		if retries >= maxRetries {
			scheduler.Clear()
			scheduler.StopBlockingChan()
		}
	})
	scheduler.StartBlocking()
	if err2 != nil || errInChain != nil {
		return txhash.String(), fmt.Errorf("failed to confirm transaction[retries:%d]:queryERR: %v,tranfulERR: %v ,status:%v", retries, err2, errInChain, status)
	} else {
		if status != nil && status.ConfirmationStatus == "processed" {
			return txhash.String(), fmt.Errorf("failed to confirm transaction[retries:%d]:queryERR: %v,tranfulERR: %v ,status:%v", retries, err2, errInChain, status)
		}
		return txhash.String(), nil
	}
}

// 交易确认tx 状态
func waitForSOLANATransactionConfirmWithClients(rpcList []*rpc.Client, txhash solana.Signature, milliseconds int, maxRetries int) (string, error) {
	var errInChain interface{}
	var err2 error
	var status *rpc.SignatureStatusesResult

	scheduler := gocron.NewScheduler(time.Local)
	retries := 0
	mylog.Infof(" waitForTx Start  TX:%s ,clients:%+v ,Every:%d ,maxRetries:%d", txhash.String(), rpcList, milliseconds, maxRetries)
	maxRetry := 0
	_, err3 := scheduler.Every(milliseconds).Millisecond().SingletonMode().LimitRunsTo(maxRetries).Tag("waitForTx").Do(func() {
		maxRetry++
		for i, client := range rpcList {
			retries++
			startTime := time.Now()
			resp, err2 := client.GetSignatureStatuses(context.Background(), true, txhash)
			if err2 != nil {
				mylog.Infof("waitForTx [%d]retries:[%d] %s (elapsed: %d ms) Error fetching err: %v", i, retries, txhash, time.Since(startTime).Milliseconds(), err2)
			}
			if resp == nil || len(resp.Value) == 0 || resp.Value[0] == nil {
				err2 = nil
				mylog.Infof("waitForTx [%d]retries:[%d] %s (elapsed: %d ms) ,status unavailable yet ", i, retries, txhash, time.Since(startTime).Milliseconds())
			}
			if err2 == nil && resp != nil && len(resp.Value) > 0 && resp.Value[0] != nil {
				errInChain = resp.Value[0].Err
				status = resp.Value[0]
				if status.Err != nil {
					mylog.Infof("waitForTx [%d]retries:[%d] %s (elapsed: %d ms) ,Error status:%v ", i, retries, txhash, time.Since(startTime).Milliseconds(), errInChain)
				} else {
					mylog.Infof("waitForTx [%d]retries:[%d] %s (elapsed: %d ms) ,success status:%v ", i, retries, txhash, time.Since(startTime).Milliseconds(), resp.Value[0])
					err2 = nil
				}
				_ = scheduler.RemoveByTag("waitForTx")
				scheduler.Clear()
				scheduler.StopBlockingChan()
			}
		}
		if maxRetry >= maxRetries {
			scheduler.Clear()
			scheduler.StopBlockingChan()
		}
	})
	if err3 != nil {
		mylog.Errorf("waitForTx gocron error:%v", err3)
	}
	scheduler.StartBlocking()
	mylog.Infof("waitForTx end retries:[%d] %s status:%+v ,err:%v, errInChain:%v", retries, txhash, status, err2, errInChain)

	if err2 != nil || errInChain != nil || status == nil {
		return "failed", fmt.Errorf("failed to confirm transaction[retries:%d]:queryERR: %v,tranfulERR: %v", retries, err2, errInChain)
	} else {
		return string(status.ConfirmationStatus), nil
	}
}

func sendETH(client *ethclient.Client, wg *model.WalletGenerated, toAddress common.Address, amount *big.Int, reqconf *hc.OpConfig) (*types.Transaction, error) {
	fromAddress := common.HexToAddress(wg.Wallet)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, err
	}

	gasLimit := uint64(21000) // 转账ETH的固定Gas限制
	var gasPrice *big.Int
	if reqconf != nil && reqconf.UnitPrice != nil && reqconf.UnitPrice.Uint64() > 0 {
		gasPrice = reqconf.UnitPrice
	} else {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			return nil, err
		}
	}
	if reqconf != nil && reqconf.UnitLimit != nil && reqconf.UnitLimit.Uint64() > 0 {
		gasLimit = reqconf.UnitLimit.Uint64()
	}
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(nonce, toAddress, amount, gasLimit, gasPrice, nil)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	signedTx, err := enc.GetEP().SigEvmTx(wg, tx, chainID)
	//types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}

func sendERC20(client *ethclient.Client, wg *model.WalletGenerated, toAddress, tokenAddress common.Address, amount *big.Int, reqconf *hc.OpConfig) (*types.Transaction, error) {
	fromAddress := common.HexToAddress(wg.Wallet)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, err
	}

	data, err := parsedABI.Pack("transfer", toAddress, amount)
	if err != nil {
		return nil, err
	}
	gasLimit := uint64(65000) // 转账ETH的固定Gas限制
	var gasPrice *big.Int
	if reqconf != nil && reqconf.UnitPrice != nil && reqconf.UnitPrice.Uint64() > 0 {
		gasPrice = reqconf.UnitPrice
	} else {
		gasPrice, err = client.SuggestGasPrice(context.Background())

	}
	if reqconf != nil && reqconf.UnitLimit != nil && reqconf.UnitLimit.Uint64() > 0 {
		gasLimit = reqconf.UnitLimit.Uint64()
	}
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(nonce, tokenAddress, big.NewInt(0), gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	signedTx, err := enc.GetEP().SigEvmTx(wg, tx, chainID) //types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}

func SendAndConfirmTransaction(c *rpc.Client, tx *solana.Transaction, typeof CallType, needToConfirm bool, timeout time.Duration) (string, string, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var txhash solana.Signature
	var err error
	if typeof == CallTypeJito {
		//txhash, err = SendTransactionWithCtx(ctx, tx)
		txhash, err = SendTransactionWithMultipleDomains(ctx, tx)
	} else {
		txhash, err = c.SendTransaction(ctx, tx)
	}

	if err != nil {
		mylog.Errorf("[jito and general] send tx error %s, %v", typeof, err)
		return txhash.String(), "", err
	}

	sigTime := time.Now()
	txhashStr := base58.Encode(txhash[:])
	mylog.Infof("txhash:%s, sigTime:%d ms", txhashStr, sigTime.Sub(startTime).Milliseconds())

	statusChan := make(chan string, 1)
	errChan := make(chan error, 1)
	if needToConfirm {
		go func() {
			defer close(statusChan)
			status, err := waitForTransactionConfirmation(ctx, c, txhash)
			if err != nil {
				errChan <- err
				close(errChan)
				return
			}
			statusChan <- status
		}()
	} else {
		statusChan <- "confirmed"
	}

	select {
	case status := <-statusChan:
		mylog.Infof("Transaction %s status: %s", txhashStr, status)
		return txhashStr, status, nil
	case err := <-errChan:
		mylog.Infof("Transaction %s failed with error: %v", txhashStr, err)
		return txhashStr, "failed", err
	case <-ctx.Done():
		mylog.Infof("Transaction %s unpub on chain", txhashStr)
		return txhashStr, "unpub", ctx.Err()
	}
}
func SendAndConfirmTransactionWithClients(rpcList []*rpc.Client, tx *solana.Transaction, typeof CallType, needToConfirm bool, timeout time.Duration) (string, string, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var txhash solana.Signature
	var err error
	if typeof == CallTypeJito {
		//txhash, err = SendTransactionWithCtx(ctx, tx)
		//txhash, err = SendTransactionWithMultipleDomains(ctx, tx)
		//txhash, err = swapData.SendSolTxByOkxApi(ctx, tx)
		txhash, err = rpcList[0].SendTransaction(ctx, tx)
	} else {
		txhash, err = rpcList[0].SendTransaction(ctx, tx)
		//txhash, err = swapData.SendSolTxByOkxApi(ctx, tx)

	}

	if err != nil {
		mylog.Errorf("[jito and general] send tx error %s, %v", typeof, err)
		return txhash.String(), "", err
	}

	sigTime := time.Now()
	txhashStr := base58.Encode(txhash[:])
	mylog.Infof("txhash:%s, sigTime:%d ms", txhashStr, sigTime.Sub(startTime).Milliseconds())

	statusChan := make(chan string, 1)
	errChan := make(chan error, 1)
	if needToConfirm {
		go func() {
			defer close(statusChan)
			status, err := waitForSOLANATransactionConfirmWithClients(rpcList, txhash, 500, 60)
			if err != nil {
				errChan <- err
				close(errChan)
				return
			}
			statusChan <- status
		}()
	} else {
		statusChan <- "confirmed"
	}

	select {
	case status := <-statusChan:
		mylog.Infof("Transaction %s status: %s", txhashStr, status)
		return txhashStr, status, nil
	case err := <-errChan:
		mylog.Infof("Transaction %s failed with error: %v", txhashStr, err)
		return txhashStr, "failed", err
	case <-ctx.Done():
		mylog.Infof("Transaction %s unpub on chain", txhashStr)
		return txhashStr, "unpub", ctx.Err()
	}
}

func SendAndConfirmTransactionWithClientsTest(rpcList []*rpc.Client, tx *solana.Transaction, typeof CallType, needToConfirm bool, timeout time.Duration) (string, string, error) {
	mylog.Info("进入SendAndConfirmTransactionWithClientsTest")
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var txhash solana.Signature
	var err error
	if typeof == CallTypeJito {
		txhash, err = SendTransactionWithCtxTest(ctx, tx)
	} else {
		txhash, err = rpcList[0].SendTransaction(ctx, tx)
	}

	if err != nil {
		mylog.Errorf("[jito and general] send tx error %s, %v", typeof, err)
		return txhash.String(), "", err
	}

	sigTime := time.Now()
	txhashStr := base58.Encode(txhash[:])
	mylog.Infof("txhash:%s, sigTime:%d ms", txhashStr, sigTime.Sub(startTime).Milliseconds())

	statusChan := make(chan string, 1)
	errChan := make(chan error, 1)
	if needToConfirm {
		go func() {
			defer close(statusChan)
			status, err := waitForSOLANATransactionConfirmWithClients(rpcList, txhash, 500, 60)
			if err != nil {
				errChan <- err
				close(errChan)
				return
			}
			statusChan <- status
		}()
	} else {
		statusChan <- "confirmed"
	}

	select {
	case status := <-statusChan:
		mylog.Infof("Transaction %s status: %s", txhashStr, status)
		return txhashStr, status, nil
	case err := <-errChan:
		mylog.Infof("Transaction %s failed with error: %v", txhashStr, err)
		return txhashStr, "failed", err
	case <-ctx.Done():
		mylog.Infof("Transaction %s unpub on chain", txhashStr)
		return txhashStr, "unpub", ctx.Err()
	}
}

func waitForTransactionConfirmation(ctx context.Context, c *rpc.Client, txhash solana.Signature) (string, error) {

	for {
		startTime := time.Now()
		select {
		case <-ctx.Done():
			mylog.Infof("unpub reached while waiting for transaction confirmation")
			return "unpub", ctx.Err()

		case <-time.After(500 * time.Millisecond):

			resp, err := c.GetSignatureStatuses(ctx, true, txhash)
			if err != nil {
				mylog.Infof("EX Error fetching transaction status: (elapsed: %d ms) %v", time.Since(startTime).Milliseconds(), err)
				return "failed", err
			}

			if resp == nil || len(resp.Value) == 0 || resp.Value[0] == nil {
				mylog.Infof("EX Transaction %s status unavailable yet (elapsed: %d ms)", txhash, time.Since(startTime).Milliseconds())
				continue
			}

			status := resp.Value[0]
			if status.Err != nil {
				mylog.Infof("Transaction %s failed with error: %v", txhash, status.Err)
				//maxSupportedTransactionVersion := uint64(0)
				//opts := rpc.GetTransactionOpts{
				//	Encoding:                       solana.EncodingBase64,
				//	Commitment:                     rpc.CommitmentConfirmed,
				//	MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
				//}
				//txResp, err1 := c.GetTransaction(ctx, txhash, &opts)
				//if err1 == nil {
				//	decodedTx, _ := solana.TransactionFromDecoder(bin.NewBinDecoder(txResp.Transaction.GetBinary()))
				//	mylog.Infof("Transaction %s GetConfirmedTransaction: txResp:%+v, decodedTx:%+v ", txhash, txResp, decodedTx)
				//} else {
				//	mylog.Infof("Transaction %s GetConfirmedTransaction err: %+v", txhash, err1)
				//}
				return "failed", fmt.Errorf("failed with error %v", status.Err)
			}

			mylog.Infof("EX Transaction %s status: %s (elapsed: %d ms)", txhash, status.ConfirmationStatus, time.Since(startTime).Milliseconds())
			if status.ConfirmationStatus == "finalized" {
				return "finalized", nil
			}
			if status.ConfirmationStatus == "confirmed" {
				return "confirmed", nil
			}

		}
	}
}

type BlockhashRes struct {
	resp  *rpc.GetLatestBlockhashResult
	err   error
	index int
}

// GetLatestBlockhashFromMultipleClients 并发请求多个RPC客户端获取最新区块哈希
// 返回LastValidBlockHeight最大的结果
func GetLatestBlockhashFromMultipleClients(clients []*rpc.Client, commitment rpc.CommitmentType) (*rpc.GetLatestBlockhashResult, error) {
	if len(clients) == 0 {
		return nil, errors.New("no RPC clients provided")
	}

	// 设置整体超时时间（10秒）
	overallCtx, overallCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer overallCancel()

	// 用于同步所有协程
	var wg sync.WaitGroup
	// 用于保护共享数据结构
	var mu sync.Mutex
	// 收集结果
	var validResults []*rpc.GetLatestBlockhashResult
	var errorList []error

	// 启动协程并发请求所有客户端
	for i, client := range clients {
		wg.Add(1)
		go func(c *rpc.Client, index int) {
			defer wg.Done()

			// 用于存储当前协程的结果
			var result *rpc.GetLatestBlockhashResult
			var resultErr error
			var elapsed time.Duration

			// panic 恢复放在最外层，避免死锁
			defer func() {
				if r := recover(); r != nil {
					mylog.Errorf("Panic in GetLatestBlockhash goroutine %d: %v", index, r)
					// 安全地添加panic错误到错误列表
					mu.Lock()
					errorList = append(errorList, fmt.Errorf("panic occurred in client %d: %v", index, r))
					mu.Unlock()
					return
				}

				// 正常情况下处理结果（无论成功还是失败）
				mu.Lock()
				defer mu.Unlock()

				if resultErr != nil {
					mylog.Errorf("GetLatestBlockhash error from client %d (elapsed: %dms): %v",
						index, elapsed.Milliseconds(), resultErr)
					errorList = append(errorList, fmt.Errorf("client %d: %v", index, resultErr))
				} else if result != nil && result.Value != nil {
					mylog.Infof("GetLatestBlockhash success from client %d (elapsed: %dms), LastValidBlockHeight: %d",
						index, elapsed.Milliseconds(), result.Value.LastValidBlockHeight)
					validResults = append(validResults, result)
				} else {
					mylog.Warnf("GetLatestBlockhash from client %d returned nil response", index)
					errorList = append(errorList, fmt.Errorf("client %d: nil response", index))
				}
			}()

			// 为每个请求设置独立的超时时间（2秒，因为整体超时是3秒）
			requestCtx, requestCancel := context.WithTimeout(overallCtx, 400*time.Millisecond)
			defer requestCancel()

			// 执行实际的RPC请求（不持锁）
			startTime := time.Now()
			result, resultErr = c.GetLatestBlockhash(requestCtx, commitment)
			elapsed = time.Since(startTime)
		}(client, i)
	}

	// 使用channel监听WaitGroup完成或整体超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待所有协程完成或整体超时
	select {
	case <-done:
		mylog.Infof("All %d RPC requests completed", len(clients))
	case <-overallCtx.Done():
		mylog.Warnf("Overall timeout reached, some requests may still be running")
		mu.Lock()
		errorList = append(errorList, errors.New("overall timeout reached"))
		mu.Unlock()
	}

	// 加锁读取最终结果
	mu.Lock()
	defer mu.Unlock()

	// 如果没有任何有效结果，返回错误
	if len(validResults) == 0 {
		if len(errorList) > 0 {
			return nil, fmt.Errorf("all requests failed, first error: %v", errorList[0])
		}
		return nil, errors.New("no valid results received")
	}

	// 按LastValidBlockHeight从大到小排序，返回最大的一个
	sort.Slice(validResults, func(i, j int) bool {
		return validResults[i].Value.LastValidBlockHeight > validResults[j].Value.LastValidBlockHeight
	})

	slots := make([]string, 0, len(validResults))
	for _, result := range validResults {
		slots = append(slots, fmt.Sprintf("%d:%d", result.Context.Slot, result.Value.LastValidBlockHeight))
	}

	mylog.Infof("获取最新区块hash: %d from %d valid responses (total errors: %d) %s",
		validResults[0].Value.LastValidBlockHeight, len(validResults), len(errorList), strings.Join(slots, ","))

	return validResults[0], nil
}
