package scripts

import (
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/crypto/ripemd160"
	"reflect"
)

type Script struct {
	Tx               *network.TxMessage
	Input            *network.TxIn
	SpentOutputValue uint64
	Opcodes          []Opcode
}

func NewScript(tx *network.TxMessage, input *network.TxIn, spentOutputValue uint64, outputScript []byte, inputScript []byte) *Script {
	script := &Script{
		Tx:               tx,
		Input:            input,
		SpentOutputValue: spentOutputValue,
	}

	for _, opcode := range append(inputScript, outputScript...) {
		script.Opcodes = append(script.Opcodes, Opcode(opcode))
	}

	return script
}

func (script *Script) Validate() (bool, error) {
	var stack [][]byte
	var numberOfBytesToPush uint8

	for _, opcode := range script.Opcodes {
		l := len(stack)

		if numberOfBytesToPush > 0 {
			stack[l-1] = append(stack[l-1], byte(opcode))
			numberOfBytesToPush--

			continue
		}

		switch opcode {
		case OP_FALSE:
			stack = append(stack, []byte{0x00})

		case OP_TRUE:
			stack = append(stack, []byte{0x01})

		case OP_DUP:
			if l < 1 {
			}

			stack = append(stack, stack[l-1])

		case OP_EQUAL:
			if l < 2 {
				return false, nil
			}

			a := stack[l-1]
			b := stack[l-2]

			stack = stack[:l-1]

			if reflect.DeepEqual(a, b) {
				stack[l-2] = []byte{0x01}
			} else {
				stack[l-2] = []byte{0x00}
			}

		case OP_VERIFY:
			if l < 1 {
				return false, nil
			}

			top := stack[l-1]
			if len(top) == 1 && top[0] == 0x00 {
				return false, nil
			}
			stack = stack[:l-1]

		case OP_HASH160:
			if l < 1 {
				return false, nil
			}

			hash := ripemd160.New()
			hash.Write(stack[l-1])
			stack[l-1] = hash.Sum(nil)

		case OP_CHECKSIG:
			if l < 2 {
				return false, nil
			}

			rawPubKey := stack[l-1]
			rawSig := stack[l-2]

			stack = stack[:l-1]

			if len(rawPubKey) != 33 {
				return false, nil
			}

			shash := script.Tx.SHash(script.Input, script.SpentOutputValue) // TODO: n2

			pubKey, err := btcec.ParsePubKey(rawPubKey, btcec.S256())
			if err != nil {
				return false, nil
			}

			sig, err := btcec.ParseDERSignature(rawSig, btcec.S256())
			if err != nil {
				return false, nil
			}

			if sig.Verify(shash[:], pubKey) {
				stack[l-2] = []byte{0x01}
			} else {
				stack[l-2] = []byte{0x00}
			}

		default:
			if 0x01 <= opcode && opcode <= 0x75 {
				numberOfBytesToPush = uint8(opcode)

				stack = append(stack, nil)
			} else {
				return false, nil
			}
		}
	}

	top := stack[len(stack)-1]
	if len(top) == 1 && top[0] == 0x00 {
		return false, nil
	}

	return true, nil
}
