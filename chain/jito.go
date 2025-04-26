package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
)

const domain = "https://tokyo.mainnet.block-engine.jito.wtf"

var (
	bundleWay = domain + "/api/v1/bundles"
	transWay  = domain + "/api/v1/transactions?bundleOnly=true"
)

type JitoRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type JitoResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	ID      int         `json:"id"`
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
	req.Header.Set("x-jito-auth", "BjfsbDKpjWjcY1NA4wbEuspo6wFKsW2bbvo5RbHYNL2W")

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

func updateInstructionIndexes(tx *solana.Transaction, insertIndex int) {

	for i, instr := range tx.Message.Instructions {
		if i == len(tx.Message.Instructions)-1 {
			return
		}
		for j, accIndex := range instr.Accounts {
			if accIndex >= uint16(insertIndex) {
				instr.Accounts[j] += uint16(1)
			}
		}

		if instr.ProgramIDIndex >= uint16(insertIndex) {
			instr.ProgramIDIndex += uint16(1)
		}

		tx.Message.Instructions[i] = instr
	}
}
