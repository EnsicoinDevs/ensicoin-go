package main

import (
	"fmt"
	"github.com/EnsicoinDevs/ensicoincoin/consensus"
	"github.com/c-bata/go-prompt"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
)

var (
	stop   func()
	server *Server
)

func init() {
	pflag.IntP("port", "p", consensus.INGOING_PORT, "listening port")
	pflag.StringP("cfgfile", "c", "", "config file")
	pflag.StringP("datadir", "d", "", "data dir")
	pflag.StringP("token", "t", "", "a discord token")
	pflag.BoolP("mining", "m", false, "enable mining")
	pflag.BoolP("interactive", "i", false, "enable prompt")
	pflag.BoolP("pprof", "P", false, "enable pprof")

	viper.BindPFlags(pflag.CommandLine)

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if viper.GetString("cfgfile") != "" {
		viper.SetConfigFile(viper.GetString("cfgfile"))
	} else {
		viper.SetConfigName("ensicoincoin")

		home, err := homedir.Dir()
		if err != nil {
			log.WithError(err).Fatal("damned")
		}

		viper.AddConfigPath(home + "/.config/ensicoincoin/")
		viper.AddConfigPath(home)
	}

	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Error("can't read config")
	}
}

var rootCmd = &cobra.Command{
	Use:   "ensicoincoin",
	Short: "EnsiCoinCoin is a questionable implementation of the Ensicoin protocol",
	Long:  `EnsiCoinCoin is a questionable implementation of the Ensicoin protocol. It is a demon that allows you to synchronize with the blockchain.`,
	Run: func(cmd *cobra.Command, args []string) {
		pflag.Parse()

		launch()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.WithError(err).Fatal("damned")
	}
}

func executor(in string) {
	c := strings.Split(in, " ")

	switch c[0] {
	case "connect":
		if len(c) < 2 {
			fmt.Println("Please specify the node address.")
			return
		}

		if err := server.ConnectTo(c[1]); err != nil {
			fmt.Println("Error connecting to " + c[1] + ": " + err.Error())
			return
		}

		fmt.Println("Connected.")
	case "exit":
		stop()
		fmt.Println("Good bye.")
		os.Exit(0)
	default:
		fmt.Println("Command not found. Type help to get the list of commands.")
	}
}

func completer(in prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "connect", Description: "Connect to an ensicoincoin node"},
		{Text: "exit", Description: "Exit the node"},
	}

	return prompt.FilterHasPrefix(s, in.GetWordBeforeCursor(), true)
}

func launch() {
	log.SetLevel(log.DebugLevel)

	if viper.GetBool("pprof") {
		log.Debug("?")

		go func() {
			log.Debug("pprof")
			log.Debug(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	log.Info("EnsiCoinCoin is starting")

	server = NewServer()

	go server.Start()

	if viper.GetString("token") != "" {
		startDiscordBootstraping(server)
	}

	log.Info("EnsiCoinCoin is running")

	if !viper.GetBool("interactive") {
		ch := make(chan bool)
		<-ch
	}

	stop = func() {
		err := server.Stop()
		if err != nil {
			log.WithError(err).Error("error stopping the server")
		}
	}

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(">>> "),
	)

	p.Run()

	stop()

	log.Info("Good bye.")
}
