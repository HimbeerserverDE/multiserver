package multiserver

import (
	"os"
	"os/signal"
)

func init() {
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		<-signalChan

		End(false, false)
	}()
}
