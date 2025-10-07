package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/example/ebyte_can_ethernet_to_slcan/cmd/bridge/internal/app"
)

func main() {
	var (
		ebyteHost      = flag.String("ebyte-host", "127.0.0.1", "Hostname or IP address of the EByte CAN-to-Ethernet adapter")
		ebytePort      = flag.Int("ebyte-port", 4001, "TCP port of the EByte CAN-to-Ethernet adapter")
		listenHost     = flag.String("listen-host", "0.0.0.0", "Host address for the CANserver UDP server")
		listenPort     = flag.Int("listen-port", 1338, "Port for the CANserver UDP server")
		reconnectDelay = flag.Duration("reconnect-delay", 2*time.Second, "Delay before retrying the connection to the adapter")
		logLevel       = flag.String("log-level", "info", "Log level (debug|info|warn|error)")
	)

	flag.Parse()

	cfg := app.Config{
		EByteAddress:   fmt.Sprintf("%s:%d", *ebyteHost, *ebytePort),
		ListenAddress:  fmt.Sprintf("%s:%d", *listenHost, *listenPort),
		ReconnectDelay: *reconnectDelay,
		LogLevel:       *logLevel,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	bridge, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialise bridge: %v", err)
	}

	if err := bridge.Run(ctx); err != nil {
		log.Fatalf("bridge terminated: %v", err)
	}
}
