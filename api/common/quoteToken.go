package common

import (
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	lock            = sync.Mutex{}
	quoteMap        = make(map[string]QuoteToken)
	mainnetTokenMap = make(map[string]MainnetTokenTemp)
	httpClient      = &http.Client{}
)

// initQuoteMap initializes the quoteMap with token data and prices
func init() {
	quoteMap = make(map[string]QuoteToken) // Clear the map
	// Initialize quote tokens

	quoteTokens := []QuoteToken{
		NewQuoteToken("0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c", "0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae", "BSC", "BNB", "BNBUSDT", "https://img.apihellodex.lol/quoteToken/bnb.png", 18, decimal.Zero),
		NewQuoteToken("0x55d398326f99059ff775485246999027b3197955", "", "BSC", "USDT", "USDT", "https://img.apihellodex.lol/quoteToken/usdt.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0xe9e7cea3dedca5984780bafc599bd69add087d56", "", "BSC", "BUSD", "USDT", "https://img.apihellodex.lol/quoteToken/bnb.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0x1af3f329e8be154074d8769d1ffa4ee058b1dbc3", "", "BSC", "DAI", "USDT", "https://img.apihellodex.lol/quoteToken/dai.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", "", "BSC", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640", "ETH", "WETH", "ETHUSDT", "https://img.apihellodex.lol/quoteToken/eth.png", 18, decimal.Zero),
		NewQuoteToken("0xdac17f958d2ee523a2206206994597c13d831ec7", "", "ETH", "USDT", "USDT", "https://img.apihellodex.lol/quoteToken/usdt.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0x6b175474e89094c44da98b954eedeac495271d0f", "", "ETH", "DAI", "USDT", "https://img.apihellodex.lol/quoteToken/dai.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "", "ETH", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0x82af49447d8a07e3bd95bd0d56f35241523fbab1", "0xc6962004f452be9203591991d15f6b388e09e8d0", "ARB", "WETH", "ETHUSDT", "https://img.apihellodex.lol/quoteToken/eth.png", 18, decimal.Zero),
		NewQuoteToken("0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", "", "ARB", "USDT", "USDT", "https://img.apihellodex.lol/quoteToken/usdt.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0xda10009cbd5d07dd0cecc66161fc93d7c9000da1", "", "ARB", "DAI", "USDT", "https://img.apihellodex.lol/quoteToken/dai.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0xaf88d065e77c8cc2239327c5edb3a432268e5831", "", "ARB", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0x4200000000000000000000000000000000000006", "0xb2cc224c1c9fee385f8ad6a55b4d94e92359dc59", "BASE", "WETH", "ETHUSDT", "https://img.apihellodex.lol/quoteToken/eth.png", 18, decimal.Zero),
		NewQuoteToken("0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22", "", "BASE", "cbETH", "ETHUSDT", "", 18, decimal.Zero),
		NewQuoteToken("0xd4a0e0b9149bcee3c920d2e00b5de09138fd8bb7", "", "BASE", "aBasWETH", "ETHUSDT", "", 18, decimal.Zero),
		NewQuoteToken("0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", "", "BASE", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0x50c5725949a6f0c72e6c4a641f24049a917db0cb", "", "BASE", "Dai", "USDT", "https://img.apihellodex.lol/quoteToken/dai.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0x4200000000000000000000000000000000000006", "", "OP", "WETH", "ETHUSDT", "https://img.apihellodex.lol/quoteToken/eth.png", 18, decimal.Zero),
		NewQuoteToken("0x7f5c764cbc14f9669b88837ca1490cca17c31607", "", "OP", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0x0b2c639c533813f4aa9d7837caf62653d097ff85", "", "OP", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0xda10009cbd5d07dd0cecc66161fc93d7c9000da1", "", "OP", "Dai", "USDT", "https://img.apihellodex.lol/quoteToken/dai.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0x94b008aa00579c1307b0ef2c499ad98a8ce58e58", "", "OP", "USDT", "USDT", "https://img.apihellodex.lol/quoteToken/usdt.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("So11111111111111111111111111111111111111112", "Czfq3xZZDmsdGdUyrNLtRhGc47cXcZtLG4crryfu44zE", "SOLANA", "SOL", "SOLUSDT", "https://img.apihellodex.lol/BSC/0x570a5d26f7765ecb712c0924e4de545b89fd43df.png", 9, decimal.Zero),
		NewQuoteToken("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "SOLANA", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "SOLANA", "USDT", "USDT", "https://img.apihellodex.lol/quoteToken/usdt.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0xe538905cf8410324e03a5a23c1c177a474d59b2b", "0x2b59b462103efaa4d04e869d62985b43b46a93c9", "XLAYER", "WOKB", "OKBUSDT", "https://img.apihellodex.lol/ETH/0x75231f58b43240c9718dd58b4967c5114342a86c.png", 18, decimal.Zero),
		NewQuoteToken("0x1e4a5963abfd975d8c9021ce480b42188849d41d", "", "XLAYER", "USDT", "USDT", "https://img.apihellodex.lol/quoteToken/usdt.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0xc5015b9d9161dca7e18e32f6f25c4ad850731fd4", "", "XLAYER", "DAI", "USDT", "https://img.apihellodex.lol/quoteToken/dai.png", 18, decimal.NewFromInt(1)),
		NewQuoteToken("0x74b7f16337b8972027f6196a17a631ac6de26d22", "", "XLAYER", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0x5a77f1443d16ee5761d310e38b62f77f726bc71c", "", "XLAYER", "WETH", "ETHUSDT", "https://img.apihellodex.lol/quoteToken/eth.png", 18, decimal.Zero),
		NewQuoteToken("0x900101d06a7426441ae63e9ab3b9b0f63be145f1", "", "CORE", "USDT", "USDT", "https://img.apihellodex.lol/quoteToken/usdt.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0xa4151b2b3e269645181dccf2d426ce75fcbdeca9", "", "CORE", "USDC", "USDT", "https://img.apihellodex.lol/quoteToken/usdc.png", 6, decimal.NewFromInt(1)),
		NewQuoteToken("0x40375c92d9faf44d2f9db9bd9ba41a3317a2404f", "0x396895a08dfec54dc5773a14a7f0888c4e1a11b6", "CORE", "WCORE", "COREUSDT", "", 18, decimal.Zero),
		NewQuoteToken("0x191e94fa59739e188dce837f7f6978d84727ad01", "0xd203eab4e8c741473f7456a9f32ce310d521fa41", "CORE", "WCORE", "COREUSDT", "", 18, decimal.Zero),
	}
	// Update quoteTokens with prices
	for i, qt := range quoteTokens {

		quoteMap[qt.ChainCode+":"+qt.Address] = quoteTokens[i]
	}
	mainnetTokenMap["ETH"] = NewMainnetTokenTemp(
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		"0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640",
		"ETH",
		"ETH",
		18,
	)
	mainnetTokenMap["BSC"] = NewMainnetTokenTemp(
		"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
		"0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae",
		"BSC",
		"BNB",
		18,
	)
	mainnetTokenMap["ARB"] = NewMainnetTokenTemp(
		"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
		"0xc6962004f452be9203591991d15f6b388e09e8d0",
		"ARB",
		"ETH",
		18,
	)
	mainnetTokenMap["BASE"] = NewMainnetTokenTemp(
		"0x4200000000000000000000000000000000000006",
		"0xb2cc224c1c9fee385f8ad6a55b4d94e92359dc59",
		"BASE",
		"ETH",
		18,
	)
	mainnetTokenMap["OP"] = NewMainnetTokenTemp(
		"0x4200000000000000000000000000000000000006",
		"0x85149247691df622eaf1a8bd0cafd40bc45154a9",
		"OP",
		"ETH",
		18,
	)
	mainnetTokenMap["XLAYER"] = NewMainnetTokenTemp(
		"0xe538905cf8410324e03a5a23c1c177a474d59b2b",
		"0x2b59b462103efaa4d04e869d62985b43b46a93c9",
		"XLAYER",
		"ETH",
		18,
	)
	mainnetTokenMap["CORE"] = NewMainnetTokenTemp(
		"0x40375c92d9faf44d2f9db9bd9ba41a3317a2404f",
		"0x396895A08dfec54dC5773a14A7f0888C4E1a11b6",
		"CORE",
		"CORE",
		18,
	)
	mainnetTokenMap["SOLANA"] = NewMainnetTokenTemp(
		"So11111111111111111111111111111111111111112",
		"58oQChx4yWmvKdwLLZzBi4ChoCc2fqCUWBkwMihLYQo2",
		"SOLANA",
		"SOL",
		9,
	)
}
func MainnetToken(key string) (MainnetTokenTemp, bool) {
	temp, exist := mainnetTokenMap[key]
	return temp, exist
}

func QuotePrice(chainCode, cc string) *QuoteToken {
	lock.Lock()
	defer lock.Unlock()
	if (chainCode == "SOLANA" && (cc == "11111111111111111111111111111111")) || cc == "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" {
		cc = ""
	}
	// 查询主网代币价格
	if cc == "" {
		m, exist := mainnetTokenMap[chainCode]
		if exist {
			quoteTokens, exist := quoteMap[chainCode+":"+m.MainnetTokenAddress]
			if exist {
				symbol := quoteTokens.PairSymbol
				symbol = strings.ReplaceAll(symbol, "USDT", "")

				if time.Now().Unix()-quoteTokens.LestTime < 5 && quoteTokens.Price.Sign() > 0 {
					return &quoteTokens
				}
				bySymbol := getQuoteBySymbol(symbol)
				if bySymbol == nil || len(bySymbol.Price) < 1 || bySymbol.Price == "0" {
					return nil
				}
				price, _ := decimal.NewFromString(bySymbol.Price)
				quoteTokens.Price = price
				quoteTokens.LestTime = time.Now().Unix()
				return &quoteTokens
			}
		}
	} else {
		quoteTokens, exist := quoteMap[chainCode+":"+cc]
		if exist && quoteTokens.PairSymbol == "USDT" {
			return &quoteTokens
		}
	}
	return nil
}
func getQuoteBySymbol(Symbol string) *QueryQuote {
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
func bianReq(symbol string) *QueryQuote {
	winFlag := false
	switch os := runtime.GOOS; os {
	case "windows":
		winFlag = true
		fmt.Println("当前系统是 Windows")
	case "linux":
		fmt.Println("当前系统是 Linux")
	default:
		fmt.Printf("当前系统是 %s\n", os)
	}
	client := &http.Client{}
	if winFlag {
		proxyURL, err := url.Parse("http://127.0.0.1:7897")
		if err != nil {
			panic(err)
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client = &http.Client{
			Transport: transport,
		}
	}
	resp, err := client.Get("https://api.binance.com/api/v3/ticker/price?symbol=" + symbol + "USDT")
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
	var ps = QueryQuote{}
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
func okxReq(symbol string) *QueryQuote {

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
						quote := QueryQuote{
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

type QueryQuote struct {
	Symbol   string `json:"symbol"`
	Price    string `json:"price"`
	LestTime int64  `json:"lastTime"`
}

type MainnetTokenTemp struct {
	MainnetTokenAddress   string // Exported (public)
	MainnetTokenLPAddress string // Exported (public)
	ChainCode             string // Unexported (private)
	Symbol                string // Unexported (private)
	Decimals              int    // Unexported (private)
}

type QuoteToken struct {
	Address     string          `json:"address"`
	PairAddress string          `json:"pairAddress"`
	ChainCode   string          `json:"chainCode"`
	Symbol      string          `json:"symbol"`
	Decimals    int             `json:"decimals"`
	PairSymbol  string          `json:"pairSymbol"`
	Price       decimal.Decimal `json:"price"`
	LestTime    int64           `json:"lestTime"`
	Logo        string          `json:"logo"`
}

func NewQuoteToken(address, pairAddress, chainCode, symbol, pairSymbol, logo string, decimals int, price decimal.Decimal) QuoteToken {
	return QuoteToken{
		Address:     address,
		PairAddress: pairAddress,
		ChainCode:   chainCode,
		Symbol:      symbol,
		Decimals:    decimals,
		PairSymbol:  pairSymbol,
		Price:       price,
		LestTime:    0,
		Logo:        logo,
	}
}
func NewMainnetTokenTemp(mainnetTokenAddress, mainnetTokenLPAddress, chainCode, symbol string, decimals int) MainnetTokenTemp {
	return MainnetTokenTemp{
		MainnetTokenAddress:   mainnetTokenAddress,
		MainnetTokenLPAddress: mainnetTokenLPAddress,
		ChainCode:             chainCode,
		Symbol:                symbol,
		Decimals:              decimals,
	}
}
