package network

import (
	"encoding/json"
	"time"
)

type Message struct {
	Magic     int         `json:"magic"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Message   interface{} `json:"message"`
}

func (message *Message) MarshalJSON() ([]byte, error) {
	type Alias Message

	return json.Marshal(&struct {
		Timestamp int64 `json:"timestamp"`
		*Alias
	}{
		Timestamp: message.Timestamp.Unix(),
		Alias:     (*Alias)(message),
	})
}

func (message *Message) UnmarshalJSON(data []byte) error {
	type Alias Message

	aux := &struct {
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{
		Alias: (*Alias)(message),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	message.Timestamp = time.Unix(aux.Timestamp, 0)
	return nil
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

type BlockHeader struct {
	Version       int       `json:"version"`
	Flags         []string  `json:"flags"`
	HashPrevBlock string    `json:"hashPrevBlock"`
	HashTxs       string    `json:"hashTransactions"`
	Timestamp     time.Time `json:"timestamp"`
	Nonce         int       `json:"nonce"`
}

type BlockMessage struct {
	Header BlockHeader `json:"header"`
	Txs    []TxMessage `json:"transactions"`
}

type TxOutpoint struct {
	TxHash string `json:"transactionHash"`
	Index  uint   `json:"index"`
}

type TxInput struct {
	PreviousOutput TxOutpoint `json:"previousOutput"`
	Script         []string   `json:"script"`
}

type TxOutput struct {
	Value  uint64   `json:"value"`
	Script []string `json:"script"`
}

type TxMessage struct {
	Version int        `json:"version"`
	Flags   []string   `json:"flags"`
	Inputs  []TxInput  `json:"inputs"`
	Outputs []TxOutput `json:"outputs"`
}

type GetBlocksMessage struct {
	Hashes   []string `json:"hashes"`
	StopHash string   `json:"stopHash"`
}

type GetMempoolMessage struct {
}
