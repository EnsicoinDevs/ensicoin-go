package main

import (
	"encoding/hex"
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	pb "github.com/EnsicoinDevs/ensicoincoin/rpc"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
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

	hashPrevBlock := hex.EncodeToString(block.Msg.Header.HashPrevBlock[:])
	hashMerkleRoot := hex.EncodeToString(block.Msg.Header.HashMerkleRoot[:])

	reply := &pb.GetBlockByHashReply{
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
	}

	for _, tx := range block.Txs {
		replyTx := &pb.Tx{
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
