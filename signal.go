package main

import (
	"os"
	"os/signal"
	"syscall"
)

func init() {
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
		<-signalChan

		End(false, false)
	}()
}
