package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/network"
	pb "github.com/EnsicoinDevs/ensicoincoin/rpc"
	"github.com/EnsicoinDevs/ensicoincoin/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
)

type rpcServer struct {
	blockchain *blockchain.Blockchain
	server     *Server
}

func newRpcServer(blockchain *blockchain.Blockchain, server *Server) *rpcServer {
	return &rpcServer{
		blockchain: blockchain,
		server:     server,
	}
}

func (s *rpcServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", 4225))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()

	pb.RegisterBlockchainServer(grpcServer, s)
	pb.RegisterTxsServer(grpcServer, s)

	if err := grpcServer.Serve(listener); err != nil {
		return err
	}

	return nil
}

func (s *rpcServer) GetBlock(ctx context.Context, in *pb.GetBlockRequest) (*pb.GetBlockReply, error) {
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
	block, err := s.blockchain.FindLongestChain()
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
