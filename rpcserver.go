package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/EnsicoinDevs/eccd/blockchain"
	"github.com/EnsicoinDevs/eccd/network"
	pb "github.com/EnsicoinDevs/eccd/rpc"
	"github.com/EnsicoinDevs/eccd/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
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

	go s.startAcceptedTxsHandler()

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

func (s *rpcServer) startAcceptedTxsHandler() error {
	for {
		select {
		case tx := <-s.acceptedTxs:
			go func() {
				s.acceptedTxsListenersMutex.Lock()
				for _, ch := range s.acceptedTxsListeners {
					ch <- tx
				}
				s.acceptedTxsListenersMutex.Unlock()
			}()
		case <-s.quit:
			s.acceptedTxsListenersMutex.Lock()
			for _, ch := range s.acceptedTxsListeners {
				close(ch)
			}
			s.acceptedTxsListenersMutex.Unlock()
			return nil
		}
	}
}

func (s *rpcServer) GetBlockTemplate(in *pb.GetBlockTemplateRequest, stream pb.Node_GetBlockTemplateServer) error {
	ch := s.notifier.Subscribe()

	go s.notifier.Notify(&Notification{
		Type:  NOTIFICATION_PUSHED_BLOCK,
		Block: &blockchain.GenesisBlock,
	})

	for notification := range ch {
		switch notification.Type {
		case NOTIFICATION_PUSHED_BLOCK:
			fallthrough
		case NOTIFICATION_POPPED_BLOCK:
			blockTemplate := &pb.BlockTemplate{
				Version:   notification.Block.Msg.Header.Version,
				Flags:     notification.Block.Msg.Header.Flags,
				PrevBlock: notification.Block.Hash()[:],
				Timestamp: uint64(time.Now().Unix()),
				Height:    notification.Block.Msg.Header.Height + 1,
			}

			blockTemplate.Target, _ = s.server.blockchain.CalcNextBlockDifficulty(notification.Block, &blockchain.Block{
				Msg: &network.BlockMessage{
					Header: &network.BlockHeader{
						Height: notification.Block.Msg.Header.Height,
					},
				},
			})

			if err := stream.Send(&pb.GetBlockTemplateReply{
				Template: blockTemplate,
			}); err != nil {
				s.notifier.Unsubscribe(ch)
				return nil
			}
		}
	}

	return nil
}

func (s *rpcServer) PublishBlock(ctx context.Context, in *pb.PublishBlockRequest) (*pb.PublishBlockReply, error) {
	return nil, nil
}

func (s *rpcServer) GetBlock(ctx context.Context, in *pb.GetBlockRequest) (*pb.GetBlockReply, error) {
	hash, err := hex.DecodeString(in.GetHash())
	if err != nil {
		return nil, err
	}

	block, err := s.server.blockchain.FindBlockByHash(utils.NewHash(hash))
	if err != nil {
		return nil, err
	}

	hashPrevBlock := hex.EncodeToString(block.Msg.Header.HashPrevBlock[:])
	hashMerkleRoot := hex.EncodeToString(block.Msg.Header.HashMerkleRoot[:])

	reply := &pb.GetBlockReply{
		Block: &pb.Block{
			Hash:           in.GetHash(),
			Version:        block.Msg.Header.Version,
			Flags:          block.Msg.Header.Flags,
			HashPrevBlock:  hashPrevBlock,
			HashMerkleRoot: hashMerkleRoot,
			Timestamp:      block.Msg.Header.Timestamp.Unix(),
			Height:         block.Msg.Header.Height,
			Bits:           block.Msg.Header.Bits,
			Nonce:          block.Msg.Header.Nonce,
		},
	}

	for _, tx := range block.Txs {
		txHash := tx.Msg.Hash()

		replyTx := &pb.Tx{
			Hash:    hex.EncodeToString(txHash[:]),
			Version: tx.Msg.Version,
			Flags:   tx.Msg.Flags,
		}

		for _, input := range tx.Msg.Inputs {
			hashPreviousOutput := hex.EncodeToString(input.PreviousOutput.Hash[:])

			replyTx.Inputs = append(replyTx.Inputs, &pb.Input{
				PreviousOutput: &pb.Outpoint{
					Hash:  hashPreviousOutput,
					Index: input.PreviousOutput.Index,
				},
				Script: input.Script,
			})
		}

		for _, output := range tx.Msg.Outputs {
			replyTx.Outputs = append(replyTx.Outputs, &pb.Output{
				Value:  output.Value,
				Script: output.Script,
			})
		}

		reply.Block.Txs = append(reply.Block.Txs, replyTx)
	}

	return reply, nil
}

