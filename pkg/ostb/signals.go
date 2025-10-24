package ostb

import (
	"os"
	"os/signal"
)

func WaitForStopSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
