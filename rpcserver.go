package main

import (
	"bytes"
	"fmt"
	"github.com/EnsicoinDevs/eccd/blockchain"
	"github.com/EnsicoinDevs/eccd/network"
	pb "github.com/EnsicoinDevs/eccd/rpc"
	"github.com/EnsicoinDevs/eccd/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"strconv"
	"sync"
	"time"
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

type rpcServer struct {
	server     *Server
	grpcServer *grpc.Server

	notifier *Notifier

	quit chan struct{}
}

func (s *rpcServer) OnPushedBlock(block *blockchain.Block) error {
	s.notifier.Notify(&Notification{
		Type:  NOTIFICATION_PUSHED_BLOCK,
		Block: block,
	})

	return nil
}

func (s *rpcServer) OnPoppedBlock(block *blockchain.Block) error {
	s.notifier.Notify(&Notification{
		Type:  NOTIFICATION_POPPED_BLOCK,
		Block: block,
	})

	return nil
}

func (s *rpcServer) HandleAcceptedTx(tx *blockchain.Tx) error {
	return nil
}

func newRpcServer(server *Server) *rpcServer {
	return &rpcServer{
		server:   server,
		notifier: NewNotifier(),
		quit:     make(chan struct{}),
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

	close(s.quit)
	s.grpcServer.GracefulStop()

	return nil
}

func (s *rpcServer) GetInfo(ctx context.Context, in *pb.GetInfoRequest) (*pb.GetInfoReply, error) {
	bestBlock, err := s.server.blockchain.FindBestBlock()
	if err != nil {
		return nil, fmt.Errorf("error finding the best block")
	}

	return &pb.GetInfoReply{
		Implementation:  "eccd 0.0.0",
		ProtocolVersion: 0,
		BestBlockHash:   bestBlock.Hash().Bytes(),
	}, nil
}

func (s *rpcServer) GetBlockByHash(ctx context.Context, in *pb.GetBlockByHashRequest) (*pb.GetBlockByHashReply, error) {
	block, err := s.server.blockchain.FindBlockByHash(utils.NewHash(in.GetHash()))
	if err != nil {
		return nil, fmt.Errorf("error finding the block")
	}

	return &pb.GetBlockByHashReply{
		Block: BlockMessageToRpcBlock(block.Msg),
	}, nil
}

func (s *rpcServer) GetTxByHash(ctx context.Context, in *pb.GetTxByHashRequest) (*pb.GetTxByHashReply, error) {
	tx := s.server.mempool.FindTxByHash(utils.NewHash(in.GetHash()))

	return &pb.GetTxByHashReply{
		Tx: TxMessageToRpcTx(tx.Msg),
	}, nil
}

func (s *rpcServer) GetBlockTemplate(in *pb.GetBlockTemplateRequest, stream pb.Node_GetBlockTemplateServer) error {
	ch := s.notifier.Subscribe()

	bestBlock, err := s.server.blockchain.FindBestBlock()
	if err != nil {
		return status.Errorf(codes.Internal, "internal error")
	}

	go func() {
		ch <- &Notification{
			Type:  NOTIFICATION_PUSHED_BLOCK,
			Block: bestBlock,
		}
	}()

	for {
		select {
		case notification := <-ch:
			switch notification.Type {
			case NOTIFICATION_PUSHED_BLOCK:
				fallthrough
			case NOTIFICATION_POPPED_BLOCK:
				timestamp := time.Now()

				nextTarget, err := s.server.blockchain.CalcNextBlockDifficulty(notification.Block, blockchain.NewBlockFromBlockMessage(&network.BlockMessage{
					Header: &network.BlockHeader{
						Height:    notification.Block.Msg.Header.Height + 1,
						Timestamp: timestamp,
					},
				}))
				if err != nil {
					return status.Errorf(codes.Internal, "internal error")
				}

				reply := &pb.GetBlockTemplateReply{
					BlockTemplate: &pb.BlockTemplate{
						Version:   notification.Block.Msg.Header.Version,
						Flags:     notification.Block.Msg.Header.Flags,
						PrevBlock: notification.Block.Hash().Bytes(),
						Timestamp: uint64(timestamp.Unix()),
						Height:    notification.Block.Msg.Header.Height + 1,
						Target:    utils.BigToHash(nextTarget).Bytes(),
					},
				}

				txs := s.server.mempool.FetchTxs()

				for _, tx := range txs {
					reply.Txs = append(reply.Txs, TxMessageToRpcTx(tx.Msg))
				}

				if err := stream.Send(reply); err != nil {
					return nil
				}
			}
		case <-s.quit:
			return nil
		}
	}
}

func (s *rpcServer) PublishRawBlock(ctx context.Context, in *pb.PublishRawBlockRequest) (*pb.PublishRawBlockReply, error) {
	blockMsg := network.NewBlockMessage()

	err := blockMsg.Decode(bytes.NewReader(in.GetRawBlock()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error decoding the raw block")
	}

	go s.server.ProcessBlock(blockMsg)

	return &pb.PublishRawBlockReply{}, nil
}

func (s *rpcServer) PublishRawTx(ctx context.Context, in *pb.PublishRawTxRequest) (*pb.PublishRawTxReply, error) {
	txMsg := network.NewTxMessage()

	err := txMsg.Decode(bytes.NewReader(in.GetRawTx()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error decoding the raw tx")
	}

	go s.server.ProcessTx(txMsg)

	return &pb.PublishRawTxReply{}, nil
}

func (s *rpcServer) ConnectPeer(ctx context.Context, in *pb.ConnectPeerRequest) (*pb.ConnectPeerReply, error) {
	if in.GetPeer() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "peer field is required")
	}

	if in.GetPeer().GetAddress() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "address field is required")
	}

	address := in.GetPeer().GetAddress()

	parsedAddress := net.JoinHostPort(address.GetIp(), strconv.Itoa(int(address.GetPort())))

	err := s.server.ConnectTo(parsedAddress)
	if err != nil {
		log.WithError(err).Warn("error connecting to ", parsedAddress)
	}

	return &pb.ConnectPeerReply{}, nil
}

