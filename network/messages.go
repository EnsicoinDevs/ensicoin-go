package network

import (
	"time"
)

type Message struct {
	Magic     int       `json:"magic"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type WhoamiMessage struct {
	Version int `json:"version"`
}

type InvMessage struct {
	Type   string   `json:"type"`
	Hashes []string `json:"hashes"`
}

type GetDataMessage struct {
	Inv InvMessage `json:"inv"`
}

type NotFoundMessage struct {
	Type string `json:"type"`
	Hash string `json:"hash"`
}

type BlockMessage struct {
	Header struct {
		Version         int       `json:"version"`
		Flags           []string  `json:"flags"`
		HashPrevBlock   string    `json:"hashPrevBlock"`
		HashTransaction string    `json:"hashTransactions"`
		Timestamp       time.Time `json:"timestamp"`
		Nonce           int       `json:"nonce"`
	} `json:"header"`
	Transactions []TransactionMessage `json:"transactions"`
}

type TransactionMessage struct {
	Version int      `json:"version"`
	Flags   []string `json:"flags"`
}

type GetBlocksMessage struct {
	Hashes   []string `json:"hashes"`
	StopHash string   `json:"stopHash"`
}

type GetMempoolMessage struct {
}
