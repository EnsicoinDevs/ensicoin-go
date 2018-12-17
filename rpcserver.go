package main

import (
	"encoding/hex"
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	pb "github.com/EnsicoinDevs/ensicoincoin/rpc"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
)

type rpcServer struct {
	blockchain *blockchain.Blockchain
}

func newRpcServer(blockchain *blockchain.Blockchain) *rpcServer {
	return &rpcServer{
		blockchain: blockchain,
	}
}

func (s *rpcServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", 4225))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	pb.RegisterBlockchainServer(grpcServer, s)

	if err := grpcServer.Serve(listener); err != nil {
		return err
	}

	return nil
}

func (s *rpcServer) GetBlockByHash(ctx context.Context, in *pb.GetBlockByHashRequest) (*pb.GetBlockByHashReply, error) {
	hash, err := hex.DecodeString(in.GetHash())
	if err != nil {
		return nil, err
	}

	block, err := s.blockchain.FindBlockByHash(utils.NewHash(hash))
	if err != nil {
		return nil, err
	}

	log.WithField("hash", in.GetHash()).WithField("hash2", hash).Info("hashes")

	log.WithField("block", block).Info("block")

	hashPrevBlock := hex.EncodeToString(block.Msg.Header.HashPrevBlock[:])
	hashMerkleRoot := hex.EncodeToString(block.Msg.Header.HashMerkleRoot[:])

	return &pb.GetBlockByHashReply{
		Block: &pb.Block{
			Version:        block.Msg.Header.Version,
			Flags:          block.Msg.Header.Flags,
			Hash:           in.GetHash(),
			HashPrevBlock:  hashPrevBlock,
			HashMerkleRoot: hashMerkleRoot,
			Timestamp:      block.Msg.Header.Timestamp.Unix(),
			Height:         block.Msg.Header.Height,
			Bits:           block.Msg.Header.Bits,
			Nonce:          block.Msg.Header.Nonce,
		},
	}, nil
}

func (s *rpcServer) GetBlockHashAtHeight(ctx context.Context, in *pb.GetBlockHashAtHeightRequest) (*pb.GetBlockHashAtHeightReply, error) {
	hash, err := s.blockchain.GetBlockHashAtHeight(in.GetHeight())
	if err != nil {
		return nil, err
	}

	return &pb.GetBlockHashAtHeightReply{
		Hash: hex.EncodeToString(hash[:]),
	}, nil
}

func (s *rpcServer) GetMainChain(ctx context.Context, in *pb.GetMainChainRequest) (*pb.GetMainChainReply, error) {
	return nil, nil
}

func (s *rpcServer) GetMainChainHeight(ctx context.Context, in *pb.GetMainChainHeightRequest) (*pb.GetMainChainHeightReply, error) {
	block, err := s.blockchain.FindLongestChain()
	if err != nil {
		return nil, err
	}

	return &pb.GetMainChainHeightReply{
		Height: block.Msg.Header.Height,
	}, nil
}
