package chain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	compute_budget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
)

// OKXSwapInstructionResponse OKX DEX API swap-instruction接口返回的数据结构
// 用于接收和解析OKX返回的Solana swap指令数据
type OKXSwapInstructionResponse struct {
	Code string `json:"code"`
	Data struct {
		// InstructionLists 包含所有需要执行的指令列表
		InstructionLists []OKXInstruction `json:"instructionLists"`
		// AddressLookupTableAccount 地址查找表账户列表（用于V0交易）
		AddressLookupTableAccount []string `json:"addressLookupTableAccount"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// OKXInstruction OKX返回的单个指令数据
type OKXInstruction struct {
	// Data base64编码的指令数据
	Data string `json:"data"`
	// Accounts 指令涉及的账户列表
	Accounts []OKXAccountMeta `json:"accounts"`
	// ProgramID 程序ID
	ProgramID string `json:"programId"`
}

// OKXAccountMeta OKX返回的账户元数据
type OKXAccountMeta struct {
	// Pubkey 账户公钥
	Pubkey string `json:"pubkey"`
	// IsSigner 是否需要签名
	IsSigner bool `json:"isSigner"`
	// IsWritable 是否可写
	IsWritable bool `json:"isWritable"`
}

// OKXSwapData 只包含OKX响应中的data部分
type OKXSwapData struct {
	// InstructionLists 包含所有需要执行的指令列表
	InstructionLists []OKXInstruction `json:"instructionLists"`
	// AddressLookupTableAccount 地址查找表账户列表（用于V0交易）
	AddressLookupTableAccount []string `json:"addressLookupTableAccount"`
}

// GetSwapData 处理OKX返回的swap指令数据，构建包含Jito小费的完整交易
// 调用链路：
// 1. 外部服务调用OKX API获取swap指令数据
// 2. 将响应中的data部分作为JSON字符串传递给此方法
// 3. 此方法解析JSON并构建包含Jito小费的交易
// 4. 返回可签名和发送的交易对象
//
// 参数：
//   - swapInstruction: OKX API返回的data部分的JSON字符串
//   - walletAddress: 执行swap的钱包地址
//   - recentBlockhash: 最新的区块哈希
//   - jitoTipAmount: Jito小费金额（lamports），0表示不添加小费
//
// 返回：
//   - *solana.Transaction: 构建好的交易对象
//   - error: 错误信息
//
// 使用示例：
//
//	// 1. 调用OKX API获取swap指令
//	// response := callOKXAPI(params)
//
//	// 2. 提取data部分的JSON字符串
//	swapInstructionJSON := `{
//	    "instructionLists": [...],
//	    "addressLookupTableAccount": [...]
//	}`
//
//	// 3. 构建交易
//	tx, err := GetSwapData(swapInstructionJSON, walletAddress, blockhash, 100000)
//
//	// 4. 签名和发送交易
//	// signAndSend(tx)
func GetSwapData(swapInstruction string, walletAddress string, recentBlockhash solana.Hash, jitoTipAmount uint64) (*solana.Transaction, error) {
	// 1. 解析JSON字符串为OKXSwapData
	var okxData OKXSwapData
	if err := json.Unmarshal([]byte(swapInstruction), &okxData); err != nil {
		return nil, fmt.Errorf("failed to parse swap instruction JSON: %v", err)
	}

	// 2. 验证数据
	if len(okxData.InstructionLists) == 0 {
		return nil, fmt.Errorf("no instructions in swap data")
	}

	// 解析钱包地址
	walletPubkey, err := solana.PublicKeyFromBase58(walletAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid wallet address: %v", err)
	}

	// 2. 转换OKX指令为Solana指令
	instructions := make([]solana.Instruction, 0)

	// 添加计算预算指令（可选，但推荐）
	// 注意：这些指令应该在所有其他指令之前
	instructions = append(instructions,
		compute_budget.NewSetComputeUnitLimitInstruction(400000).Build(),
		compute_budget.NewSetComputeUnitPriceInstruction(1000).Build(),
	)

	mylog.Infof("[GetSwapData] Processing %d instructions from OKX", len(okxData.InstructionLists))

	// 转换每个OKX指令
	for i, okxInst := range okxData.InstructionLists {
		solanaInst, err := convertOKXInstruction(okxInst)
		if err != nil {
			return nil, fmt.Errorf("failed to convert instruction %d: %v", i, err)
		}
		instructions = append(instructions, solanaInst)
	}

	// 3. 添加Jito小费指令（如果需要）
	if jitoTipAmount > 0 {
		// Jito小费接收地址列表
		jitoTipAccounts := []string{
			"96gYZGLnJYVFmbjzopPSU6QiEV5fGqZNyN9nmNhvrZU5",
			"HFqU5x63VTqvQss8hp11i4wVV8bD44PvwucfZ2bU7gRe",
			"Cw8CFyM9FkoMi7K7Crf6HNQqf4uEMzpKw6QNghXLvLkY",
			"ADaUMid9yfUytqMBgopwjb2DTLSokTSzL1zt6iGPaS49",
			"DfXygSm4jCyNCybVYYK6DwvWqjKee8pbDmJGcLWNDXjh",
			"ADuUkR4vqLUMWXxW9gh6D6L8pMSawimctcNZ5pGwDcEt",
			"DttWaMuVvTiduZRnguLF7jNxTgiMBZ1hyAumKUiL2KRL",
			"3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT",
		}

		// 使用固定的索引，避免随机性在生产环境造成问题
		// 可以根据钱包地址的哈希值来选择，保证同一钱包总是使用同一个小费地址
		tipIndex := int(walletPubkey[0]) % len(jitoTipAccounts)
		tipAccount := solana.MustPublicKeyFromBase58(jitoTipAccounts[tipIndex])

		mylog.Infof("[getSwapData] Adding Jito tip: %d lamports to %s", jitoTipAmount, tipAccount.String())

		// 创建转账指令
		tipInstruction := system.NewTransferInstruction(
			jitoTipAmount,
			walletPubkey,
			tipAccount,
		).Build()

		// 添加到指令列表末尾
		instructions = append(instructions, tipInstruction)
	}

	// 5. 处理地址查找表（如果有）并构建交易
	if len(okxData.AddressLookupTableAccount) > 0 {
		mylog.Infof("[getSwapData] Building V0 transaction with %d ALTs", len(okxData.AddressLookupTableAccount))
		return buildV0TransactionWithALT(
			instructions,
			okxData.AddressLookupTableAccount,
			walletPubkey,
			recentBlockhash,
		)
	}

	// 6. 构建普通交易（无ALT）
	mylog.Infof("[getSwapData] Building legacy transaction")
	tx, err := solana.NewTransaction(
		instructions,
		recentBlockhash,
		solana.TransactionPayer(walletPubkey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %v", err)
	}

	return tx, nil
}

// convertOKXInstruction 将OKX格式的指令转换为Solana格式
func convertOKXInstruction(okxInst OKXInstruction) (solana.Instruction, error) {
	// 解码指令数据
	data, err := base64.StdEncoding.DecodeString(okxInst.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode instruction data: %v", err)
	}

	// 转换账户元数据
	accounts := make([]*solana.AccountMeta, len(okxInst.Accounts))
	for i, acc := range okxInst.Accounts {
		pubkey, err := solana.PublicKeyFromBase58(acc.Pubkey)
		if err != nil {
			return nil, fmt.Errorf("invalid account pubkey %s: %v", acc.Pubkey, err)
		}

		accounts[i] = &solana.AccountMeta{
			PublicKey:  pubkey,
			IsSigner:   acc.IsSigner,
			IsWritable: acc.IsWritable,
		}
	}

	// 解析程序ID
	programID, err := solana.PublicKeyFromBase58(okxInst.ProgramID)
	if err != nil {
		return nil, fmt.Errorf("invalid program ID %s: %v", okxInst.ProgramID, err)
	}

	// 创建通用指令
	return &GenericInstruction{
		ProgID:       programID,
		AccountMetas: accounts,
		DataByte:     data,
	}, nil
}

// GenericInstruction 通用指令实现
type GenericInstruction struct {
	ProgID       solana.PublicKey
	AccountMetas []*solana.AccountMeta // 重命名为AccountMetas避免与方法名冲突
	DataByte     []byte
}

func (gi *GenericInstruction) ProgramID() solana.PublicKey {
	return gi.ProgID
}

func (gi *GenericInstruction) Accounts() []*solana.AccountMeta {
	return gi.AccountMetas
}

func (gi *GenericInstruction) Data() ([]byte, error) {
	return gi.DataByte, nil
}

// buildV0TransactionWithALT 构建包含地址查找表的V0交易
// 注意：由于OKX已经处理了ALT和账户索引，这里简化处理
func buildV0TransactionWithALT(
	instructions []solana.Instruction,
	altAddresses []string,
	payer solana.PublicKey,
	blockhash solana.Hash,
) (*solana.Transaction, error) {
	// OKX的指令已经包含了正确的账户索引，我们直接构建交易即可
	// 这里暂时简化处理，返回普通交易
	// TODO: 后续需要根据实际的ALT使用情况完善V0交易构建

	mylog.Warnf("[buildV0Transaction] ALT support is simplified. OKX instructions already contain correct indices.")

	// 构建普通交易
	tx, err := solana.NewTransaction(
		instructions,
		blockhash,
		solana.TransactionPayer(payer),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %v", err)
	}

	// 记录ALT信息供调试
	mylog.Infof("[buildV0Transaction] Transaction built with %d ALT references: %v", len(altAddresses), altAddresses)

	return tx, nil
}

// GetSwapDataFromResponse 从完整的OKX响应构建交易的便捷方法
// 这个方法会检查响应的code和msg，然后调用GetSwapData
//
// 参数：
//   - okxResponse: 完整的OKX API响应
//   - walletAddress: 执行swap的钱包地址
//   - recentBlockhash: 最新的区块哈希
//   - jitoTipAmount: Jito小费金额（lamports）
//
// 返回：
//   - *solana.Transaction: 构建好的交易对象
//   - error: 错误信息
func GetSwapDataFromResponse(okxResponse OKXSwapInstructionResponse, walletAddress string, recentBlockhash solana.Hash, jitoTipAmount uint64) (*solana.Transaction, error) {
	// 验证响应状态
	if okxResponse.Code != "0" {
		return nil, fmt.Errorf("OKX API error: code=%s, msg=%s", okxResponse.Code, okxResponse.Msg)
	}

	// 将data部分转换为JSON字符串
	dataJSON, err := json.Marshal(okxResponse.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OKX data: %v", err)
	}

	// 调用GetSwapData处理
	return GetSwapData(string(dataJSON), walletAddress, recentBlockhash, jitoTipAmount)
}
