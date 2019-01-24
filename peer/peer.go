package peer

import (
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
)

type PeerCallbacks struct {
	OnReady      func()
	OnWhoami     func(*network.WhoamiMessage)
	OnWhoamiAck  func(*network.WhoamiAckMessage)
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

func (peer Peer) String() string {
	return fmt.Sprintf("Peer[addr=%s]", peer.conn.RemoteAddr().String())
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

	var message network.Message
	var err error

	for {
		message, err = network.ReadMessage(peer.conn)
		if err != nil {
			log.WithError(err).Error("error reading a message")
			peer.conn.Close() // TODO: fixme
			return
		}

		go peer.handleMessage(message)
	}
}

func (peer *Peer) handleMessage(message network.Message) {
	log.WithFields(log.Fields{
		"peer":    peer,
		"message": message,
	}).Info("message received")

	switch message.MsgType() {
	case "whoami":
		m := message.(*network.WhoamiMessage)

		if peer.config.Callbacks.OnWhoami != nil {
			peer.config.Callbacks.OnWhoami(m)
		}
	case "whoamiack":
		m := message.(*network.WhoamiAckMessage)

		if peer.config.Callbacks.OnWhoamiAck != nil {
			peer.config.Callbacks.OnWhoamiAck(m)
		}
	case "inv":
		m := message.(*network.InvMessage)

		if peer.config.Callbacks.OnInv != nil {
			peer.config.Callbacks.OnInv(m)
		}
	case "getdata":
		m := message.(*network.GetDataMessage)

		if peer.config.Callbacks.OnGetData != nil {
			peer.config.Callbacks.OnGetData(m)
		}
	case "notfound":
		m := message.(*network.NotFoundMessage)

		if peer.config.Callbacks.OnNotFound != nil {
			peer.config.Callbacks.OnNotFound(m)
		}
	case "block":
		m := message.(*network.BlockMessage)

		if peer.config.Callbacks.OnBlock != nil {
			peer.config.Callbacks.OnBlock(m)
		}
	case "tx":
		m := message.(*network.TxMessage)

		if peer.config.Callbacks.OnTx != nil {
			peer.config.Callbacks.OnTx(m)
		}
	case "getblocks":
		m := message.(*network.GetBlocksMessage)

		if peer.config.Callbacks.OnGetBlocks != nil {
			peer.config.Callbacks.OnGetBlocks(m)
		}
	case "getmempool":
		m := message.(*network.GetMempoolMessage)

		if peer.config.Callbacks.OnGetMempool != nil {
			peer.config.Callbacks.OnGetMempool(m)
		}
	}
}

func (peer *Peer) Send(message network.Message) error {
	err := network.WriteMessage(peer.conn, message)
	if err != nil {
		return errors.Wrap(err, "error writing the message")
	}

	log.WithFields(log.Fields{
		"peer":    peer,
		"message": message,
	}).Info("message sent")

	return nil
}