func (s *rpcServer) DisconnectPeer(ctx context.Context, in *pb.DisconnectPeerRequest) (*pb.DisconnectPeerReply, error) {
	if in.GetPeer() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "peer field is required")
	}

	if in.GetPeer().GetAddress() == nil {
		return nil, status.Errorf(codes.InvalidArgument, "address field is required")
	}

	address := in.GetPeer().GetAddress()

	parsedAddress := net.JoinHostPort(address.GetIp(), strconv.Itoa(int(address.GetPort())))

	peer, err := s.server.FindPeerByAddress(parsedAddress)
	if err != nil {
		log.WithError(err).Warn("error disconnecting from", parsedAddress)
		return &pb.DisconnectPeerReply{}, nil
	}
	if peer == nil {
		log.WithError(fmt.Errorf("peer not found")).Warn("error disconnecting from", parsedAddress)
		return &pb.DisconnectPeerReply{}, nil
	}

	err = s.server.DisconnectFrom(peer)
	if err != nil {
		log.WithError(err).Warn("error disconnecting from", parsedAddress)
	}

	return &pb.DisconnectPeerReply{}, nil
}

func BlockMessageToRpcBlock(blockMsg *network.BlockMessage) *pb.Block {
	block := &pb.Block{
		Hash:       blockMsg.Header.Hash().Bytes(),
		Version:    blockMsg.Header.Version,
		Flags:      blockMsg.Header.Flags,
		PrevBlock:  blockMsg.Header.HashPrevBlock.Bytes(),
		MerkleRoot: blockMsg.Header.HashMerkleRoot.Bytes(),
		Timestamp:  uint64(blockMsg.Header.Timestamp.Unix()),
		Height:     blockMsg.Header.Height,
		Target:     blockMsg.Header.Target.Bytes(),
	}

	for _, tx := range blockMsg.Txs {
		block.Txs = append(block.Txs, TxMessageToRpcTx(tx))
	}

	return block
}

func TxMessageToRpcTx(txMsg *network.TxMessage) *pb.Tx {
	tx := &pb.Tx{
		Hash:    txMsg.Hash().Bytes(),
		Version: txMsg.Version,
		Flags:   txMsg.Flags,
	}

	for _, input := range txMsg.Inputs {
		tx.Inputs = append(tx.Inputs, &pb.TxInput{
			PreviousOutput: &pb.Outpoint{
				Hash:  input.PreviousOutput.Hash.Bytes(),
				Index: input.PreviousOutput.Index,
			},
			Script: input.Script,
		})
	}

	for _, output := range txMsg.Outputs {
		tx.Outputs = append(tx.Outputs, &pb.TxOutput{
			Value:  output.Value,
			Script: output.Script,
		})
	}

	return tx
}
