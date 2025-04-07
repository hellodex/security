package wallet

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hellodex/HelloSecurity/model"
	"github.com/hellodex/HelloSecurity/wallet/enc"
	"github.com/mr-tron/base58"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
)

type ChainCode string

type WalletObj struct {
	Address string
	Epm     string
	mem     string
	pk      string
}

const (
	ETH    ChainCode = "ETH"
	SOLANA ChainCode = "SOLANA"
	BSC    ChainCode = "BSC"
	BASE   ChainCode = "BASE"
	OP     ChainCode = "OP"
	ARB    ChainCode = "ARB"
	XLAYER ChainCode = "XLAYER"
)

type Quote struct {
	Symbol   string `json:"symbol"`
	Price    string `json:"price"`
	LestTime int64  `json:"lastTime"`
}

var (
	httpClient = &http.Client{}
	quoteMap   = map[string]*Quote{}
	lock       = sync.Mutex{}
	ChainsPair = map[ChainCode]string{
		ETH:    "ETH",
		SOLANA: "SOL",
		BSC:    "BNB",
		BASE:   "ETH",
		OP:     "ETH",
		ARB:    "ETH",
		XLAYER: "ETH",
	}
	suppChains []ChainCode = []ChainCode{ETH, SOLANA, BSC, BASE, OP, ARB, XLAYER}
)

