package scripts

type Opcode byte

const (
	OP_FALSE Opcode = 0x00
	OP_TRUE  Opcode = 0x50

	OP_DUP   Opcode = 0x64
	OP_EQUAL Opcode = 0x78

	OP_VERIFY Opcode = 0x8c

	OP_HASH160  Opcode = 0xa0
	OP_CHECKSIG Opcode = 0xaa
)
