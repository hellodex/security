package chain

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

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

const maxRetries = 30

var ZERO = big.NewInt(0)

const fixedTestAddr = "KERxu1WdAfziZbmRkZnpj7mUgyJrLGdYC7d1VMwPR25"

var transferFnSignature = []byte("transfer(address,uint256)")

const erc20ABI = `[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

func HandleMessage(t *config.ChainConfig, messageStr string, to string, typecode string,
	value *big.Int,
	conf *hc.OpConfig,
	wg *model.WalletGenerated,
	needConfirm bool,
) (txhash string, sig []byte, err error) {
	if len(t.GetRpc()) == 0 {
		return txhash, sig, errors.New("rpc_config")
	}
	rpcUrlDefault := t.GetRpc()[0]
	if len(conf.Rpc) > 0 {
		rpcUrlDefault = conf.Rpc
	}
	log.Infof("RPC for transaction current used: %s", rpcUrlDefault)

	if wg.ChainCode == "SOLANA" {
		message, _ := base64.StdEncoding.DecodeString(messageStr)
		if typecode == "sign" {
			sig, err = enc.Porter().SigSol(wg, message)
			if err != nil {
				log.Error("type=", typecode, err)
				return txhash, sig, err
			}
			return txhash, sig, err
		}

		casttype, err := parseCallType(conf.Type)
		if err != nil {
			casttype = CallTypeGeneral
		}

		c := rpc.New(rpcUrlDefault)

		tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(message))
		if err != nil {
			return txhash, sig, err
		}

		// if wg.Wallet == fixedTestAddr {
		// 	casttype = CallTypeJito
		// }

		var tipAdd string
		var sepdr = solana.MustPublicKeyFromBase58(wg.Wallet)
		if casttype == CallTypeJito {
			tipAdd, err = getTipAccounts()
			log.Infof("[jito]fetch account response %v, %v", tipAdd, err)
			if err != nil {
				return txhash, sig, err
			}

			log.Infof("[jito] request %v", conf)
			if len(tipAdd) > 0 {
				tipAcc, err := solana.PublicKeyFromBase58(tipAdd)
				if err != nil {
					log.Errorf("[jito]unparsed data %s %v", tipAdd, err)
				} else if conf.Tip.Cmp(ZERO) == 1 {
					var numSigs = tx.Message.Header.NumRequiredSignatures
					var numRSig = tx.Message.Header.NumReadonlySignedAccounts
					var numRUSig = tx.Message.Header.NumReadonlyUnsignedAccounts
					log.Infof("[jito] tx header summary %d %d %d", numSigs, numRSig, numRUSig)
					programIDIndex := uint16(0)
					foundSystem := false
					for i, acc := range tx.Message.AccountKeys {
						if acc.Equals(system.ProgramID) {
							programIDIndex = uint16(i)
							foundSystem = true
							break
						}
					}
					if !foundSystem {
						log.Info("[jito]reset system program id")
						tx.Message.AccountKeys = append(tx.Message.AccountKeys, system.ProgramID)
						programIDIndex = uint16(len(tx.Message.AccountKeys) - 1)
					}

					writableStartIndex := int(tx.Message.Header.NumRequiredSignatures)
					// writableEndIndex := len(tx.Message.AccountKeys) - int(tx.Message.Header.NumReadonlyUnsignedAccounts)

					// tx.Message.AccountKeys = append(tx.Message.AccountKeys, tipAcc)
					preBoxes := append([]solana.PublicKey{}, tx.Message.AccountKeys[:writableStartIndex]...)
					postBoxes := append([]solana.PublicKey{}, tx.Message.AccountKeys[writableStartIndex:]...)
					tx.Message.AccountKeys = append(
						append(preBoxes, tipAcc),
						postBoxes...,
					)

					log.Infof("[jito] program index %d, %d", programIDIndex, writableStartIndex)

					transferInstruction := system.NewTransferInstruction(
						conf.Tip.Uint64(),
						sepdr,
						tipAcc,
					)
					data := transferInstruction.Build()
					dData, _ := data.Data()
					if programIDIndex >= uint16(writableStartIndex) {
						programIDIndex += uint16(1)
					}

					compiledTransferInstruction := solana.CompiledInstruction{
						ProgramIDIndex: programIDIndex,
						Accounts: []uint16{
							0,
							uint16(writableStartIndex),
						},
						Data: dData,
					}
					tx.Message.Instructions = append(tx.Message.Instructions, compiledTransferInstruction)

					updateInstructionIndexes(tx, writableStartIndex)
				}
			}
		}

		timeStart := time.Now().UnixMilli()
		hashResult, err := c.GetLatestBlockhash(context.Background(), "")
		timeEnd := time.Now().UnixMilli() - timeStart
		log.Infof("EX getblock %dms", timeEnd)
		if err != nil {
			log.Error("Get block hash error: ", err)
			return txhash, sig, err
		}
		tx.Message.RecentBlockhash = hashResult.Value.Blockhash

		msgBytes, _ := tx.Message.MarshalBinary()
		sig, err = enc.Porter().SigSol(wg, msgBytes)
		if err != nil {
			return txhash, sig, err
		}

		log.Infof("EX Signed result sig %s %dms", base64.StdEncoding.EncodeToString(sig), time.Now().UnixMilli()-timeEnd)
		timeEnd = time.Now().UnixMilli() - timeEnd
		tx.Signatures = []solana.Signature{solana.Signature(sig)}

		//txhash, err := c.SendTransaction(context.Background(), tx)
		txhash, status, err := SendAndConfirmTransaction(c, tx, casttype, needConfirm)
		log.Infof("EX Txhash %s, status:%s, %dms", txhash, status, time.Now().UnixMilli()-timeEnd)

		if status == "finalized" || status == "confirmed" {
			return txhash, sig, err
		}

		if err != nil {
			return txhash, sig, fmt.Errorf(err.Error()+" status:%s", status)
		} else {
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
	log.Infof("RPC for transfer current used: %s", rpcUrlDefault)

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
				log.Error("Get block hash error: ", err)
				return txhash, err
			}
			transaction.Message.RecentBlockhash = outHash.Value.Blockhash

			messageHash, _ := transaction.Message.MarshalBinary()
			sig, err := enc.Porter().SigSol(wg, messageHash)
			if err != nil {
				return txhash, err
			}
			transaction.Signatures = []solana.Signature{solana.Signature(sig)}

			txbytes, _ := transaction.MarshalBinary()
			log.Info(base64.StdEncoding.EncodeToString(txbytes))

			txhash, err := client.SendTransaction(context.Background(), &transaction)
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
				log.Info(ownaddr)
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
			log.Info(acs)

			retryWithSameHash := false
			var outHash solana.Hash
			var sig []byte

			for retries := 0; retries < maxRetries; retries++ {
				if !retryWithSameHash {
					outHashResponse, err := client.GetLatestBlockhash(context.Background(), "")
					if err != nil {
						log.Errorf("Failed to get latest blockhash: %v", err)
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
					log.Infof("txhash: %s, transaction data: %s, recentBlockHash: %s", txhash.String(), base64tx, outHash.String())
					return txhash.String(), err
				}

				if strings.Contains(err.Error(), "Blockhash not found") {
					log.Info("Blockhash not found, retrying with same blockhash and signature...")
					retryWithSameHash = true
				} else {
					// 其他错误，重置 retryWithSameHash 并重新获取 blockhash 和签名
					log.Errorf("Send transaction failed: %v", err)
					retryWithSameHash = false
				}

				if retries == maxRetries-1 {
					log.Errorf("Transaction send failed after %d attempts: %v", 5, err)
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
				log.Errorf("Failed to send ETH: %v", err)
				return "", err
			}
			return tx.Hash().Hex(), nil
		} else {
			tx, err := sendERC20(client, wg, toAddress, tokenAddress, amount, reqconf)
			if err != nil {
				log.Errorf("Failed to send ERC20 token: %v", err)
				return "", err
			}
			return tx.Hash().Hex(), nil
		}
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

func SendAndConfirmTransaction(c *rpc.Client, tx *solana.Transaction, typeof CallType, needToConfirm bool) (string, string, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var txhash solana.Signature
	var err error
	if typeof == CallTypeJito {
		txhash, err = SendTransactionWithCtx(ctx, tx)
	} else {
		txhash, err = c.SendTransaction(ctx, tx)
	}

	if err != nil {
		log.Errorf("[jito and general] send tx error %s, %v", typeof, err)
		return txhash.String(), "", err
	}

	sigTime := time.Now()
	txhashStr := base58.Encode(txhash[:])
	log.Infof("txhash:%s, sigTime:%d ms", txhashStr, sigTime.Sub(startTime).Milliseconds())

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
		log.Infof("Transaction %s status: %s", txhashStr, status)
		return txhashStr, status, nil
	case err := <-errChan:
		log.Infof("Transaction %s failed with error: %v", txhashStr, err)
		return txhashStr, "failed", err
	case <-ctx.Done():
		log.Infof("Transaction %s unpub on chain", txhashStr)
		return txhashStr, "unpub", ctx.Err()
	}
}

func waitForTransactionConfirmation(ctx context.Context, c *rpc.Client, txhash solana.Signature) (string, error) {

	for {
		startTime := time.Now()
		select {
		case <-ctx.Done():
			log.Infof("unpub reached while waiting for transaction confirmation")
			return "unpub", ctx.Err()

		case <-time.After(500 * time.Millisecond):

			resp, err := c.GetSignatureStatuses(ctx, true, txhash)
			if err != nil {
				log.Infof("EX Error fetching transaction status: (elapsed: %d ms) %v", time.Since(startTime).Milliseconds(), err)
				return "failed", err
			}

			if resp == nil || len(resp.Value) == 0 || resp.Value[0] == nil {
				log.Infof("EX Transaction %s status unavailable yet (elapsed: %d ms)", txhash, time.Since(startTime).Milliseconds())
				continue
			}

			status := resp.Value[0]
			if status.Err != nil {
				log.Infof("Transaction %s failed with error: %v", txhash, status.Err)
				//maxSupportedTransactionVersion := uint64(0)
				//opts := rpc.GetTransactionOpts{
				//	Encoding:                       solana.EncodingBase64,
				//	Commitment:                     rpc.CommitmentConfirmed,
				//	MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
				//}
				//txResp, err1 := c.GetTransaction(ctx, txhash, &opts)
				//if err1 == nil {
				//	decodedTx, _ := solana.TransactionFromDecoder(bin.NewBinDecoder(txResp.Transaction.GetBinary()))
				//	log.Infof("Transaction %s GetConfirmedTransaction: txResp:%+v, decodedTx:%+v ", txhash, txResp, decodedTx)
				//} else {
				//	log.Infof("Transaction %s GetConfirmedTransaction err: %+v", txhash, err1)
				//}
				return "failed", fmt.Errorf("failed with error %v", status.Err)
			}

			log.Infof("EX Transaction %s status: %s (elapsed: %d ms)", txhash, status.ConfirmationStatus, time.Since(startTime).Milliseconds())
			if status.ConfirmationStatus == "finalized" {
				return "finalized", nil
			}
			if status.ConfirmationStatus == "confirmed" {
				return "confirmed", nil
			}

		}
	}
}