func QuotePrice(cc string) string {
	lock.Lock()
	defer lock.Unlock()
	sp := ChainsPair[ChainCode(cc)]
	quote, exist := quoteMap[sp]
	if exist && time.Now().Unix()-quote.LestTime < 5 && len(quote.Price) > 0 && quote.Price != "0" {
		return quote.Price
	}
	quote = getQuoteBySymbol(sp)
	if quote == nil {
		return ""
	}
	quoteMap[sp] = quote
	return quote.Price
}
func getQuoteBySymbol(Symbol string) *Quote {
	for range 3 {
		req := bianReq(Symbol)
		if req != nil {
			return req
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}
	for range 3 {
		req := okxReq(Symbol)
		if req != nil {
			return req
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}
	return nil
}
func bianReq(symbol string) *Quote {
	resp, err := httpClient.Get("https://api.binance.com/api/v3/ticker/price?symbol=" + symbol + "USDT")
	if err != nil {
		fmt.Println("getQuoteBySymbol bian req error:", err)

	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("close resp body error: %v", err)
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("getQuoteBySymbol req error:", err)

	}
	var ps = Quote{}
	fmt.Println("resp body :", string(body))
	if err := json.Unmarshal(body, &ps); err != nil {
		log.Printf("json unmarshal error: %v", err)

	}
	if len(ps.Price) > 0 && ps.Price != "0" {
		ps.LestTime = time.Now().Unix()
		return &ps
	}
	return nil
}
func okxReq(symbol string) *Quote {

	resp, err := httpClient.Get("https://www.okx.com/api/v5/market/index-tickers?instId=" + symbol + "-USDT")
	if err != nil {
		fmt.Println("req error:", err)

	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("getQuoteBySymbol req error:", err)

	}
	var ps map[string]interface{}
	fmt.Println("resp body :", string(body))
	if err := json.Unmarshal(body, &ps); err != nil {
		log.Printf("json unmarshal error: %v", err)

	}
	if code, ok := ps["code"].(string); ok && code == "0" {

		if data, ok := ps["data"].([]interface{}); ok {
			for _, v := range data {
				if d, ok := v.(map[string]interface{}); ok {
					if symbol, ok := d["instId"].(string); ok && symbol == symbol+"-USDT" {
						price := d["idxPx"].(string)
						quote := Quote{
							Symbol:   symbol,
							Price:    price,
							LestTime: time.Now().Unix(),
						}
						return &quote
					}
				}
			}
		}
	}

	return nil
}

func IsSupp(cc ChainCode) (bool, bool) {
	for _, v := range suppChains {
		evm := true
		if cc == v {
			if cc == SOLANA {
				evm = false
			}
			return true, evm
		}
	}
	return false, false
}

func CheckAllCodes(ccs []string) []string {
	valid := make([]string, 0)
	for _, v := range ccs {
		supp, _ := IsSupp(ChainCode(v))
		if supp {
			valid = append(valid, v)
		}
	}
	return valid
}
func CheckAllCodesByIndex(cs string) []string {
	var selected []string
	if len(cs) == 0 {
		for _, bit := range suppChains {
			selected = append(selected, string(bit))
		}
		return selected
	}
	i2 := len(suppChains)
	for i, bit := range cs {

		if bit != '0' && i < i2 {
			selected = append(selected, string(suppChains[i]))
		}
	}
	return selected
}
func GetAllCodesByIndex() string {
	var selected = strings.Builder{}
	for _, _ = range suppChains {
		selected.WriteString("1")
	}

	return selected.String()
}

func New(addr, mem, pk string) *WalletObj {
	t := &WalletObj{}
	t.Address = addr
	t.mem = mem
	t.pk = pk
	t.Epm = "AES"
	return t
}

func (t *WalletObj) GetMem() string {
	return t.mem
}
func (t *WalletObj) GetPk() string {
	return t.pk
}

func Generate(wg *model.WalletGroup, chainCode ChainCode) (*WalletObj, error) {
	if wg == nil {
		return nil, errors.New("empty mnenomic")
	}
	if supp, evm := IsSupp(chainCode); supp {
		var addr, mem, pk string
		var err error
		if evm {
			addr, mem, pk, err = enc.GenerateEVM(wg)
			if err != nil {
				return nil, err
			}
		} else {
			if chainCode == SOLANA {
				addr, mem, pk, err = enc.GenerateSolana(wg)
				if err != nil {
					return nil, err
				}
			}
		}
		if len(addr) == 0 {
			return nil, errors.New("unknown error for creating wallet")
		}
		return New(addr, mem, pk), nil
	}
	return nil, errors.New("unsupport chain")
}

func generateEVM() (string, string, string, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", "", "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", "", "", err
	}
	seed := bip39.NewSeed(mnemonic, "")
	privateKey, err := crypto.ToECDSA(pbkdf2.Key(seed, []byte("ethereum"), 2048, 32, sha256.New))
	if err != nil {
		return "", "", "", err
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)

	privateKeyStr := common.Bytes2Hex(privateKeyBytes)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", "", "", errors.New("error casting public key to ECDSA")
	}
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	fmt.Println("Public Key:", common.Bytes2Hex(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	mneBytes, err := enc.Porter().Encrypt([]byte(mnemonic))
	if err != nil {
		return "", "", "", err
	}
	pkBytes, _ := enc.Porter().Encrypt([]byte(privateKeyStr))

	return address, base64.StdEncoding.EncodeToString(mneBytes), base64.StdEncoding.EncodeToString(pkBytes), nil
}

func generateSolana() (string, string, string, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", "", "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", "", "", err
	}

	seed := bip39.NewSeed(mnemonic, "")

	privateKeySeed := pbkdf2.Key(seed, []byte("ed25519 seed"), 2048, ed25519.SeedSize, sha256.New)

	privateKey := ed25519.NewKeyFromSeed(privateKeySeed)

	publicKey := privateKey.Public().(ed25519.PublicKey)

	address := base58.Encode(publicKey)

	mneBytes, err := enc.Porter().Encrypt([]byte(mnemonic))
	if err != nil {
		return "", "", "", err
	}

	pkBytes, err := enc.Porter().Encrypt([]byte(base64.StdEncoding.EncodeToString(privateKey)))
	if err != nil {
		return "", "", "", err
	}

	return address, base64.StdEncoding.EncodeToString(mneBytes), base64.StdEncoding.EncodeToString(pkBytes), nil
}
