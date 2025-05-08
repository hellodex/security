package chain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	addressLookupTable "github.com/gagliardetto/solana-go/programs/address-lookup-table"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	solsystem "github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/hellodex/HelloSecurity/api/common"
	system "github.com/hellodex/HelloSecurity/log"
	"github.com/hellodex/HelloSecurity/wallet"
	"github.com/klauspost/compress/gzhttp"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	EVMMethodID                = []byte{0xa9, 0x05, 0x9c, 0xbb}
	defaultMaxIdleConnsPerHost = 50
	defaultTimeout             = 5 * time.Minute
	defaultKeepAlive           = 180 * time.Second
	maxVersion                 = uint64(0)
	httpOpts                   = jsonrpc.RPCClientOpts{
		HTTPClient: &http.Client{
			Timeout:   defaultTimeout,
			Transport: gzhttp.Transport(newHTTPTransport()),
		},
	}
	Handlers = map[string]DecoderHandler{
		"11111111111111111111111111111111":             handleSystemDecoder,
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA":  handleTokenDecoder,
		"ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL": handleAssociatedTokenDecoder,
	}
	Transfer           = "Transfer"
	InitializeAccount  = "InitializeAccount"
	InitializeAccount2 = "InitializeAccount2"
	InitializeAccount3 = "InitializeAccount3"
	SysTransfer        = "STransfer"
	TransferChecked    = "TransferChecked"
	idoAddrMap         = map[string]bool{
		"": true,
	}
)

type DecoderHandler func(accounts []*solana.AccountMeta, data []byte) (*DecodeInsData, error)
type DecodeInsData struct {
	Plat string
	Key  string
	Data interface{}
}

