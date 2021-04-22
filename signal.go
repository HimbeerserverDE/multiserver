package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		<-signalChan

		log.Print("Caught SIGINT or SIGTERM, shutting down")

		End(false, false)
	}()
}