func (s *rpcServer) GetBestBlockHash(ctx context.Context, in *pb.GetBestBlockHashRequest) (*pb.GetBestBlockHashReply, error) {
	block, err := s.server.blockchain.FindBestBlock()
	if err != nil {
		return nil, err
	}

	return &pb.GetBestBlockHashReply{
		Hash: hex.EncodeToString(block.Hash()[:]),
	}, nil
}

func (s *rpcServer) PublishTx(ctx context.Context, in *pb.PublishTxRequest) (*pb.PublishTxReply, error) {
	tx := network.NewTxMessage()
	buf := bytes.NewBuffer(in.GetTx())
	err := tx.Decode(buf)
	if err != nil {
		return nil, err
	}

	s.server.ProcessTx(tx)

	return &pb.PublishTxReply{}, nil
}

func (s *rpcServer) ListenIncomingTxs(in *pb.ListenIncomingTxsRequest, stream pb.Node_ListenIncomingTxsServer) error {
	ch := make(chan txWithBlock)
	s.acceptedTxsListenersMutex.Lock()
	s.acceptedTxsListeners = append(s.acceptedTxsListeners, ch)
	s.acceptedTxsListenersMutex.Unlock()
	log.Warn("added")
	defer log.Warn("good bye")

	for txWithBlock := range ch {
		var block *pb.Block

		if txWithBlock.Block != nil {
			block = &pb.Block{
				Hash:           hex.EncodeToString(txWithBlock.Block.Hash()[:]),
				Version:        txWithBlock.Block.Msg.Header.Version,
				Flags:          txWithBlock.Block.Msg.Header.Flags,
				HashPrevBlock:  hex.EncodeToString(txWithBlock.Block.Msg.Header.HashPrevBlock[:]),
				HashMerkleRoot: hex.EncodeToString(txWithBlock.Block.Msg.Header.HashMerkleRoot[:]),
				Timestamp:      txWithBlock.Block.Msg.Header.Timestamp.Unix(),
				Height:         txWithBlock.Block.Msg.Header.Height,
				Bits:           txWithBlock.Block.Msg.Header.Bits,
				Nonce:          txWithBlock.Block.Msg.Header.Nonce,
			}
		}

		tx := &pb.Tx{
			Hash:    hex.EncodeToString(txWithBlock.Tx.Hash()[:]),
			Version: txWithBlock.Tx.Msg.Version,
			Flags:   txWithBlock.Tx.Msg.Flags,
		}

		for _, input := range txWithBlock.Tx.Msg.Inputs {
			tx.Inputs = append(tx.Inputs, &pb.Input{
				PreviousOutput: &pb.Outpoint{
					Hash:  hex.EncodeToString(input.PreviousOutput.Hash[:]),
					Index: input.PreviousOutput.Index,
				},
				Script: input.Script,
			})
		}

		for _, output := range txWithBlock.Tx.Msg.Outputs {
			tx.Outputs = append(tx.Outputs, &pb.Output{
				Value:  output.Value,
				Script: output.Script,
			})
		}

		if err := stream.Send(&pb.TxWithBlock{
			Tx:    tx,
			Block: block,
		}); err != nil {
			s.removeAcceptedTxsListener(ch)

			return err
		}
	}

	return s.removeAcceptedTxsListener(ch)
}

func (s *rpcServer) removeAcceptedTxsListener(ch chan txWithBlock) error {
	log.Warn("removed")

	s.acceptedTxsListenersMutex.Lock()
	defer s.acceptedTxsListenersMutex.Unlock()

	for i, och := range s.acceptedTxsListeners {
		if och == ch {
			s.acceptedTxsListeners = append(s.acceptedTxsListeners[:i], s.acceptedTxsListeners[i+1:]...)
			return nil
		}
	}

	return nil
}
