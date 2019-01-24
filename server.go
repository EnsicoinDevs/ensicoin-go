package main

import (
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
	"github.com/EnsicoinDevs/ensicoincoin/miner"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/peer"
	"github.com/EnsicoinDevs/ensicoincoin/sync"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type invWithPeer struct {
	msg  *network.InvMessage
	peer *ServerPeer
}

type ServerPeer struct {
	*peer.Peer

	server *Server
}

func (server *Server) newServerPeer() *ServerPeer {
	return &ServerPeer{
		server: server,
	}
}

func (sp *ServerPeer) newConfig() *peer.Config {
	return &peer.Config{
		Callbacks: peer.PeerCallbacks{
			OnReady:     sp.onReady,
			OnWhoami:    sp.onWhoami,
			OnWhoamiAck: sp.onWhoamiAck,
			OnInv:       sp.onInv,
			OnGetBlocks: sp.onGetBlocks,
			OnGetData:   sp.onGetData,
			OnTx:        sp.onTx,
			OnBlock:     sp.onBlock,
		},
	}
}

func (server *Server) newIngoingServerPeer(conn net.Conn) *ServerPeer {
	sp := server.newServerPeer()

	sp.Peer = peer.NewIngoingPeer(sp.newConfig())

	sp.AttachConn(conn)

	return sp
}

func (server *Server) newOutgoingServerPeer(conn net.Conn) *ServerPeer {
	sp := server.newServerPeer()

	sp.Peer = peer.NewOutgoingPeer(sp.newConfig())

	sp.AttachConn(conn)

	return sp
}

type Server struct {
	blockchain *blockchain.Blockchain
	mempool    *mempool.Mempool
	miner      *miner.Miner
	sync       *sync.Synchronizer

	invs chan *invWithPeer

	peers        []*ServerPeer
	peersManager *peer.PeersManager
	listener     net.Listener

	synced bool
}

func NewServer(blockchain *blockchain.Blockchain, mempool *mempool.Mempool, miner *miner.Miner) *Server {
	server := &Server{
		blockchain: blockchain,
		mempool:    mempool,
		miner:      miner,

		invs: make(chan *invWithPeer),

		peersManager: peer.NewPeersManager(),
	}

	server.sync = sync.NewSynchronizer(server.blockchain, server.mempool, server.peersManager, server.miner)

	return server
}

func (server *Server) registerPeer(peer *ServerPeer) {
	server.peers = append(server.peers, peer)
	server.peersManager.Add(peer.Peer)
}

func (server *Server) syncWith(peer *ServerPeer) {
	longestChain, err := server.blockchain.FindLongestChain()
	if err != nil {
		log.WithError(err).Error("error finding the longest chain")
		return
	}

	peer.Send(&network.GetBlocksMessage{
		BlockLocator: []*utils.Hash{longestChain.Hash()},
		HashStop:     utils.NewHash(nil),
	})

	server.synced = true
}

func (server *Server) Start() {
	server.sync.Start()

	var err error
	server.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", peerPort))
	if err != nil {
		log.WithError(err).Panic("error launching the tcp server")
	}

	go server.handleInvs()

	for {
		conn, err := server.listener.Accept()
		if err != nil {
			log.WithError(err).Error("error accepting a new connection")
		} else {
			log.Info("we have a new ingoing peer")
			server.registerPeer(server.newIngoingServerPeer(conn))
		}
	}
}

func (server *Server) handleInvs() {
	for inv := range server.invs {
		var inventory []*network.InvVect

		for _, invVect := range inv.msg.Inventory {
			switch invVect.InvType {
			case network.INV_VECT_BLOCK:
				block, err := server.blockchain.FindBlockByHash(invVect.Hash)
				if err != nil {
					log.WithError(err).WithField("hash", invVect.Hash).Error("error finding a block")
				}

				if block == nil {
					inventory = append(inventory, &network.InvVect{
						InvType: network.INV_VECT_BLOCK,
						Hash:    invVect.Hash,
					})
				}
			case network.INV_VECT_TX:
				tx := server.mempool.FindTxByHash(invVect.Hash)
				if tx == nil {
					inventory = append(inventory, &network.InvVect{
						InvType: network.INV_VECT_TX,
						Hash:    invVect.Hash,
					})
				}
			default:
				log.Error("unknown inv vect type")
			}
		}

		if len(inventory) > 0 {
			inv.peer.Send(&network.GetDataMessage{
				Inventory: inventory,
			})
		}
	}
}

func (server *Server) Stop() {
	server.listener.Close()
}

func (server *Server) RegisterOutgoingPeer(conn net.Conn) {
	server.registerPeer(server.newOutgoingServerPeer(conn))
}

func (sp *ServerPeer) onReady() {
	if !sp.Ingoing() {
		sp.Send(&network.WhoamiMessage{
			Version: 0,
			From: &network.Address{
				Timestamp: time.Now(),
				IP:        net.IPv4(0, 0, 0, 0),
			},
			Services: []string{"node"},
		})
	}
}

func (sp *ServerPeer) onWhoami(message *network.WhoamiMessage) {
	log.WithField("peer", sp).Info("whoami received")

	if sp.Ingoing() {
		sp.Send(&network.WhoamiMessage{
			Version: 0,
			From: &network.Address{
				Timestamp: time.Now(),
				IP:        net.IPv4(0, 0, 0, 0),
			},
			Services: []string{"node"},
		})
		// sp.Connected = true
	} else {
		// sp.Connected = true
	}

	sp.Send(&network.WhoamiAckMessage{})

	if !sp.server.synced {
		sp.server.syncWith(sp)
	}
}

func (sp *ServerPeer) onWhoamiAck(message *network.WhoamiAckMessage) {
	log.WithField("peer", sp).Info("connection established")
}

func (peer *ServerPeer) onInv(message *network.InvMessage) {
	peer.server.invs <- &invWithPeer{
		msg:  message,
		peer: peer,
	}
}

func (server *Server) ProcessBlock(message *network.BlockMessage) {
	server.sync.ProcessBlock(message)
}

func (server *Server) ProcessTx(message *network.TxMessage) {
	tx := blockchain.NewTxFromTxMessage(message)
	acceptedTxs := server.mempool.ProcessTx(tx)
	if len(acceptedTxs) > 0 {
		server.broadcastTxs(acceptedTxs)
	}
}

func (peer *ServerPeer) onBlock(message *network.BlockMessage) {
	peer.server.sync.ProcessBlock(message)
}

func (server *Server) broadcastTx(tx *blockchain.Tx, sourcePeer *ServerPeer) {
	for _, peer := range server.peers {
		if peer != sourcePeer {
			peer.Send(&network.InvMessage{
				Inventory: []*network.InvVect{&network.InvVect{
					InvType: network.INV_VECT_TX,
					Hash:    tx.Hash(),
				}},
			})
		}
	}
}

func (server *Server) broadcastBlock(block *network.BlockMessage, sourcePeer *ServerPeer) {
	for _, peer := range server.peers {
		if peer != sourcePeer {
			peer.Send(&network.InvMessage{
				Inventory: []*network.InvVect{&network.InvVect{
					InvType: network.INV_VECT_BLOCK,
					Hash:    block.Header.Hash(),
				}},
			})
		}
	}
}

func (server *Server) broadcastTxs(txHashes []*utils.Hash) {
	var inventory []*network.InvVect

	for _, hash := range txHashes {
		inventory = append(inventory, &network.InvVect{
			InvType: network.INV_VECT_TX,
			Hash:    hash,
		})
	}

	invMessage := &network.InvMessage{
		Inventory: inventory,
	}

	for _, peer := range server.peers {
		peer.Send(invMessage)
	}
}

func (peer *ServerPeer) onTx(message *network.TxMessage) {
	peer.server.ProcessTx(message)
}

func (peer *ServerPeer) onGetData(message *network.GetDataMessage) {
	for _, invVect := range message.Inventory {
		switch invVect.InvType {
		case network.INV_VECT_BLOCK:
			block, err := peer.server.blockchain.FindBlockByHash(invVect.Hash)
			if err != nil {
				log.WithFields(log.Fields{
					"peer": peer,
				}).WithError(err).Error("error finding a block by hash")
			}

			if block == nil {
				continue // TODO: send a NotFound message
			}

			peer.Send(block.Msg)

			log.Info("block sent")
		case network.INV_VECT_TX:

		default:
			log.Error("unknown inv vect type")
		}
	}
}

func (peer *ServerPeer) onGetBlocks(message *network.GetBlocksMessage) {
	var startAt *utils.Hash

	for _, hash := range message.BlockLocator {
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

		startAt = hash
		break
	}

	if startAt == nil {
		startAt = peer.server.blockchain.GenesisBlock.Hash()
	}

	var inventory []*network.InvVect

	hashes, err := peer.server.blockchain.FindBlockHashesStartingAt(startAt)
	if err != nil {
		log.WithField("startAt", startAt).WithError(err).Error("error finding the block hashes")
		return
	}

	for _, hash := range hashes {
		log.WithField("hash", *hash).Debug("adding a hash")

		inventory = append(inventory, &network.InvVect{
			InvType: network.INV_VECT_BLOCK,
			Hash:    hash,
		})
	}

	peer.Send(&network.InvMessage{
		Inventory: inventory,
	})
}
