package chain

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	evmcommon "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"testing"
	"time"
)

func TestHash(t *testing.T) {
	ethrpc := "0"
	solrpc := "0"
	fmt.Println(ethrpc)
	fmt.Println(solrpc)
	tx, err := TxParser("0x096345e782606304b0695fc86ec33ae24012d3c035f94701f2c62c892ba5eedb", ethrpc, "ETH")
	fmt.Println(err)
	fmt.Printf("%+v\n", tx)
	//tx1, err1 := TxParser("2ghh7ujpZ1TRHG9aK4hEdQWE3zvmB2UHjvcfWn16vqnxjKUtYjuNwtb6wjfnQ75qMQ92xpRchXpPx5LdEFU63pJB",
	//	solrpc ,
	//	"SOLANA")
	//fmt.Println(err1)
	//fmt.Printf("%+v\n", tx1)
}

func TestAddress(t *testing.T) {
	isAddress := evmcommon.IsHexAddress("0x096345e782606304b0695fc86ec33ae24012d3c035f94701f2c62c892ba5eedb")
	fmt.Println(isAddress)
	isAddress = evmcommon.IsHexAddress("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5")
	fmt.Println(isAddress)
	isAddress = evmcommon.IsHexAddress("0x95222290dD7278aa3ddd389cc1e1d165cc4bafe5")
	fmt.Println(isAddress)
	isAddress = evmcommon.IsHexAddress("0x95222290DD7278aa3ddd389cc1e1d165cc4bafe5")
	fmt.Println(isAddress)
	isAddress = evmcommon.IsHexAddress("0x95222290DD7278aa3ddd389cc1e1d165cc4bafe")
	fmt.Println(isAddress)
	isAddress = evmcommon.IsHexAddress("0x096345e782606304b0695fc86ec33ae24012d3c035f94701f2c62c892ba5eedb")
	fmt.Println(isAddress)

}
func Test_GetLatestBlockhashFromMultipleClients(t *testing.T) {
	rpcs := []string{"https://api.zan.top/node/v1/solana/mainnet/34dbe590b04a4ce3bfd99823c7456c24",
		"https://mainnet.helius-rpc.com/?api-key=50465b2c-93d8-4d53-8987-a9ccd7962504",
		"https://solana-rpc.publicnode.com",
	}
	clients := make([]*rpc.Client, len(rpcs))
	for i, s := range rpcs {
		clients[i] = rpc.New(s)
	}
	for _ = range 2 {
		now := time.Now()
		multipleClients, err := GetLatestBlockhashFromMultipleClients(clients, rpc.CommitmentFinalized)
		spew.Dump(time.Since(now), multipleClients, err)
	}

}
func Test_SendTransactionWithMultipleDomains(t *testing.T) {
	tx := &solana.Transaction{}
	domains, err := SendTransactionWithMultipleDomains(context.TODO(), tx)
	spew.Dump(domains, err)

}
