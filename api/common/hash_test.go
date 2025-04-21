package common

import (
	"testing"
)

func TestEtherSign(t *testing.T) {
	wallet := "??"
	private := "??"
	hex, err := EtherGenSignHex(private,
		"hello1")
	if err != nil {
		t.Error(err)
	}
	verify, _ := EtherVerifySign(hex, "hello12", wallet)
	if !verify {
		t.Error("VerifyEtherSign failed")
	}
}

func TestSolanaSign(t *testing.T) {
	wallet := "??"
	private := "??"

	signature, err := SolanaGenSign("hello1", private)
	if err != nil {
		t.Error(err)
	}
	verify, _ := SolanaVerifySign(signature, "hello1", wallet)
	if !verify {
		t.Error("VerifyEtherSign failed")
	}
}
