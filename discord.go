package main

import (
	"github.com/EnsicoinDevs/ensicoincoin/consensus"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strconv"
	"strings"
)

var dg *discordgo.Session
var daddress string

func startDiscordBootstraping(server *Server) {
	var err error
	dg, err = discordgo.New(viper.GetString("token"))
	if err != nil {
		log.WithError(err).Error("error creating Discord session")
		return
	}

	dg.AddHandler(discordMessageCreate)

	err = dg.Open()
	if err != nil {
		log.WithError(err).Error("error opening Discord connection")
	}

	daddress = server.listener.Addr().String()

	discordHelloWorld(dg)
}

func closeDiscordBootstraping() {
	dg.Close()
}

func discordMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		log.WithError(err).Error("error fetching the discord channel of a message")
		return
	}

	if channel.Name != "ensicoin" {
		return
	}

	networkMagicNumber := strconv.Itoa(consensus.NETWORK_MAGIC_NUMBER)

	if !strings.HasPrefix(m.Content, networkMagicNumber) {
		return
	}

	msg := strings.Split(strings.TrimPrefix(m.Content, networkMagicNumber+" "), " ")

	if msg[0] == "hello" && msg[2] == "please_connect" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "olala omw")
	}
}

func discordHelloWorld(s *discordgo.Session) {
	for _, guild := range s.State.Guilds {
		for _, channel := range guild.Channels {
			if channel.Name == "ensicoin" {
				log.Info("found a #ensicoin channel on discord")
				_, _ = s.ChannelMessageSend(channel.ID, strconv.Itoa(consensus.NETWORK_MAGIC_NUMBER)+" hello_world_i_m_an_ensicoin_peer "+daddress)
				return
			}
		}
	}
}
