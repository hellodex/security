package main

import (
	md5 "crypto/md5"
	"fmt"
)

func TestMd5() {
	hash := md5.Sum([]byte("hellodex"))
	fmt.Println(fmt.Sprintf("%x", hash))
}

func main() {
	TestMd5()
}
