package chain

import "errors"

type CallType string

const (
	CallTypeGeneral CallType = "general"
	CallTypeJito    CallType = "jito"
)

var validTransactionTypes = map[string]CallType{
	"general": CallTypeGeneral,
	"jito":    CallTypeJito,
}

func parseCallType(input string) (CallType, error) {
	if t, exists := validTransactionTypes[input]; exists {
		return t, nil
	}
	return "", errors.New("invalid transaction type: " + input)
}
