package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
)

var interruptSignals = []os.Signal{os.Interrupt}

func newInterruptListener() <-chan struct{} {
	ch := make(chan struct{})

	go func() {
		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, interruptSignals...)

		select {
		case sig := <-interruptChannel:
			log.WithField("signal", sig).Info("shutting down")
		}

		close(ch)

		for {
			select {
			case sig := <-interruptChannel:
				log.WithField("signal", sig).Info("already shutting down")
			}
		}
	}()

	return ch
}
