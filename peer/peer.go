package peer

import (
	"encoding/json"
	"github.com/EnsicoinDevs/ensicoincoin/consensus"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type PeerCallbacks struct {
	OnReady      func()
	OnWhoami     func(*network.WhoamiMessage)
	OnInv        func(*network.InvMessage)
	OnGetData    func(*network.GetDataMessage)
	OnNotFound   func(*network.NotFoundMessage)
	OnBlock      func(*network.BlockMessage)
	OnTx         func(*network.TxMessage)
	OnGetBlocks  func(*network.GetBlocksMessage)
	OnGetMempool func(*network.GetMempoolMessage)
}

type Config struct {
	Callbacks PeerCallbacks
}

type Peer struct {
	conn net.Conn

	config  Config
	ingoing bool
}

func newPeer(config *Config, ingoing bool) *Peer {
	return &Peer{
		config:  *config,
		ingoing: ingoing,
	}
}

func NewIngoingPeer(config *Config) *Peer {
	return newPeer(config, true)
}

func NewOutgoingPeer(config *Config) *Peer {
	return newPeer(config, false)
}

func (peer *Peer) Ingoing() bool {
	return peer.ingoing
}

func (peer *Peer) AttachConn(conn net.Conn) {
	peer.conn = conn

	go peer.run()
}

func (peer *Peer) run() {
	peer.config.Callbacks.OnReady()

	jsonDecoder := json.NewDecoder(peer.conn)

	for {
		var subMessage json.RawMessage
		message := network.Message{
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

func (peer *Peer) handleMessage(message *network.Message, subMessage *json.RawMessage) {
	log.WithFields(log.Fields{
		"peer":    peer,
		"message": message,
	}).Info("message received")

	switch message.Type {
	case "whoami":
		var m *network.WhoamiMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnWhoami != nil {
			peer.config.Callbacks.OnWhoami(m)
		}
	case "inv":
		var m *network.InvMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnInv != nil {
			peer.config.Callbacks.OnInv(m)
		}
	case "getdata":
		var m *network.GetDataMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnGetData != nil {
			peer.config.Callbacks.OnGetData(m)
		}
	case "notfound":
		var m *network.NotFoundMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnNotFound != nil {
			peer.config.Callbacks.OnNotFound(m)
		}
	case "block":
		var m *network.BlockMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnBlock != nil {
			peer.config.Callbacks.OnBlock(m)
		}
	case "transaction":
		var m *network.TxMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnTx != nil {
			peer.config.Callbacks.OnTx(m)
		}
	case "getblocks":
		var m *network.GetBlocksMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnGetBlocks != nil {
			peer.config.Callbacks.OnGetBlocks(m)
		}
	case "getmempool":
		var m *network.GetMempoolMessage
		err := json.Unmarshal(*subMessage, &m)
		if err != nil {
			log.WithError(err).Error("error unmarshaling a message")
			return
		}

		if peer.config.Callbacks.OnGetMempool != nil {
			peer.config.Callbacks.OnGetMempool(m)
		}
	}
}

func (peer *Peer) Send(messageType string, message interface{}) error {
	finalMessage := network.Message{
		Magic:     consensus.NETWORK_MAGIC_NUMBER,
		Type:      messageType,
		Timestamp: time.Now(),
		Message:   message,
	}

	finalMessageBytes, err := json.Marshal(&finalMessage)

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
