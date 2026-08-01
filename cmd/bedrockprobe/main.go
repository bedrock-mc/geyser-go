package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func main() {
	address := flag.String("address", "127.0.0.1:19132", "Bedrock/RakNet bridge address")
	username := flag.String("username", "GeyserProbe", "offline-mode Bedrock username")
	readFor := flag.Duration("read-for", 0, "after spawn, read and print Bedrock packets for this duration")
	sendAuthInput := flag.Bool("send-auth-input", false, "send one PlayerAuthInput movement packet after spawn")
	sendBlockAction := flag.Bool("send-block-action", false, "include one block-break action in the auth-input packet")
	sendItemUse := flag.Bool("send-item-use", false, "include one click-air item interaction in the auth-input packet")
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
	if *sendAuthInput {
		input := protocol.NewBitset(packet.PlayerAuthInputBitsetSize)
		input.Set(packet.InputFlagVerticalCollision)
		auth := &packet.PlayerAuthInput{
			Position:  conn.GameData().PlayerPosition,
			Yaw:       conn.GameData().Yaw,
			Pitch:     conn.GameData().Pitch,
			HeadYaw:   conn.GameData().Yaw,
			InputData: input,
		}
		if *sendBlockAction {
			input.Set(packet.InputFlagPerformBlockActions)
			auth.BlockActions = []protocol.PlayerBlockAction{{
				Action:   protocol.PlayerActionStartBreak,
				BlockPos: protocol.BlockPos{int32(auth.Position.X()), int32(auth.Position.Y()) - 2, int32(auth.Position.Z())},
				Face:     1,
			}}
		}
		if *sendItemUse {
			input.Set(packet.InputFlagPerformItemInteraction)
			auth.InteractYaw = auth.Yaw
			auth.InteractPitch = auth.Pitch
			auth.ItemInteractionData = protocol.UseItemTransactionData{
				ActionType: protocol.UseItemActionClickAir,
				Position:   auth.Position,
			}
		}
		if err := conn.WritePacket(auth); err != nil {
			panic(err)
		}
	}
	_ = conn.WritePacket(&packet.Text{TextType: packet.TextTypeChat, Message: "geyser-go probe"})
	fmt.Printf("Bedrock spawn succeeded: protocol=%d version=%s items=%d world=%q entity=%d\n", conn.Proto().ID(), conn.Proto().Ver(), len(conn.GameData().Items), conn.GameData().WorldName, conn.GameData().EntityRuntimeID)
	if *readFor <= 0 {
		return
	}
	deadline := time.Now().Add(*readFor)
	if err := conn.SetReadDeadline(deadline); err != nil {
		panic(err)
	}
	for {
		batch, err := conn.ReadBatch()
		if err != nil {
			fmt.Printf("Bedrock read finished: %v\n", err)
			return
		}
		for _, pk := range batch {
			switch pk := pk.(type) {
			case *packet.Text:
				fmt.Printf("Bedrock text: type=%d source=%q message=%q\n", pk.TextType, pk.SourceName, pk.Message)
			default:
				fmt.Printf("Bedrock packet: %T\n", pk)
			}
		}
	}
}
