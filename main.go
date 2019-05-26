package main

import (
	"fmt"
	"github.com/EnsicoinDevs/eccd/consensus"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

var (
	server *Server
)

func init() {
	pflag.StringP("level", "l", "info", "log level (trace, debug, info, error)")
	pflag.IntP("port", "p", consensus.INGOING_PORT, "listening port")
	pflag.Int("rpcport", consensus.INGOING_PORT+1, "rpc listening port")
	pflag.StringP("cfgdir", "c", getAppConfigDir(), "config directory")
	pflag.StringP("datadir", "d", getAppDataDir(), "data directory")
	pflag.String("cpuprofile", "", "write cpu profile to file")

	viper.BindPFlags(pflag.CommandLine)

	cobra.OnInitialize(initConfig)
}

func generateConfig() error {
	f, err := os.Create(filepath.Join(viper.GetString("cfgdir"), "config.yaml"))
	if err != nil {
		return err
	}

	_ = f

	return nil
}

func initConfig() {
	viper.SetConfigName("config")

	viper.AddConfigPath(viper.GetString("cfgdir"))

	stat, err := os.Stat(viper.GetString("cfgdir"))
	if os.IsNotExist(err) {
		os.MkdirAll(viper.GetString("cfgdir"), os.ModePerm)
		if err = generateConfig(); err != nil {
			log.WithError(err).Fatal("fatal error generating the configuration file")
			os.Exit(1)
		}
	} else if err != nil {
		log.WithError(err).Fatal("fatal error checking if the configuration directory exists")
		os.Exit(1)
	} else if !stat.IsDir() {
		log.Fatal("the configuration directory is not a directory")
		os.Exit(1)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(*viper.ConfigFileNotFoundError); ok {
			if err = generateConfig(); err != nil {
				log.WithError(err).Fatal("fatal error generating the configuration file")
				os.Exit(1)
			}

			if err = viper.ReadInConfig(); err != nil {
				log.WithError(err).Fatal("fatal error reading the configuration")
			}
		} else {
			log.WithError(err).Fatal("fatal error reading the configuration")
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "eccd",
	Short: "EnsiCoinCoin is a questionable implementation of the Ensicoin protocol",
	Long:  `EnsiCoinCoin is a questionable implementation of the Ensicoin protocol. It is a daemon that allows you to synchronize with the blockchain.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		switch viper.GetString("level") {
		case "trace":
			log.SetLevel(log.TraceLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		default:
			fmt.Println("Error: invalid log level")
			cmd.Usage()
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := launch(); err != nil {
			os.Exit(1)
		}
	},
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func launch() error {
	interruptListener := newInterruptListener()
	defer log.Info("shutdown complete")

	if err := checkDataDir(); err != nil {
		log.WithError(err).Fatal("fatal error checking the data dir")
		os.Exit(1)
	}

	if viper.GetString("cpuprofile") != "" {
		f, err := os.Create(viper.GetString("cpuprofile"))
		if err != nil {
			log.WithError(err).Fatal("fatal error creating the cpuprofile file")
		}

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	log.Info("version 0.0.0")
	log.WithField("cfgdir", viper.GetString("cfgdir")).Info("configuration directory is")
	log.WithField("datadir", viper.GetString("datadir")).Info("data directory is")

	server = NewServer()
	go server.Start()
	defer func() {
		log.Info("gracefully shutting down")
		server.Stop()
	}()

	<-interruptListener
	return nil
}

func checkDataDir() error {
	stat, err := os.Stat(viper.GetString("datadir"))
	if os.IsNotExist(err) {
		os.MkdirAll(viper.GetString("datadir"), os.ModePerm)
	} else if err != nil {
		return err
	} else if !stat.IsDir() {
		return errors.Wrap(err, "the data directory is not a directory")
	}

	return nil
}
