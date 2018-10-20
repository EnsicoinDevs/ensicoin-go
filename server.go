package main

import (
	"fmt"
	"github.com/EnsicoinDevs/ensicoin-go/blockchain"
	"github.com/EnsicoinDevs/ensicoin-go/consensus"
	"github.com/EnsicoinDevs/ensicoin-go/mempool"
	"github.com/EnsicoinDevs/ensicoin-go/network"
	log "github.com/sirupsen/logrus"
	"net"
)

type invWithPeer struct {
	hashes []string
	peer   *ServerPeer
}

type ServerPeer struct {
	*network.Peer
	Ingoing   bool
	Connected bool

	server *Server
}

func (server *Server) newServerPeer(conn net.Conn) *ServerPeer {
	peer := &ServerPeer{
		server: server,
	}

	peer.Peer = network.NewPeer(&network.PeerCallbacks{
		OnReady:     peer.onReady,
		OnWhoami:    peer.onWhoami,
		OnInv:       peer.onInv,
		OnGetBlocks: peer.onGetBlocks,
	})

	peer.Peer.AttachConn(conn)

	return peer
}

func (server *Server) newIngoingServerPeer(conn net.Conn) *ServerPeer {
	peer := server.newServerPeer(conn)

	peer.Ingoing = true

	return peer
}

func (server *Server) newOutgoingServerPeer(conn net.Conn) *ServerPeer {
	peer := server.newServerPeer(conn)

	peer.Ingoing = false

	return peer
}

type Server struct {
	blockchain *blockchain.Blockchain
	mempool    *mempool.Mempool

	blocksInvs       chan *invWithPeer
	transactionsInvs chan *invWithPeer

	peers    []*ServerPeer
	listener net.Listener

	synced bool
}

func NewServer(blockchain *blockchain.Blockchain, mempool *mempool.Mempool) *Server {
	server := &Server{
		blockchain: blockchain,
		mempool:    mempool,

		blocksInvs:       make(chan *invWithPeer),
		transactionsInvs: make(chan *invWithPeer),
	}

	return server
}

func (server *Server) registerAndRunPeer(peer *ServerPeer) {
	server.peers = append(server.peers, peer)
	go peer.Run()
}

func (server *Server) syncWith(peer *ServerPeer) {
	longestChain, err := server.blockchain.FindLongestChain()
	if err != nil {
		log.WithError(err).Error("error finding the longest chain")
		return
	}

	peer.Send("getblocks", &network.GetBlocksMessage{
		Hashes: []string{longestChain.Hash},
	})

	server.synced = true
}

func (server *Server) Start() {
	var err error
	server.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", consensus.INGOING_PORT))
	if err != nil {
		log.WithError(err).Panic("error launching the tcp server")
	}

	go server.handleBlocksInvs()
	go server.handleTransactionsInvs()

	for {
		conn, err := server.listener.Accept()
		if err != nil {
			log.WithError(err).Error("error accepting a new connection")
		} else {
			log.Info("we have a new ingoing peer")
			server.registerAndRunPeer(server.newIngoingServerPeer(conn))
		}
	}
}

func (server *Server) handleBlocksInvs() {
	for inv := range server.blocksInvs {
		invToSend := network.InvMessage{
			Type: "b",
		}

		for _, hash := range inv.hashes {
			block, err := server.blockchain.FindBlockByHash(hash)
			if err != nil {
				log.WithError(err).WithField("hash", hash).Error("error finding a block")
			}

			if block == nil {
				invToSend.Hashes = append(invToSend.Hashes, hash)
			}
		}

		if len(invToSend.Hashes) > 0 {
			inv.peer.Send("getdata", &network.GetDataMessage{
				Inv: invToSend,
			})
		}
	}
}

func (server *Server) handleTransactionsInvs() {
	for _ = range server.transactionsInvs {
	}
}

func (server *Server) Stop() {
	server.listener.Close()
}

func (server *Server) RegisterOutgoingPeer(conn net.Conn) {
	server.registerAndRunPeer(server.newOutgoingServerPeer(conn))
}

func (peer *ServerPeer) onReady() {
	if !peer.Ingoing {
		peer.Send("whoami", &network.WhoamiMessage{
			Version: 0,
		})
	}
}

func (peer *ServerPeer) onWhoami(message *network.WhoamiMessage) {
	log.WithField("peer", peer).Info("whoami received")

	if peer.Ingoing {
		peer.Send("whoami", &network.WhoamiMessage{
			Version: 0,
		})

		peer.Connected = true
	} else {
		peer.Connected = true
	}

	log.WithField("peer", peer).Info("connection established")

	if !peer.server.synced {
		peer.server.syncWith(peer)
	}
}

func (peer *ServerPeer) onInv(message *network.InvMessage) {
	if message.Type == "b" {
		peer.server.blocksInvs <- &invWithPeer{
			hashes: message.Hashes,
			peer:   peer,
		}
	} else if message.Type == "t" {
		peer.server.transactionsInvs <- &invWithPeer{
			hashes: message.Hashes,
			peer:   peer,
		}
	} else {
		log.WithFields(log.Fields{
			"peer":    peer,
			"message": message,
		}).Error("unknown inv type")
	}
}

func (peer *ServerPeer) onBlock(message *network.BlockMessage) {
}

func (peer *ServerPeer) onTransaction(message *network.TransactionMessage) {

}

func (peer *ServerPeer) onGetBlocks(message *network.GetBlocksMessage) {
	var startAt string

	for _, hash := range message.Hashes {
		block, err := peer.server.blockchain.FindBlockByHash(hash)
		if err != nil {
			log.WithFields(log.Fields{
				"peer": peer,
			}).WithError(err).Error("error finding a block by hash")
			continue
		}

		if block == nil {
			continue
		}

		startAt = block.Hash
		break
	}

	if startAt == "" {
		startAt = peer.server.blockchain.GenesisBlock.Hash
	}

	invToSend := &network.InvMessage{
		Type: "b",
	}

	hashes, err := peer.server.blockchain.FindBlockHashesStartingAt(startAt)
	if err != nil {
		log.WithField("startAt", startAt).WithError(err).Error("error finding the block hashes")
		return
	}

	for _, hash := range hashes {
		invToSend.Hashes = append(invToSend.Hashes, hash)
	}

	peer.Send("inv", invToSend)
}
