package peer

import (
	"github.com/EnsicoinDevs/eccd/network"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"time"
)

type PeerCallbacks struct {
	OnReady        func(*Peer)
	OnDisconnected func(*Peer)
	OnWhoami       func(*Peer, *network.WhoamiMessage)
	OnWhoamiAck    func(*Peer, *network.WhoamiAckMessage)
	OnInv          func(*Peer, *network.InvMessage)
	OnGetData      func(*Peer, *network.GetDataMessage)
	OnNotFound     func(*Peer, *network.NotFoundMessage)
	OnBlock        func(*Peer, *network.BlockMessage)
	OnTx           func(*Peer, *network.TxMessage)
	OnGetBlocks    func(*Peer, *network.GetBlocksMessage)
	OnGetMempool   func(*Peer, *network.GetMempoolMessage)
}

type Config struct {
	Callbacks PeerCallbacks
}

type Peer struct {
	conn net.Conn

	config  Config
	ingoing bool

	sending chan network.Message

	quit chan struct{}
}

func (peer Peer) String() string {
	return peer.RemoteAddr().String()
}

func NewPeer(conn net.Conn, config *Config, ingoing bool) *Peer {
	return &Peer{
		conn:    conn,
		config:  *config,
		ingoing: ingoing,

		sending: make(chan network.Message, 1),

		quit: make(chan struct{}),
	}
}

func (peer *Peer) Ingoing() bool {
	return peer.ingoing
}

func (peer *Peer) Outgoing() bool {
	return !peer.ingoing
}

func (peer *Peer) Start() error {
	go peer.handleOutgoingMessages()

	err := peer.negotiate()
	if err != nil {
		return err
	}

	peer.config.Callbacks.OnReady(peer)

	go peer.startPingLoop()

	go func() {
		var message network.Message

		for {
			message, err = network.ReadMessage(peer.conn)
			select {
			case <-peer.quit:
				return
			default:
			}
			if err != nil {
				if err == io.EOF {
					peer.conn.Close()

					close(peer.quit)

					if peer.config.Callbacks.OnDisconnected != nil {
						peer.config.Callbacks.OnDisconnected(peer)
					}
					return
				} else {
					log.WithError(err).Error("error reading a message")
					continue
				}
			}

			go peer.handleMessage(message)
		}
	}()

	return nil
}

func (peer *Peer) Stop() error {
	close(peer.quit)
	peer.conn.Close() // TODO: improve

	if peer.config.Callbacks.OnDisconnected != nil {
		peer.config.Callbacks.OnDisconnected(peer)
	}

	return nil
}

func (peer *Peer) RemoteAddr() net.Addr {
	return peer.conn.RemoteAddr()
}

func (peer *Peer) negotiate() error {
	if peer.Outgoing() {
		// > whoami
		err := peer.send(&network.WhoamiMessage{
			Version: 0,
			From: &network.Address{
				Timestamp: time.Now(),
				IP:        net.IPv4(0, 0, 0, 0),
			},
			Services: []string{"node"},
		})
		if err != nil {
			return err
		}

		// < whoami
		message, err := network.ReadMessage(peer.conn)
		if err != nil {
			return err
		}

		whoami, ok := message.(*network.WhoamiMessage)
		if !ok {
			return errors.New("peer is insane")
		}

		_ = whoami

		// < whomiack
		message, err = network.ReadMessage(peer.conn)
		if err != nil {
			return err
		}

		whoamiack, ok := message.(*network.WhoamiAckMessage)
		if !ok {
			return errors.New("peer is insane")
		}

		_ = whoamiack

		// > whoamiack
		err = peer.send(&network.WhoamiAckMessage{})
		if err != nil {
			return err
		}
	} else {
		// < whoami
		message, err := network.ReadMessage(peer.conn)
		if err != nil {
			return err
		}

		whoami, ok := message.(*network.WhoamiMessage)
		if !ok {
			return errors.New("peer is insane")
		}

		_ = whoami

		// > whoami
		err = peer.send(&network.WhoamiMessage{
			Version: 0,
			From: &network.Address{
				Timestamp: time.Now(),
				IP:        net.IPv4(0, 0, 0, 0),
			},
			Services: []string{"node"},
		})
		if err != nil {
			return err
		}

		// > whoamiack
		err = peer.send(&network.WhoamiAckMessage{})
		if err != nil {
			return err
		}

		// < whomiack
		message, err = network.ReadMessage(peer.conn)
		if err != nil {
			return err
		}

		whoamiack, ok := message.(*network.WhoamiAckMessage)
		if !ok {
			return errors.New("peer is insane")
		}

		_ = whoamiack
	}
	return nil
}

func (peer *Peer) handleMessage(message network.Message) {
	log.WithFields(log.Fields{
		"peer":    peer,
		"message": message.MsgType(),
	}).Debug("message received")

	switch message.MsgType() {
	case "inv":
		m := message.(*network.InvMessage)

		if peer.config.Callbacks.OnInv != nil {
			peer.config.Callbacks.OnInv(peer, m)
		}
	case "getdata":
		m := message.(*network.GetDataMessage)

		if peer.config.Callbacks.OnGetData != nil {
			peer.config.Callbacks.OnGetData(peer, m)
		}
	case "notfound":
		m := message.(*network.NotFoundMessage)

		if peer.config.Callbacks.OnNotFound != nil {
			peer.config.Callbacks.OnNotFound(peer, m)
		}
	case "block":
		m := message.(*network.BlockMessage)

		if peer.config.Callbacks.OnBlock != nil {
			peer.config.Callbacks.OnBlock(peer, m)
		}
	case "tx":
		m := message.(*network.TxMessage)

		if peer.config.Callbacks.OnTx != nil {
			peer.config.Callbacks.OnTx(peer, m)
		}
	case "getblocks":
		m := message.(*network.GetBlocksMessage)

		if peer.config.Callbacks.OnGetBlocks != nil {
			peer.config.Callbacks.OnGetBlocks(peer, m)
		}
	case "getmempool":
		m := message.(*network.GetMempoolMessage)

		if peer.config.Callbacks.OnGetMempool != nil {
			peer.config.Callbacks.OnGetMempool(peer, m)
		}
	case "2plus2is4":
		peer.Send(&network.MinusOneThatsThreeMessage{})
	}
}

func (peer *Peer) Send(message network.Message) error {
	peer.sending <- message

	return nil
}

func (peer *Peer) send(message network.Message) error {
	err := network.WriteMessage(peer.conn, message)
	if err != nil {
		return errors.Wrap(err, "error writing the message")
	}

	return nil
}

func (peer *Peer) startPingLoop() {
	ticker := time.NewTicker(42 * time.Second)

	for {
		select {
		case <-ticker.C:
			peer.Send(&network.TwoPlusTwoIsFourMessage{})
		case <-peer.quit:
			return
		}
	}
}

func (peer *Peer) handleOutgoingMessages() {
	for {
		select {
		case message := <-peer.sending:
			log.WithFields(log.Fields{
				"peer":    peer,
				"message": message.MsgType(),
			}).Debug("sending message")

			if err := peer.send(message); err != nil {
				log.WithError(err).Error("error sending a message")
			}
		case <-peer.quit:
			return
		}
	}
}
