package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bedrock-mc/geyser-go/bridge"
)

func main() {
	bedrockListen := flag.String("bedrock-listen", "0.0.0.0:19132", "Bedrock/RakNet listen address")
	javaAddress := flag.String("java-address", "", "Java server address, for example 127.0.0.1:25565")
	noAuth := flag.Bool("no-auth", false, "disable Xbox Live authentication for local trusted testing")
	snappy := flag.Bool("snappy", false, "opt into Snappy Bedrock compression")
	flag.Parse()

	if *javaAddress == "" {
		fmt.Fprintln(os.Stderr, "-java-address is required until a translator implementation is configured")
		os.Exit(2)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	server, err := bridge.New(bridge.Config{
		BedrockListen:          *bedrockListen,
		JavaAddress:            *javaAddress,
		AuthenticationDisabled: *noAuth,
		Snappy:                 *snappy,
		Logger:                 logger,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := server.ListenAndServe(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
