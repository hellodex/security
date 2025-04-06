package config

import (
	"encoding/base64"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/hellodex/HelloSecurity/log"
	"testing"
)

func TestConfigInit(t *testing.T) {
	rpcList := systemConfig.Chain[0].GetRpc()
	rpcMapList := systemConfig.Chain[0].GetRpcMapper()
	fmt.Println(rpcList, rpcMapList)

	i := systemConfig.Chain[0].RpcMap[rpcList[0]]
	fmt.Println(i)
	spew.Dump(systemConfig)
}
func TestDecodeHash(t *testing.T) {
	messageStr := "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAQAGCp9rpYhQ8Qq6XZ8HQzSB40WEFd/StM5Lw/GwWV6Y81BLTad/bpYC3qDswLjnKYSxkDkTCPgIoUaB+e4tk1YmeRROWiXGNIaC5fyJbMeGrzbz0fzFOVeVh0G4YPoMdy3DYmu8TE4GnGt8pIzceBKPARKkGMpzVeik/jRsY7sV1PKhAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADBkZv5SEXMv/srbpyw5vnvIzlu8X3EmssQ5s6QAAAAAR51VvyMcBu7nTFbs5oFQf9sbLeo/SOUQKxzaJWvBOPBt324ddloZPZy+FGzut5rBy0he1fWzeROoz1hX7/AKmMlyWPTiSJ8bs9ECkUjg2DC1oTmdr/EIQEjnvY2+n4WbQ/+if11/ZKdMCbHylYed5LCas238ndUUsyGqezjOXo7KkvH+IJdnat/O3WAuKgQ8KqDjC/Z/j6KQ7y/v0Yxm0FBQAFAobxAQAFAAkDoIYBAAAAAAAIBgABAA0EBwEBBhsHAAIBBg0DCQYOEQAPEA0CAQoMEgsHBwQIEw4j5RfLl3rjrSoBAAAASWQAAWxuhBwAAAAAg0IVAgAAAABQAGQHAwEAAAEJAZ2XJzX6zibHR+b4OiBnLPyWzlNUvlJLX9qDadSmpMHeA9nW3AcCPzw43dVC"
	message, err := base64.StdEncoding.DecodeString(messageStr)
	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(message))

	log.Errorf("tx: %s, err: %v", tx, err)
}
