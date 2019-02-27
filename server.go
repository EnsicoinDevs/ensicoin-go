package main

import (
	"encoding/hex"
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
	"github.com/EnsicoinDevs/ensicoincoin/miner"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"github.com/EnsicoinDevs/ensicoincoin/peer"
	"github.com/EnsicoinDevs/ensicoincoin/sssync"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"sync"
)

type Server struct {
	blockchain   *blockchain.Blockchain
	mempool      *mempool.Mempool
	miner        *miner.Miner
	synchronizer *sssync.Synchronizer

	listener net.Listener

	mutex *sync.Mutex
	peers []*peer.Peer

	quit chan struct{}
}

func NewServer(blockchain *blockchain.Blockchain, mempool *mempool.Mempool, miner *miner.Miner) *Server {
	server := &Server{
		blockchain: blockchain,
		mempool:    mempool,
		miner:      miner,

		mutex: &sync.Mutex{},

		quit: make(chan struct{}),
	}

	server.synchronizer = sssync.NewSynchronizer(&sssync.Config{
		Broadcast: server.Broadcast,
	}, server.blockchain, server.mempool, server.miner)
	server.miner.Config.ProcessBlock = server.synchronizer.ProcessBlock

	return server
}

func (server *Server) Start() {
	server.synchronizer.Start()

	var err error
	server.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("port")))
	if err != nil {
		log.WithError(err).Panic("error launching the tcp server")
	}

	for {
		conn, err := server.listener.Accept()
		if err != nil {
			log.WithError(err).Error("error accepting a new connection")
		} else {
			log.Info("we have a new ingoing peer")
			peer.NewPeer(conn, &peer.Config{
				Callbacks: peer.PeerCallbacks{
					OnReady:        server.onReady,
					OnDisconnected: server.onDisconnected,
					OnGetData:      server.onGetData,
					OnGetBlocks:    server.onGetBlocks,
					OnInv:          server.onInv,
					OnBlock:        server.onBlock,
				},
			}, true).Start()
		}
	}
}

func (server *Server) Stop() {
	close(server.quit)

	server.listener.Close()
}

func (server *Server) ProcessTx(message *network.TxMessage) {
	server.mempool.ProcessTx(blockchain.NewTxFromTxMessage(message))
}

func (server *Server) ConnectTo(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	peer.NewPeer(conn, &peer.Config{
		Callbacks: peer.PeerCallbacks{
			OnReady:        server.onReady,
			OnDisconnected: server.onDisconnected,
			OnGetData:      server.onGetData,
			OnGetBlocks:    server.onGetBlocks,
			OnInv:          server.onInv,
			OnBlock:        server.onBlock,
		},
	}, false).Start()

	return nil
}

func (server *Server) Broadcast(message network.Message) {
	for _, peer := range server.peers {
		peer.Send(message)
	}
}

func (server *Server) addPeer(peer *peer.Peer) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.peers = append(server.peers, peer)
}

func (server *Server) onReady(peer *peer.Peer) {
	log.WithField("peer", peer).Debug("a peer is ready")
	server.addPeer(peer)
	server.synchronizer.HandleReadyPeer(peer)
}

func (server *Server) onDisconnected(peer *peer.Peer) {
	server.mutex.Lock()
	for i, otherPeer := range server.peers {
		if otherPeer == peer {
			server.peers = append(server.peers[:i], server.peers[i+1:]...)
		}
	}
	server.mutex.Unlock()

	server.synchronizer.HandleDisconnectedPeer(peer)
}

func (server *Server) onInv(peer *peer.Peer, message *network.InvMessage) {
	for _, inv := range message.Inventory {
		if inv.InvType == network.INV_VECT_BLOCK {
			server.synchronizer.HandleBlockInvVect(peer, inv)
		} else if inv.InvType == network.INV_VECT_TX {

		} else {
			// TODO: handle this error
		}
	}
}

func (server *Server) onBlock(peer *peer.Peer, message *network.BlockMessage) {
	log.WithFields(log.Fields{
		"hash":   hex.EncodeToString(message.Header.Hash()[:]),
		"height": message.Header.Height,
	}).Info("received a block")

	server.synchronizer.HandleBlock(peer, message)
}

func (server *Server) onGetData(peer *peer.Peer, message *network.GetDataMessage) {
	for _, invVect := range message.Inventory {
		switch invVect.InvType {
		case network.INV_VECT_BLOCK:
			block, err := server.blockchain.FindBlockByHash(invVect.Hash)
			if err != nil {
				log.WithFields(log.Fields{
					"peer": peer,
				}).WithError(err).Error("error finding a block by hash")
			}

			if block == nil {
				// TODO: send a NotFound message
			}

			peer.Send(block.Msg)
		case network.INV_VECT_TX:

		default:
			log.Error("unknown inv vect type")
		}
	}
}

func (server *Server) onGetBlocks(peer *peer.Peer, message *network.GetBlocksMessage) {
	var startAt *utils.Hash

	for _, hash := range message.BlockLocator {
		block, err := server.blockchain.FindBlockByHash(hash)
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
		startAt = server.blockchain.GenesisBlock.Hash()
	}

	var inventory []*network.InvVect

	hashes, err := server.blockchain.FindBlockHashesStartingAt(startAt)
	if err != nil {
		log.WithField("startAt", startAt).WithError(err).Error("error finding the block hashes")
		return
	}

	for _, hash := range hashes {
		inventory = append(inventory, &network.InvVect{
			InvType: network.INV_VECT_BLOCK,
			Hash:    hash,
		})

	}

	peer.Send(&network.InvMessage{
		Inventory: inventory,
	})
}
