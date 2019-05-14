package main

import (
	"fmt"
	"github.com/EnsicoinDevs/eccd/blockchain"
	pb "github.com/EnsicoinDevs/eccd/rpc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"sync"
)

type NotificationType int

const (
	NOTIFICATION_PUSHED_BLOCK = iota
	NOTIFICATION_POPPED_BLOCK
)

type Notification struct {
	Type  NotificationType
	Block *blockchain.Block
}

type Notifier struct {
	mutex sync.RWMutex
	chans map[chan *Notification]struct{}
}

func NewNotifier() *Notifier {
	return &Notifier{
		mutex: sync.RWMutex{},
		chans: make(map[chan *Notification]struct{}),
	}
}

func (n *Notifier) Subscribe() chan *Notification {
	ch := make(chan *Notification)

	n.mutex.Lock()
	n.chans[ch] = struct{}{}
	n.mutex.Unlock()

	return ch
}

func (n *Notifier) Unsubscribe(ch chan *Notification) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	delete(n.chans, ch)

	return nil
}

func (n *Notifier) Notify(notification *Notification) error {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for ch := range n.chans {
		ch <- notification
	}

	return nil
}

type txWithBlock struct {
	Tx    *blockchain.Tx
	Block *blockchain.Block
}

type rpcServer struct {
	server     *Server
	grpcServer *grpc.Server

	acceptedTxs chan txWithBlock

	notifier *Notifier

	acceptedTxsListenersMutex sync.Mutex
	acceptedTxsListeners      []chan txWithBlock

	quit chan struct{}
}

func (s *rpcServer) OnPushedBlock(block *blockchain.Block) error {
	for _, tx := range block.Txs {
		s.acceptedTxs <- txWithBlock{tx, block}
	}

	return nil
}

func (s *rpcServer) OnPoppedBlock(block *blockchain.Block) error {
	return nil
}

func (s *rpcServer) HandleAcceptedTx(tx *blockchain.Tx) error {
	s.acceptedTxs <- txWithBlock{tx, nil}

	return nil
}

func newRpcServer(server *Server) *rpcServer {
	return &rpcServer{
		server:      server,
		acceptedTxs: make(chan txWithBlock),
		notifier:    NewNotifier(),
		quit:        make(chan struct{}),
	}
}

func (s *rpcServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("rpcport")))
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer()

	pb.RegisterNodeServer(s.grpcServer, s)

	log.Infof("rpc server listening on :%d", viper.GetInt("rpcport"))

	if err := s.grpcServer.Serve(listener); err != nil {
		return err
	}

	return nil
}

func (s *rpcServer) Stop() error {
	log.Debug("rpc server shutting down")
	defer log.Debug("rpc server shutdown complete")

	s.grpcServer.GracefulStop()

	close(s.quit)

	return nil
}

func (s *rpcServer) GetInfo(ctx context.Context, in *pb.GetInfoRequest) (*pb.GetInfoReply, error) {
	return nil, nil
}

func (s *rpcServer) GetBlockByHash(ctx context.Context, in *pb.GetBlockByHashRequest) (*pb.GetBlockByHashReply, error) {
	return nil, nil
}

func (s *rpcServer) GetBlockTemplate(in *pb.GetBlockTemplateRequest, stream pb.Node_GetBlockTemplateServer) error {
	return nil
}

func (s *rpcServer) GetTxByHash(ctx context.Context, in *pb.GetTxByHashRequest) (*pb.GetTxByHashReply, error) {
	return nil, nil
}

func (s *rpcServer) PublishRawBlock(stream pb.Node_PublishRawBlockServer) error {
	return nil
}

func (s *rpcServer) PublishRawTx(stream pb.Node_PublishRawTxServer) error {
	return nil
}
