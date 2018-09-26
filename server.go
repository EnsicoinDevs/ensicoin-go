package main

import (
	"fmt"
	"github.com/johynpapin/ensicoin-go/blockchain"
	"github.com/johynpapin/ensicoin-go/consensus"
	"github.com/johynpapin/ensicoin-go/mempool"
	"github.com/johynpapin/ensicoin-go/network"
	log "github.com/sirupsen/logrus"
	"net"
)

type ServerPeer struct {
	*network.Peer
	Ingoing   bool
	Connected bool
}

func newServerPeer(conn net.Conn) *ServerPeer {
	peer := &ServerPeer{}

	peer.Peer = network.NewPeer(&network.PeerCallbacks{
		OnReady:  peer.onReady,
		OnWhoami: peer.onWhoami,
	})

	peer.Peer.AttachConn(conn)

	return peer
}

func newIngoingServerPeer(conn net.Conn) *ServerPeer {
	peer := newServerPeer(conn)

	peer.Ingoing = true

	return peer
}

func newOutgoingServerPeer(conn net.Conn) *ServerPeer {
	peer := newServerPeer(conn)

	peer.Ingoing = false

	return peer
}

type Server struct {
	peers    []*ServerPeer
	listener net.Listener
}

func NewServer(blockchain *blockchain.Blockchain, mempool *mempool.Mempool) *Server {
	server := &Server{}

	return server
}

func (server *Server) registerAndRunPeer(peer *ServerPeer) {
	server.peers = append(server.peers, peer)
	go peer.Run()
}

func (server *Server) Start() {
	var err error
	server.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", consensus.INGOING_PORT))
	if err != nil {
		log.WithError(err).Panic("error launching the tcp server")
	}

	for {
		conn, err := server.listener.Accept()
		if err != nil {
			log.WithError(err).Error("error accepting a new connection")
		} else {
			log.Info("we have a new ingoing peer")
			server.registerAndRunPeer(newIngoingServerPeer(conn))
		}
	}
}

func (server *Server) Stop() {
	server.listener.Close()
}

func (server *Server) RegisterOutgoingPeer(conn net.Conn) {
	server.registerAndRunPeer(newOutgoingServerPeer(conn))
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
}
