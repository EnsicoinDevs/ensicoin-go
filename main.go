package main

import (
	"bufio"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strings"

	"github.com/EnsicoinDevs/ensicoincoin/blockchain"
	"github.com/EnsicoinDevs/ensicoincoin/consensus"
	"github.com/EnsicoinDevs/ensicoincoin/mempool"
)

var peerPort int

func init() {
	flag.IntVar(&peerPort, "port", consensus.INGOING_PORT, "The port of the node.")
}

func main() {
	log.SetLevel(log.DebugLevel)

	log.Info("ENSICOINCOIN is starting")

	flag.Parse()

	blockchain := blockchain.NewBlockchain()
	blockchain.Load()

	mempool := mempool.NewMempool(&mempool.Config{
		FetchUtxos: blockchain.FetchUtxos,
	})

	server := NewServer(blockchain, mempool)

	go server.Start()

	log.Info("ENSICOINCOIN is now running")

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">>> ")
		rawCommand, _ := reader.ReadString('\n')

		command := strings.Split(rawCommand[:len(rawCommand)-1], " ")

		if command[0] == "q" || command[0] == "quit" {
			break
		}

		switch command[0] {
		case "help":
			log.Info("quit, help, connect, show_peers, show_mempool")
		case "connect":
			address := command[1]

			conn, err := net.Dial("tcp", address)
			if err != nil {
				log.WithError(err).WithField("address", address).Error("error dialing this address")
				break
			}

			server.RegisterOutgoingPeer(conn)
		}
	}

	server.Stop()

	log.Info("Good bye.")
}