func TxParser(txHash string, rpcUrl string, chainCode string) (*common.TransferParsed, error) {
	if chainCode == string(wallet.SOLANA) {
		txSig := solana.MustSignatureFromBase58(txHash)
		client := rpc.NewWithCustomRPCClient(jsonrpc.NewClientWithOpts(rpcUrl, &httpOpts))
		out, err := client.GetTransaction(
			context.TODO(),
			txSig,
			&rpc.GetTransactionOpts{
				Encoding:                       solana.EncodingBase64,
				MaxSupportedTransactionVersion: &maxVersion,
			},
		)
		if err != nil || out.Transaction == nil {
			return nil, fmt.Errorf("get transaction,tx:%v, error:%s", out, err)
		}
		txMeta := out.Meta
		txTrans := out.Transaction
		txTran, _ := solana.TransactionFromDecoder(bin.NewBinDecoder(txTrans.GetBinary()))
		//txSender := txTran.Message.Signers()[0].String()
		// 获取 Address Lookup Table 的公共密钥
		// 获取完整的账户列表
		metas, err := txTran.Message.AccountMetaList()
		lookupTableAccounts := txTran.Message.AddressTableLookups
		table := make(map[solana.PublicKey]solana.PublicKeySlice)
		for _, lookup := range lookupTableAccounts {
			lookupTable, err := GetAddressLookupTableWithRetry(client, context.Background(), lookup.AccountKey)
			if err == nil {
				table[lookup.AccountKey] = lookupTable.Addresses
			}
			if err != nil {
				return nil, err
			}
			if len(table) > 0 {
				// 初始化地址表
				err := txTran.Message.SetAddressTables(table)
				if err != nil {
					return nil, err
				}
			}

		}
		metas, err = txTran.Message.AccountMetaList()
		if err != nil {
			return nil, err
		}
		createAccounts := make([]*DecodeInsData, 0)
		associatedTokenAccountInstructions := make([]*associatedtokenaccount.Instruction, 0)
		transfers := make([]*token.Instruction, 0)
		sysTransfers := make([]*solsystem.Instruction, 0)
		transferChecked := make([]*token.Instruction, 0)
		accountMap := make(map[string]common.TempAccountData)
		tokenDecimalsMap := processTempAcountsAndBalance(txMeta, metas, accountMap)
		for i, _ := range txTran.Message.Instructions {
			varr := txTran.Message.Instructions[i]

			programPub, _ := txTran.Message.ResolveProgramIDIndex(varr.ProgramIDIndex)
			accounts := make([]*solana.AccountMeta, 0)
			for _, v := range varr.Accounts {
				if int(v) >= len(metas) {
					return nil, errors.New("地址表超限")
				}
				accounts = append(accounts, metas[v])
			}

			//s := programPub.String()
			//fmt.Println("programPub:", s)
			result, err := ParseDecode(programPub, accounts, varr.Data)
			if err != nil {
				//Log.Println(err)
			} else {
				switch result.Key {
				case Transfer:
					transfers = append(transfers, result.Data.(*token.Instruction))
				case SysTransfer:
					sysTransfers = append(sysTransfers, result.Data.(*solsystem.Instruction))
				case TransferChecked:
					transferChecked = append(transferChecked, result.Data.(*token.Instruction))
				case InitializeAccount, InitializeAccount2, InitializeAccount3:
					createAccounts = append(createAccounts, result)
				case associatedtokenaccount.ProgramName:
					associatedTokenAccountInstructions = append(associatedTokenAccountInstructions, result.Data.(*associatedtokenaccount.Instruction))
				default:
					system.Logger.Println("未知指令:", result.Key)
				}
			}
		}
		processTempAccountMap(tokenDecimalsMap, createAccounts, accountMap, associatedTokenAccountInstructions)
		for _, t := range transfers {
			transfer := t.Impl.(*token.Transfer)
			amount := transfer.Amount
			source := transfer.GetSourceAccount().PublicKey.String()
			destination := transfer.GetDestinationAccount().PublicKey.String()
			//authority := transfer.GetOwnerAccount().PublicKey.String()
			tokenAddr := ""
			if tempAccount, ok := accountMap[source]; ok {
				source = tempAccount.Owner
				tokenAddr = tempAccount.Mint
			}
			if tempAccountBydestination, ok := accountMap[destination]; ok {
				destination = tempAccountBydestination.Owner
				tokenAddr = tempAccountBydestination.Mint
			}

			decimals := uint8(0)
			if u, ok := tokenDecimalsMap[tokenAddr]; ok {
				decimals = u
			}
			if _, ok := idoAddrMap[destination]; !ok {
				pass := common.TransferParsed{
					Tx:         txHash,
					From:       source,
					To:         destination,
					Amount:     big.NewInt(0).SetUint64(*amount),
					Contract:   tokenAddr,
					Decimals:   decimals,
					Block:      out.Slot,
					BlockTime:  uint64(*out.BlockTime),
					GasUsed:    *txMeta.ComputeUnitsConsumed,
					GasPrice:   txMeta.Fee,
					ChainCode:  chainCode,
					ParsedTime: time.Now(),
				}
				quote := common.QuotePrice(chainCode, tokenAddr)
				if quote != nil {
					pass.Symbol = quote.Symbol
					pass.Decimals = uint8(quote.Decimals)
					if quote.Price.Sign() > 0 {
						pass.Price = quote.Price
					}

				}
				return &pass, nil
			}

		}
		for _, t := range transferChecked {
			transfer := t.Impl.(*token.TransferChecked)
			amount := transfer.Amount
			source := transfer.GetSourceAccount().PublicKey.String()
			destination := transfer.GetDestinationAccount().PublicKey.String()
			//authority := transfer.GetOwnerAccount().PublicKey.String()
			tokenAddr := transfer.GetMintAccount().PublicKey.String()

			if tempAccount, ok := accountMap[source]; ok {
				source = tempAccount.Owner
			}
			if tempAccountBydestination, ok := accountMap[source]; ok {
				destination = tempAccountBydestination.Owner
			}
			decimals := *transfer.Decimals
			if _, ok := idoAddrMap[destination]; !ok {
				pass := common.TransferParsed{
					Tx:         txHash,
					From:       source,
					To:         destination,
					Amount:     big.NewInt(0).SetUint64(*amount),
					Contract:   tokenAddr,
					Decimals:   decimals,
					Block:      out.Slot,
					BlockTime:  uint64(*out.BlockTime),
					GasUsed:    *txMeta.ComputeUnitsConsumed,
					GasPrice:   txMeta.Fee,
					ChainCode:  string(wallet.SOLANA),
					ParsedTime: time.Now(),
				}
				quote := common.QuotePrice(chainCode, tokenAddr)
				if quote != nil {
					pass.Symbol = quote.Symbol
					if quote.Price.Sign() > 0 {
						pass.Price = quote.Price
					}

				}
				return &pass, nil
			}

		}
		for _, s := range sysTransfers {
			transfer := s.Impl.(*solsystem.Transfer)
			amount := transfer.Lamports
			from := transfer.AccountMetaSlice[0].PublicKey.String()
			to := transfer.AccountMetaSlice[1].PublicKey.String()
			decimals := uint8(9)
			//authority := transfer.GetOwnerAccount().PublicKey.String()
			tokenAddr := ""
			if _, ok := idoAddrMap[to]; !ok {
				pass := common.TransferParsed{
					Tx:         txHash,
					From:       from,
					To:         to,
					Amount:     big.NewInt(0).SetUint64(*amount),
					Contract:   tokenAddr,
					Decimals:   decimals,
					Block:      out.Slot,
					BlockTime:  uint64(*out.BlockTime),
					GasUsed:    *txMeta.ComputeUnitsConsumed,
					GasPrice:   txMeta.Fee,
					ChainCode:  string(wallet.SOLANA),
					ParsedTime: time.Now(),
				}
				quote := common.QuotePrice(chainCode, tokenAddr)
				if quote != nil {
					pass.Symbol = quote.Symbol
					pass.Decimals = uint8(quote.Decimals)
					if quote.Price.Sign() > 0 {
						pass.Price = quote.Price
					}

				}
				return &pass, nil
			}

		}

		return nil, errors.New("解析失败")
	} else {
		dial, err := ethclient.Dial(rpcUrl)
		if err != nil {
			return nil, err
		}

		tx, pending, err := dial.TransactionByHash(context.Background(), ethcommon.HexToHash(txHash))
		fmt.Println("Pending: ", pending)
		if tx == nil || err != nil || pending {
			return nil, fmt.Errorf("交易不存在或未确认,tx:%v,pending:%v,err:%v", tx, pending, err)
		}
		from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
		to := tx.To()
		value := tx.Value()
		data := tx.Data()
		var pass common.TransferParsed
		pass.ChainCode = chainCode
		pass.ParsedTime = time.Now()
		// 普通交易
		if value.Cmp(big.NewInt(0)) > 0 && len(data) == 0 {
			mToken, b := common.MainnetToken(chainCode)
			if b && to != nil {
				pass.Tx = txHash
				pass.From = strings.ToLower(from.Hex())
				pass.To = strings.ToLower(to.Hex())
				pass.Amount = value
				pass.Contract = ""
				pass.Symbol = mToken.Symbol
				pass.Decimals = uint8(mToken.Decimals)
				receipt, err := dial.TransactionReceipt(context.Background(), ethcommon.HexToHash(txHash))

				// 或者区块, 区块时间, 交易gasUsed, 交易gasPrice
				if err == nil {
					pass.Block = receipt.BlockNumber.Uint64()
					// 区块时间
					number, err := dial.HeaderByNumber(context.Background(), receipt.BlockNumber)
					if err == nil {
						pass.BlockTime = number.Time
					}
					pass.GasUsed = receipt.GasUsed
					pass.GasPrice = tx.GasPrice().Uint64()
				}
				quote := common.QuotePrice(chainCode, "")
				if quote != nil && quote.Price.Sign() > 0 {
					pass.Price = quote.Price
				}
				return &pass, nil
			}
			if to == nil {
				return nil, fmt.Errorf("可能是合约创建交易 tx:%s", txHash)
			}
		}
		if len(data) >= 4 && to != nil {
			index := bytes.Index(data[:4], EVMMethodID)
			if len(data) >= 68 && index >= 0 {
				fmt.Println("Type: ERC-20 交易")
				fmt.Printf("合约地址: %s\n", to.Hex())
				fmt.Printf("From: %s\n", from.Hex())

				// Parse 'to' address (bytes 12-44, skip 12 bytes of padding)
				tokenTo := ethcommon.BytesToAddress(data[4:36])
				fmt.Printf("To: %s\n", tokenTo.Hex())

				// Parse amount (bytes 44-76)
				tokenAmount := new(big.Int).SetBytes(data[36:68])
				fmt.Printf("Token Amount: %s\n", tokenAmount.String())
				pass.Tx = txHash
				pass.From = strings.ToLower(from.Hex())
				pass.To = strings.ToLower(tokenTo.Hex())
				pass.Amount = tokenAmount
				pass.Contract = strings.ToLower(to.Hex())
				receipt, err := dial.TransactionReceipt(context.Background(), ethcommon.HexToHash(txHash))
				// 获得区块, 区块时间, 交易gasUsed, 交易gasPrice
				if err == nil {
					pass.Block = receipt.BlockNumber.Uint64()
					// 区块时间
					number, err := dial.HeaderByNumber(context.Background(), receipt.BlockNumber)
					if err == nil {
						pass.BlockTime = number.Time
					}
					pass.GasUsed = receipt.GasUsed
					pass.GasPrice = tx.GasPrice().Uint64()
				}
				// 价格
				quote := common.QuotePrice(chainCode, strings.ToLower(to.Hex()))
				if quote != nil {
					pass.Symbol = quote.Symbol
					pass.Decimals = uint8(quote.Decimals)
					if quote.Price.Sign() > 0 {
						pass.Price = quote.Price
					}

				}
				return &pass, nil
			}

		}
		return nil, err
	}

}

