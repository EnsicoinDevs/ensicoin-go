package blockchain

import (
	"golang.org/x/crypto/ripemd160"
)

type Script struct {
	inputScript  []string
	outputScript []string

	pile []interface{}
}

func NewScript(inputScript []string, outputScript []string) *Script {
	return &Script{
		inputScript:  inputScript,
		outputScript: outputScript,
	}
}

func (script *Script) Execute() bool {
	fullScript := append(script.outputScript, script.inputScript...)

	for _, op := range fullScript {
		switch op {
		case "OP_DUP":
			if len(script.pile) < 1 {
				return false
			}

			script.pile = append(script.pile, script.pile[len(script.pile)-1])
		case "OP_HASH160":
			if len(script.pile) < 1 {
				return false
			}

			h := ripemd160.New()

			val := script.pile[len(script.pile)-1]

			h.Write(val.([]byte))

			script.pile[len(script.pile)-1] = h.Sum(nil)
		case "OP_EQUAL":
			if len(script.pile) < 2 {
				return false
			}

			val1 := script.pile[len(script.pile)-1]
			val2 := script.pile[len(script.pile)-2]
			script.pile = script.pile[:len(script.pile)-2]

			script.pile = append(script.pile, val1 == val2)
		case "OP_VERIFY":
			if len(script.pile) < 1 {
				return false
			}

			val, ok := script.pile[len(script.pile)-1].(bool)
			if !ok || !val {
				return false
			}
		case "OP_CHECKSIG":
			if len(script.pile) < 2 {
				return false
			}
		default:
			script.pile = append(script.pile, op)
		}
	}

	return true
}
