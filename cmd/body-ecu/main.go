package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"example.com/qemu-arm-can-test/internal/protocol"
	"example.com/qemu-arm-can-test/internal/socketcan"
)

func main() {
	interfaceName := flag.String("interface", "can0", "SocketCAN interface")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bus, err := socketcan.Open(*interfaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "body-ecu: %v\n", err)
		os.Exit(2)
	}
	defer bus.Close()

	fmt.Printf("ECU_READY interface=%s\n", *interfaceName)
	for {
		request, err := bus.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			fmt.Fprintf(os.Stderr, "body-ecu: receive: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("RX %s (%s)\n", request, protocol.Describe(request))

		response, ok := protocol.HandleHeadlightRequest(request)
		if !ok {
			fmt.Printf("IGNORED %s\n", request)
			continue
		}
		if err := bus.Send(response); err != nil {
			fmt.Fprintf(os.Stderr, "body-ecu: send: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("TX %s\n", response)
	}
}
