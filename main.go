package main

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"strings"

	"github.com/EnsicoinDevs/ensicoin-go/blockchain"
	"github.com/EnsicoinDevs/ensicoin-go/mempool"
)

func main() {
	log.Info("ENSICOIN-GO is starting")

	blockchain := blockchain.NewBlockchain()
	blockchain.Load()

	mempool := mempool.NewMempool()

	server := NewServer(blockchain, mempool)

	go server.Start()

	log.Info("ENSICOIN-GO is now running")

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
