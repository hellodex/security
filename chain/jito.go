package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hellodex/HelloSecurity/config"

	"github.com/gagliardetto/solana-go"
)

const domain = "https://tokyo.mainnet.block-engine.jito.wtf"

var (
	bundleWay    = domain + "/api/v1/bundles"
	transWay     = domain + "/api/v1/transactions?bundleOnly=true" + "&uuid=" + config.GetConfig().JitoUUID
	transWayUUID = "&uuid=" + config.GetConfig().JitoUUID
)

type JitoRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type JitoResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	Result  interface{}  `json:"result"`
	Error   *interface{} `json:"error,omitempty"`
	ID      int          `json:"id"`
}

func retrieveBundPath() string {
	return bundleWay
}

func retrieveTransPath() string {
	return transWay
}

func getTipAccounts() (string, error) {
	reqBody := JitoRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "getTipAccounts",
		Params:  []interface{}{},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(retrieveBundPath(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var jitoResp JitoResponse
	if err := json.Unmarshal(body, &jitoResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	accSlice, ok := jitoResp.Result.([]interface{})
	if !ok || len(accSlice) == 0 {
		return "", fmt.Errorf("empty tip account list")
	}

	return fmt.Sprintf("%v", accSlice[0]), nil
}

func SendTransactionWithCtx(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	txBase64, err := tx.ToBase64()
	mylog.Info("transaction content: ", txBase64, err)
	if err != nil {
		return solana.Signature{}, err
	}
	reqBody := JitoRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "sendTransaction",
		Params: []interface{}{
			txBase64,
			map[string]string{"encoding": "base64"},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", transWay, bytes.NewBuffer(jsonData))
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("x-jito-auth", "BjfsbDKpjWjcY1NA4wbEuspo6wFKsW2bbvo5RbHYNL2W")

	startms := time.Now().UnixMilli()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send request: %v, %dms", err, time.Now().UnixMilli()-startms)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to read response: %v", err)
	}
	mylog.Infof("EX jito request %dms", time.Now().UnixMilli()-startms)

	var jitoResp JitoResponse
	if err := json.Unmarshal(body, &jitoResp); err != nil {
		return solana.Signature{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	sigstr, ok := jitoResp.Result.(string)
	if !ok || len(sigstr) == 0 {
		return solana.Signature{}, fmt.Errorf("empty signature response")
	}

	sig, err := solana.SignatureFromBase58(sigstr)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("invalid signature format: %v", err)
	}

	return sig, nil
}
func SendTransactionWithCtxTest(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	txBase64, err := tx.ToBase64()
	mylog.Info("进入SendTransactionWithCtxTest ")
	if err != nil {
		return solana.Signature{}, err
	}
	reqBody := JitoRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "sendTransaction",
		Params: []interface{}{
			txBase64,
			map[string]string{"encoding": "base64"},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", transWay, bytes.NewBuffer(jsonData))
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("x-jito-auth", "BjfsbDKpjWjcY1NA4wbEuspo6wFKsW2bbvo5RbHYNL2W")

	startms := time.Now().UnixMilli()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send request: %v, %dms", err, time.Now().UnixMilli()-startms)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to read response: %v", err)
	}
	mylog.Infof("EX jito request %dms", time.Now().UnixMilli()-startms)

	var jitoResp JitoResponse
	if err := json.Unmarshal(body, &jitoResp); err != nil {
		return solana.Signature{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	sigstr, ok := jitoResp.Result.(string)
	if !ok || len(sigstr) == 0 {
		return solana.Signature{}, fmt.Errorf("empty signature response")
	}

	sig, err := solana.SignatureFromBase58(sigstr)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("invalid signature format: %v", err)
	}

	return sig, nil
}

// 第三方测试 上链 fountainhead.land
func SendTransactionWithCtxTestFountainhead(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	txBase64, err := tx.ToBase64()
	mylog.Info("调用第三方 SendTransactionWithCtxTestFountainhead 发送交易 ")
	if err != nil {
		return solana.Signature{}, err
	}
	reqBody := JitoRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "sendTransaction",
		Params: []interface{}{
			txBase64,
			map[string]string{"encoding": "base64"},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to marshal request: %v", err)
	}
	transWay = "https://landing-ams.fountainhead.land"
	req, err := http.NewRequestWithContext(ctx, "POST", transWay, bytes.NewBuffer(jsonData))
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("x-jito-auth", "BjfsbDKpjWjcY1NA4wbEuspo6wFKsW2bbvo5RbHYNL2W")

	startms := time.Now().UnixMilli()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send request: %v, %dms", err, time.Now().UnixMilli()-startms)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to read response: %v", err)
	}
	mylog.Infof("EX jito request %dms", time.Now().UnixMilli()-startms)

	var jitoResp JitoResponse
	if err := json.Unmarshal(body, &jitoResp); err != nil {
		return solana.Signature{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	sigstr, ok := jitoResp.Result.(string)
	if !ok || len(sigstr) == 0 {
		return solana.Signature{}, fmt.Errorf("empty signature response")
	}

	sig, err := solana.SignatureFromBase58(sigstr)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("invalid signature format: %v", err)
	}

	return sig, nil
}

// updateInstructionIndexes 更新 Solana 交易中指令的账户索引和程序 ID 索引。
// 当在交易的账户列表中插入新账户（如 Tip 账户）时，需要调整指令中的索引以保持正确性。
// 参数：
// - tx: Solana 交易对象，包含消息和指令列表。
// - insertIndex: 新账户插入的位置索引，插入后该索引及以上的账户索引需要递增。
func updateInstructionIndexes(tx *solana.Transaction, insertIndex int) {
	// 遍历交易消息中的所有指令。
	for i, instr := range tx.Message.Instructions {
		// 如果当前指令是最后一个指令，直接返回，跳过处理。
		// 注意：此条件可能有误，因为最后一个指令仍需更新索引，可能是逻辑错误。
		if i == len(tx.Message.Instructions)-1 {
			return
		}

		// 遍历指令中的账户索引列表。
		for j, accIndex := range instr.Accounts {
			// 如果账户索引大于或等于插入点索引，则将其递增 1。
			// 这是因为插入新账户导致原索引大于等于 insertIndex 的账户向后偏移一位。
			if accIndex >= uint16(insertIndex) {
				instr.Accounts[j] += uint16(1)
			}
		}

		// 如果指令的程序 ID 索引大于或等于插入点索引，则将其递增 1。
		// 程序 ID 通常指向账户列表中的程序账户，插入新账户可能导致程序 ID 的索引偏移。
		if instr.ProgramIDIndex >= uint16(insertIndex) {
			instr.ProgramIDIndex += uint16(1)
		}

		// 将更新后的指令写回到交易的指令列表中。
		tx.Message.Instructions[i] = instr
	}
}

// 不同区域机房的 Jito RPC 域名列表

var JitoDomains = []string{
	"https://mainnet.block-engine.jito.wtf",
	"https://amsterdam.mainnet.block-engine.jito.wtf",
	"https://frankfurt.mainnet.block-engine.jito.wtf",
	"https://london.mainnet.block-engine.jito.wtf",
	"https://ny.mainnet.block-engine.jito.wtf",
	"https://slc.mainnet.block-engine.jito.wtf",
	"https://singapore.mainnet.block-engine.jito.wtf",
	"https://tokyo.mainnet.block-engine.jito.wtf",
}

// SendTransactionWithMultipleDomains 并发向多个 Jito 域名发送交易
// 一旦有任何一个请求成功，立即返回结果并取消其他请求
func SendTransactionWithMultipleDomains(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	if len(JitoDomains) == 0 {
		return solana.Signature{}, errors.New("EX jito request no  domains provided")
	}

	// 设置整体超时时间（3秒）
	overallCtx, overallCancel := context.WithTimeout(ctx, 1*time.Second)
	defer overallCancel()

	// 用于同步所有协程
	var wg sync.WaitGroup
	// 用于传递第一个成功的结果
	resultChan := make(chan struct {
		sig    solana.Signature
		err    error
		domain string
	}, len(JitoDomains))

	// 用于标记是否已有成功结果
	var successOnce sync.Once
	var finalResult solana.Signature
	var finalError = errors.New("no successful request yet")

	// 启动协程并发请求所有域名
	for i, domain1 := range JitoDomains {
		wg.Add(1)
		go func(domainURL string, index int) {
			defer wg.Done()
			defer func() {
				// 防止panic中断程序
				if r := recover(); r != nil {
					mylog.Errorf("EX jito request Panic goroutine %d (domain: %s): %v", index, domainURL, r)
					select {
					case resultChan <- struct {
						sig    solana.Signature
						err    error
						domain string
					}{solana.Signature{}, fmt.Errorf("EX jito request Panic goroutine: %v", r), domainURL}:
					case <-overallCtx.Done():
						// context已取消，不发送结果
					}
				}
			}()

			// 为每个请求设置独立的超时时间（2秒）
			requestCtx, requestCancel := context.WithTimeout(overallCtx, 400*time.Second)
			defer requestCancel()

			//startTime := time.Now()
			sig, err := sendTransactionToDomain(requestCtx, tx, domainURL)
			//elapsed := time.Since(startTime)

			if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
				//mylog.Errorf("EX jito request failed for domain %s (elapsed: %dms): %v",
				//	domainURL, elapsed.Milliseconds(), err)
				//mylog.Info()
			} else {
				//mylog.Infof("EX jito request success for domain %s (elapsed: %dms), signature: %s",
				//	domainURL, elapsed.Milliseconds(), sig.String())
				mylog.Info()
			}

			// 发送结果到通道
			select {
			case resultChan <- struct {
				sig    solana.Signature
				err    error
				domain string
			}{sig, err, domainURL}:
			case <-overallCtx.Done():
				// context已取消，不发送结果
			}
		}(domain1, i)
	}

	// 监听第一个成功的结果
	go func() {
		for {
			select {
			case result := <-resultChan:
				if result.err == nil {
					successOnce.Do(func() {
						finalResult = result.sig
						finalError = nil
						// 第一个成功的请求
						//mylog.Infof("EX jito res successful  from domain: %s, signature: %s",
						//	result.domain, result.sig.String())
						overallCancel() // 取消其他请求
					})
					return
				} else {
				}
			case <-overallCtx.Done():
				return
			}
		}
	}()

	// 等待所有协程完成或第一个成功结果
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有请求完成
		if finalError == nil {
			//mylog.Infof("EX jito res successful  successfully to multiple domains!")
			return finalResult, nil
		}
		// 所有请求都失败了
		return solana.Signature{}, errors.New("EX jito req all domain requests failed")
	case <-overallCtx.Done():
		// 整体超时或有成功结果
		if finalError == nil {
			return finalResult, nil
		}
		return solana.Signature{}, errors.New("request timeout or cancelled")
	}
}

