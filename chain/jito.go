package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const jitoAPI = "https://mainnet.block-engine.jito.wtf/api/v1/bundles"

type JitoRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type JitoResponse struct {
	JSONRPC string   `json:"jsonrpc"`
	Result  []string `json:"result"`
	ID      int      `json:"id"`
}

func getTipAccounts() (*JitoResponse, error) {
	reqBody := JitoRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "getTipAccounts",
		Params:  []interface{}{},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(jitoAPI, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var jitoResp JitoResponse
	if err := json.Unmarshal(body, &jitoResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &jitoResp, nil
}
