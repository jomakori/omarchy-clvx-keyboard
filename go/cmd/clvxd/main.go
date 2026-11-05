// Command clvxd is the CLVX keyboard daemon: heartbeat session + Unix socket API.
//
//	clvxd [--transport auto|usb|ble] [--socket path]
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"clvx/internal/daemon"
)

func main() {
	transport := flag.String("transport", "auto", "auto, usb or ble")
	socket := flag.String("socket", daemon.SocketPath(), "unix socket path")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d := daemon.New(*transport)
	go d.Run(ctx)
	log.Printf("socket %s (transport %s)", *socket, *transport)
	if err := daemon.Serve(ctx, d, *socket); err != nil {
		log.Fatal(err)
	}
}
