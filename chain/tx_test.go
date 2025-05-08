package chain

import (
	"fmt"
	"testing"
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
	//	"https://mainnet.helius-rpc.com/?api-key=255d1bfc-db66-46d5-b027-ba66aac51d91",
	//	"SOLANA")
	//fmt.Println(err1)
	//fmt.Printf("%+v\n", tx1)
}
