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
		var subMessage json.RawMessage
		message := Message{
			Message: &subMessage,
		}
		err := jsonDecoder.Decode(&message)
		if err != nil {
			log.WithError(err).Error("error decoding a message")
			return
		}

		go peer.handleMessage(&message, &subMessage)
	}
}

func (peer *Peer) handleMessage(message *Message, subMessage *json.RawMessage) {
	log.WithFields(log.Fields{
		"peer":    peer,
		"message": message,
	}).Info("message received")

	switch message.Type {
	case "whoami":
		var m *WhoamiMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnWhoami(m)
	case "inv":
		var m *InvMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnInv(m)
	case "getdata":
		var m *GetDataMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnGetData(m)
	case "notfound":
		var m *NotFoundMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnNotFound(m)
	case "block":
		var m *BlockMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnBlock(m)
	case "transaction":
		var m *TransactionMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnTransaction(m)
	case "getblocks":
		var m *GetBlocksMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnGetBlocks(m)
	case "getmempool":
		var m *GetMempoolMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		peer.callbacks.OnGetMempool(m)
	}
}

func (peer *Peer) Send(messageType string, message interface{}) error {
	finalMessage := Message{
		Magic:     consensus.NETWORK_MAGIC_NUMBER,
		Type:      messageType,
		Timestamp: time.Now(),
		Message:   message,
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
