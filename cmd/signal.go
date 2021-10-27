package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sch := make(chan os.Signal)
	signal.Notify(sch)
	defer signal.Stop(sch)

	for {
		select {
		case s := <-sch:
			switch s {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL:
				log.Println("exit", s)
				return
			default:
				log.Println("signal", s)
			}
		}
	}
}
