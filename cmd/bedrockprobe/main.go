package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func main() {
	address := flag.String("address", "127.0.0.1:19132", "Bedrock/RakNet bridge address")
	username := flag.String("username", "GeyserProbe", "offline-mode Bedrock username")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dialer := minecraft.Dialer{
		IdentityData: login.IdentityData{
			Identity: uuid.New().String(), DisplayName: *username,
		},
		EnableBatchReading: true,
	}
	conn, err := dialer.DialContext(ctx, "raknet", *address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if err := conn.DoSpawnContext(ctx); err != nil {
		panic(err)
	}
	_ = conn.WritePacket(&packet.Text{TextType: packet.TextTypeChat, Message: "geyser-go probe"})
	fmt.Printf("Bedrock spawn succeeded: protocol=%d version=%s items=%d world=%q entity=%d\n", conn.Proto().ID(), conn.Proto().Ver(), len(conn.GameData().Items), conn.GameData().WorldName, conn.GameData().EntityRuntimeID)
}
