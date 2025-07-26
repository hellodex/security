package chain

import "errors"

type CallType string

const (
	CallTypeGeneral   CallType = "general"
	CallTypeJito      CallType = "jito"
	AuthForceCloseAll CallType = "AuthForceCloseAll"
)

var validTransactionTypes = map[string]CallType{
	"general":           CallTypeGeneral,
	"jito":              CallTypeJito,
	"AuthForceCloseAll": AuthForceCloseAll,
}

func parseCallType(input string) (CallType, error) {
	if t, exists := validTransactionTypes[input]; exists {
		return t, nil
	}
	return "", errors.New("invalid transaction type: " + input)
}