func newHTTPTransport() *http.Transport {
	return &http.Transport{
		IdleConnTimeout:     defaultTimeout,
		MaxConnsPerHost:     defaultMaxIdleConnsPerHost,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultTimeout,
			KeepAlive: defaultKeepAlive,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2: true,
		// MaxIdleConns:          100,
		TLSHandshakeTimeout: 10 * time.Second,
		// ExpectContinueTimeout: 1 * time.Second,
	}
}
func GetAddressLookupTableWithRetry(
	client *rpc.Client,
	ctx context.Context,
	address solana.PublicKey,
) (*addressLookupTable.AddressLookupTableState, error) {
	start := time.Now().UnixMilli()
	RetryCount := 0
	defer func() {
		timeConsuming := time.Now().UnixMilli() - start
		if timeConsuming > 5000 {
			system.Logger.Infof("耗时1 GetAddressLookupTableWithRetry:%dms,次数:%d, 参数: %s", timeConsuming, RetryCount, address.String())
		}
	}()
	const maxRetries = 3
	var account *rpc.GetAccountInfoResult
	var err error
	for i := 0; i < maxRetries; i++ {
		RetryCount++
		account, err = client.GetAccountInfo(ctx, address)
		if err == nil && account != nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
		}
	}
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}
	return addressLookupTable.DecodeAddressLookupTableState(account.GetBinary())
}
func processTempAcountsAndBalance(txMeta *rpc.TransactionMeta, metas solana.AccountMetaSlice,
	accountMap map[string]common.TempAccountData) (
	tokenDecimalsMap map[string]uint8) {

	tokenDecimalsMap = make(map[string]uint8)

	//sol Decimals 总是为9
	tokenDecimalsMap["So11111111111111111111111111111111111111112"] = 9
	tokenDecimalsMap["SOL"] = 9
	tokenDecimalsMap["11111111111111111111111111111111"] = 9

	//交易之前
	for _, balance := range txMeta.PreTokenBalances {
		sAccount := metas[balance.AccountIndex].PublicKey.String()
		_, exist := accountMap[sAccount]
		owner := balance.Owner.String()
		tokenAddr := balance.Mint.String()
		decimals := balance.UiTokenAmount.Decimals
		tokenDecimalsMap[tokenAddr] = decimals
		//组合spl account 和spltoken 和 owner 的映射
		if !exist {
			accountMap[sAccount] = common.TempAccountData{
				Account: sAccount,
				Mint:    tokenAddr,
				Owner:   owner,
			}
		}
	}
	//交易之后余额
	for _, balance := range txMeta.PostTokenBalances {
		sAccount := metas[balance.AccountIndex].PublicKey.String()
		_, exist := accountMap[sAccount]
		owner := balance.Owner.String()
		tokenAddr := balance.Mint.String()
		decimals := balance.UiTokenAmount.Decimals
		tokenDecimalsMap[tokenAddr] = decimals
		if !exist {
			accountMap[sAccount] = common.TempAccountData{
				Account: sAccount,
				Mint:    tokenAddr,
				Owner:   owner,
			}
		}

	}

	return tokenDecimalsMap
}
func handleSystemDecoder(accounts []*solana.AccountMeta, data []byte) (*DecodeInsData, error) {
	sysInstru, err := solsystem.DecodeInstruction(accounts, data)
	if err != nil {
		return nil, err
	}

	typeName := solsystem.InstructionIDToName(sysInstru.TypeID.Uint32())
	if typeName == "Transfer" {
		typeName = "STransfer"
	}
	return &DecodeInsData{
		Key:  typeName,
		Data: sysInstru,
	}, nil
}
func handleTokenDecoder(accounts []*solana.AccountMeta, data []byte) (*DecodeInsData, error) {
	instru, err := token.DecodeInstruction(accounts, data)
	if err != nil {
		return nil, err
	}

	typeName := token.InstructionIDToName(instru.TypeID.Uint8())

	return &DecodeInsData{
		Key:  typeName,
		Data: instru,
	}, nil
}
func handleAssociatedTokenDecoder(accounts []*solana.AccountMeta, data []byte) (*DecodeInsData, error) {
	instru, err := associatedtokenaccount.DecodeInstruction(accounts, data)
	if err != nil {
		return nil, err
	}

	typeName := associatedtokenaccount.ProgramName

	return &DecodeInsData{
		Key:  typeName,
		Data: instru,
	}, nil
}
func ParseDecode(program solana.PublicKey, accounts []*solana.AccountMeta, data []byte) (*DecodeInsData, error) {
	//spew.Dump(fmt.Sprintf("program: %s", program.String()))
	if handler, exists := Handlers[program.String()]; exists && handler != nil {
		return handler(accounts, data)
	}
	return nil, errors.New("no decoder found")
}
func processTempAccountMap(tokenDecimalsMap map[string]uint8, createAccounts []*DecodeInsData,
	accountMap map[string]common.TempAccountData, associatedTokenaccountS []*associatedtokenaccount.Instruction) {
	//收集临时账户
	for _, c := range createAccounts {
		key := c.Key
		accountIns := c.Data.(*token.Instruction)
		switch key {
		case InitializeAccount:
			accountImpl := accountIns.Impl.(*token.InitializeAccount)
			account3 := accountImpl.AccountMetaSlice[0].PublicKey.String()
			_, exist := accountMap[account3]
			if !exist {
				mint := accountImpl.GetAccounts()[1].PublicKey.String()
				aaccount := common.TempAccountData{
					Account: account3,
					Mint:    accountImpl.GetMintAccount().PublicKey.String(),
					Owner:   accountImpl.GetOwnerAccount().PublicKey.String(),
				}
				u, exis := tokenDecimalsMap[mint]
				if exis {
					aaccount.Decimals = u
				}

				accountMap[account3] = aaccount
			}

		case InitializeAccount2:
			accountImpl := accountIns.Impl.(*token.InitializeAccount2)
			account3 := accountImpl.AccountMetaSlice[0].PublicKey.String()
			_, exist := accountMap[account3]
			if !exist {
				mint := accountImpl.GetAccounts()[1].PublicKey.String()
				aaccount := common.TempAccountData{
					Account: account3,
					Mint:    accountImpl.GetMintAccount().PublicKey.String(),
					Owner:   accountImpl.Owner.String(),
				}
				u, exis := tokenDecimalsMap[mint]
				if exis {
					aaccount.Decimals = u
				}

				accountMap[account3] = aaccount
			}
		case InitializeAccount3:
			accountImpl := accountIns.Impl.(*token.InitializeAccount3)
			account3 := accountImpl.AccountMetaSlice[0].PublicKey.String()
			_, exist := accountMap[account3]
			if !exist {
				mint := accountImpl.GetAccounts()[1].PublicKey.String()
				aaccount := common.TempAccountData{
					Account: account3,
					Mint:    accountImpl.GetMintAccount().PublicKey.String(),
					Owner:   accountImpl.Owner.String(),
				}
				u, exis := tokenDecimalsMap[mint]
				if exis {
					aaccount.Decimals = u
				}

				accountMap[account3] = aaccount
			}
		default:
			continue
		}
	}
	for _, tokenAccount := range associatedTokenaccountS {
		accounts := tokenAccount.Accounts()
		source := accounts[0].PublicKey.String()
		account := accounts[1].PublicKey.String()
		owner := accounts[2].PublicKey.String()
		mint := accounts[3].PublicKey.String()
		_, exist := accountMap[source]
		if !exist && mint == "So11111111111111111111111111111111111111112" {
			accountMap[source] = common.TempAccountData{
				Account:  source,
				Mint:     mint,
				Owner:    owner,
				Decimals: 9,
			}
		}
		_, exist = accountMap[account]
		if !exist && mint == "So11111111111111111111111111111111111111112" {
			accountMap[account] = common.TempAccountData{
				Account:  source,
				Mint:     mint,
				Owner:    owner,
				Decimals: 9,
			}
		}
	}

}
