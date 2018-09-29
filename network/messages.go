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
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{
		Alias:     (*Alias)(message),
		Timestamp: message.Timestamp.Unix(),
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
