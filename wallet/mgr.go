package wallet

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

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

// 顺序不能变!
var suppChains []ChainCode = []ChainCode{ETH, SOLANA, BSC, BASE, OP, ARB, XLAYER}

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