// sendTransactionToDomain 向指定域名发送交易请求
func sendTransactionToDomain(ctx context.Context, tx *solana.Transaction, domain string) (solana.Signature, error) {
	txBase64, err := tx.ToBase64()
	fmt.Println("签名后：Base64:", txBase64)
	fmt.Println("")
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to encode transaction: %v", err)
	}

	// 构建请求体
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "sendTransaction",
		"params": []interface{}{
			txBase64,
			map[string]string{"encoding": "base64"},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// 构建完整的URL
	transactionURL := domain + "/api/v1/transactions?bundleOnly=false"
	transactionURL = transactionURL + transWayUUID

	req, err := http.NewRequestWithContext(ctx, "POST", transactionURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	bundleId := resp.Header.Get("x-bundle-id")
	body, err := io.ReadAll(resp.Body)
	mylog.Infof("jito request bundleId:%s, url:%s ,res:%s ", bundleId, transactionURL, body)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to read response: %v", err)
	}

	var jitoResp JitoResponse
	if err := json.Unmarshal(body, &jitoResp); err != nil {
		return solana.Signature{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	if jitoResp.Error != nil {
		return solana.Signature{}, fmt.Errorf("jitorpcerror::%+v", jitoResp.Error)
	}
	sigstr, ok := jitoResp.Result.(string)
	if !ok || len(sigstr) == 0 {
		return solana.Signature{}, fmt.Errorf("empty signature response")
	}

	sig, err := solana.SignatureFromBase58(sigstr)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("invalid signature format: %v", err)
	}

	return sig, nil
}
