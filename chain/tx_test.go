package chain

import (
	"fmt"
	evmcommon "github.com/ethereum/go-ethereum/common"
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
