package network

import (
	"encoding/json"
	"github.com/EnsicoinDevs/ensicoin-go/consensus"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type PeerCallbacks struct {
	OnReady       func()
	OnWhoami      func(*WhoamiMessage)
	OnInv         func(*InvMessage)
	OnGetData     func(*GetDataMessage)
	OnNotFound    func(*NotFoundMessage)
	OnBlock       func(*BlockMessage)
	OnTransaction func(*TransactionMessage)
	OnGetBlocks   func(*GetBlocksMessage)
	OnGetMempool  func(*GetMempoolMessage)
}

type Peer struct {
	conn      net.Conn
	callbacks *PeerCallbacks
}

func NewPeer(callbacks *PeerCallbacks) *Peer {
	return &Peer{
		callbacks: callbacks,
	}
}

func (peer *Peer) AttachConn(conn net.Conn) {
	peer.conn = conn
}

func (peer *Peer) Run() {
	peer.callbacks.OnReady()

	jsonDecoder := json.NewDecoder(peer.conn)

	for {
		var message Message
		err := jsonDecoder.Decode(&message)
		if err != nil {
			log.WithError(err).Error("error decoding a message")
			return
		}

		go peer.handleMessage(&message)
	}
}

func (peer *Peer) handleMessage(message *Message) {
	log.WithFields(log.Fields{
		"peer":    peer,
		"message": message,
	}).Info("message received")

	switch message.Type {
	case "whoami":
		var m *WhoamiMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnWhoami(m)
	case "inv":
		var m *InvMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnInv(m)
	case "getdata":
		var m *GetDataMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnGetData(m)
	case "notfound":
		var m *NotFoundMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnNotFound(m)
	case "block":
		var m *BlockMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnBlock(m)
	case "transaction":
		var m *TransactionMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnTransaction(m)
	case "getblocks":
		var m *GetBlocksMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnGetBlocks(m)
	case "getmempool":
		var m *GetMempoolMessage
		json.Unmarshal([]byte(message.Message), m)
		peer.callbacks.OnGetMempool(m)
	}
}

func (peer *Peer) Send(messageType string, message interface{}) error {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "error marshaling the message")
	}

	messageString := string(messageBytes)

	finalMessage := Message{
		Magic:     consensus.NETWORK_MAGIC_NUMBER,
		Type:      messageType,
		Timestamp: time.Now(),
		Message:   messageString,
	}

	finalMessageBytes, err := json.Marshal(finalMessage)

	if err != nil {
		return errors.Wrap(err, "error marshaling the final message")
	}

	_, err = peer.conn.Write(finalMessageBytes)
	if err != nil {
		return errors.Wrap(err, "error writing the message")
	}

	log.WithFields(log.Fields{
		"peer":    peer,
		"message": finalMessage,
	}).Info("message sent")

	return nil
}
